package service

import (
	"context"
	"fmt"

	"github.com/mysunshines/blog-comment/internal/model"
	"github.com/mysunshines/blog-comment/internal/repository"
	"github.com/mysunshines/blog-comment/pkg/errors"
	user "github.com/mysunshines/blog-user/proto/pb"
	"github.com/mysunshines/gocommon/grpcclient"
	"github.com/mysunshines/gocommon/pool"

	"gorm.io/gorm"
)

// CommentService 评论服务接口
type CommentService interface {
	CreateComment(ctx context.Context, req *model.CreateCommentRequest) (*model.Comment, error)
	GetComment(ctx context.Context, id uint) (*model.Comment, error)
	UpdateComment(ctx context.Context, id uint, req *model.UpdateCommentRequest) (*model.Comment, error)
	DeleteComment(ctx context.Context, id uint, req *model.DeleteCommentRequest) error
	ListComments(ctx context.Context, req *model.ListCommentsRequest) ([]*model.Comment, int64, error)
	GetArticleComments(ctx context.Context, req *model.GetArticleCommentsRequest) ([]*model.Comment, int64, bool, error)
	ReplyComment(ctx context.Context, parentID uint, req *model.ReplyCommentRequest) (*model.Comment, error)
	LikeComment(ctx context.Context, commentID uint, req *model.LikeCommentRequest) (uint, error)
	GetCommentReplies(ctx context.Context, req *model.GetCommentRepliesRequest) ([]*model.Comment, int64, error)
	EnableComment(ctx context.Context, req *model.EnableCommentRequest) error
	DisableComment(ctx context.Context, req *model.DisableCommentRequest) error
}

// commentService 评论服务实现
type commentService struct {
	commentRepo     repository.CommentRepository
	commentLikeRepo repository.CommentLikeRepository
	db              *gorm.DB
}

// NewCommentService 创建评论服务
func NewCommentService(
	commentRepo repository.CommentRepository,
	commentLikeRepo repository.CommentLikeRepository,
	db *gorm.DB,
) CommentService {
	return &commentService{
		commentRepo:     commentRepo,
		commentLikeRepo: commentLikeRepo,
		db:              db,
	}
}

// CreateComment 创建评论
func (s *commentService) CreateComment(ctx context.Context, req *model.CreateCommentRequest) (*model.Comment, error) {
	// 参数校验
	if req.Content == "" {
		return nil, errors.BadRequest("评论内容不能为空")
	}

	// 检查文章是否存在
	article, err := s.commentRepo.GetArticle(ctx, req.ArticleID)
	if err != nil {
		return nil, err
	}

	// 检查文章是否允许评论
	if !article.AllowComment {
		return nil, errors.CommentDisabled()
	}

	// 检查用户是否在黑名单：api 直接取用户服务 pb 生成的全方法名常量
	// （"/user.v1.UserService/IsInBlacklist"），由 proto 单一来源产出，方法改名时编译期报错。
	var blkResp user.IsBlacklistResponse
	if err := grpcclient.SendRequest(ctx, user.UserService_IsInBlacklist_FullMethodName, &user.IsBlacklistRequest{
		UserId:       uint32(article.UserID),
		TargetUserId: uint32(req.UserID),
	}, &blkResp); err == nil && blkResp.InBlacklist {
		return nil, errors.InBlacklist()
	}

	// 创建评论
	comment := &model.Comment{
		ArticleID: req.ArticleID,
		UserID:    req.UserID,
		ParentID:  req.ParentID,
		Content:   req.Content,
		Status:    1,
	}

	// 使用事务
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建评论
		if err := tx.Create(comment).Error; err != nil {
			return errors.CommentCreateFailed(err)
		}

		// 如果是回复，增加父评论的回复数
		if req.ParentID > 0 {
			tx.Model(&model.Comment{}).Where("id = ?", req.ParentID).
				UpdateColumn("reply_count", gorm.Expr("reply_count + 1"))
		}

		// 增加文章评论数
		tx.Model(&model.Article{}).Where("id = ?", req.ArticleID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1"))

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 重新获取完整评论信息
	return s.commentRepo.GetByID(ctx, comment.ID)
}

// GetComment 获取评论详情
func (s *commentService) GetComment(ctx context.Context, id uint) (*model.Comment, error) {
	return s.commentRepo.GetByID(ctx, id)
}

// UpdateComment 更新评论
func (s *commentService) UpdateComment(ctx context.Context, id uint, req *model.UpdateCommentRequest) (*model.Comment, error) {
	// 获取原评论
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 检查权限
	if comment.UserID != req.UserID {
		return nil, errors.PermissionDenied()
	}

	// 更新内容
	comment.Content = req.Content

	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}

	return s.commentRepo.GetByID(ctx, id)
}

// DeleteComment 删除评论
func (s *commentService) DeleteComment(ctx context.Context, id uint, req *model.DeleteCommentRequest) error {
	// 获取原评论
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限（评论作者或管理员）
	if comment.UserID != req.UserID && req.IsAdmin == 0 {
		return errors.PermissionDenied()
	}

	// 使用事务
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 如果是回复，减少父评论的回复数
		if comment.ParentID > 0 {
			tx.Model(&model.Comment{}).Where("id = ?", comment.ParentID).
				UpdateColumn("reply_count", gorm.Expr("GREATEST(reply_count - 1, 0)"))
		}

		// 删除该评论的所有回复
		tx.Where("parent_id = ?", id).Delete(&model.Comment{})

		// 删除评论
		tx.Delete(&model.Comment{}, id)

		// 减少文章评论数
		tx.Model(&model.Article{}).Where("id = ?", comment.ArticleID).
			UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - 1, 0)"))

		return nil
	})
}

// ListComments 获取用户评论列表
func (s *commentService) ListComments(ctx context.Context, req *model.ListCommentsRequest) ([]*model.Comment, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}

	return s.commentRepo.ListByUser(ctx, req.UserID, int(req.Page), int(req.Size))
}

// GetArticleComments 获取文章评论
func (s *commentService) GetArticleComments(ctx context.Context, req *model.GetArticleCommentsRequest) ([]*model.Comment, int64, bool, error) {
	// 检查文章是否存在
	article, err := s.commentRepo.GetArticle(ctx, req.ArticleID)
	if err != nil {
		return nil, 0, false, err
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}

	comments, total, err := s.commentRepo.GetByArticleID(ctx, req.ArticleID, int(req.Page), int(req.Size), req.IncludeReplies)
	if err != nil {
		return nil, 0, false, err
	}

	return comments, total, article.AllowComment, nil
}

// ReplyComment 回复评论
func (s *commentService) ReplyComment(ctx context.Context, parentID uint, req *model.ReplyCommentRequest) (*model.Comment, error) {
	// 参数校验
	if req.Content == "" {
		return nil, errors.BadRequest("回复内容不能为空")
	}

	// 获取父评论
	parentComment, err := s.commentRepo.GetByID(ctx, parentID)
	if err != nil {
		return nil, err
	}

	// 检查文章是否存在且允许评论
	article, err := s.commentRepo.GetArticle(ctx, parentComment.ArticleID)
	if err != nil {
		return nil, err
	}
	if !article.AllowComment {
		return nil, errors.CommentDisabled()
	}

	// 创建回复
	reply := &model.Comment{
		ArticleID: parentComment.ArticleID,
		UserID:    req.UserID,
		ParentID:  parentID,
		Content:   req.Content,
		Status:    1,
	}

	// 使用事务
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建回复
		if err := tx.Create(reply).Error; err != nil {
			return errors.CommentCreateFailed(err)
		}

		// 增加父评论的回复数
		tx.Model(&model.Comment{}).Where("id = ?", parentID).
			UpdateColumn("reply_count", gorm.Expr("reply_count + 1"))

		// 增加文章评论数
		tx.Model(&model.Article{}).Where("id = ?", parentComment.ArticleID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1"))

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.commentRepo.GetByID(ctx, reply.ID)
}

// LikeComment 点赞评论
func (s *commentService) LikeComment(ctx context.Context, commentID uint, req *model.LikeCommentRequest) (uint, error) {
	// 并行：检查评论是否存在 + 检查是否已点赞（两个查询互不依赖）
	results := pool.Go(ctx,
		func(ctx context.Context) (interface{}, error) {
			_, err := s.commentRepo.GetByID(ctx, commentID)
			return nil, err
		},
		func(ctx context.Context) (interface{}, error) {
			return s.commentLikeRepo.GetByCommentAndUser(ctx, commentID, req.UserID)
		},
	)

	if results[0].Err != nil {
		return 0, results[0].Err
	}
	if results[1].Err != nil {
		return 0, results[1].Err
	}

	existingLike, _ := results[1].Value.(*model.CommentLike)
	if existingLike != nil {
		return 0, errors.AlreadyLiked()
	}

	// 创建点赞记录
	like := &model.CommentLike{
		CommentID: commentID,
		UserID:    req.UserID,
	}

	// 使用事务
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建点赞记录
		if err := tx.Create(like).Error; err != nil {
			return err
		}

		// 增加评论点赞数
		tx.Model(&model.Comment{}).Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1"))

		return nil
	})

	if err != nil {
		return 0, errors.Internal(fmt.Sprintf("点赞失败: %v", err))
	}

	// 获取最新的点赞数
	return s.commentLikeRepo.GetLikeCount(ctx, commentID)
}

// GetCommentReplies 获取评论回复
func (s *commentService) GetCommentReplies(ctx context.Context, req *model.GetCommentRepliesRequest) ([]*model.Comment, int64, error) {
	// 检查评论是否存在
	_, err := s.commentRepo.GetByID(ctx, req.CommentID)
	if err != nil {
		return nil, 0, err
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}

	return s.commentRepo.GetRepliesByParentID(ctx, req.CommentID, int(req.Page), int(req.Size))
}

// EnableComment 开启文章评论
func (s *commentService) EnableComment(ctx context.Context, req *model.EnableCommentRequest) error {
	// 获取文章
	article, err := s.commentRepo.GetArticle(ctx, req.ArticleID)
	if err != nil {
		return err
	}

	// 检查权限（只有文章作者可以开启/关闭评论）
	if article.UserID != req.UserID {
		return errors.PermissionDenied()
	}

	return s.commentRepo.UpdateArticleAllowComment(ctx, req.ArticleID, true)
}

// DisableComment 关闭文章评论
func (s *commentService) DisableComment(ctx context.Context, req *model.DisableCommentRequest) error {
	// 获取文章
	article, err := s.commentRepo.GetArticle(ctx, req.ArticleID)
	if err != nil {
		return err
	}

	// 检查权限
	if article.UserID != req.UserID {
		return errors.PermissionDenied()
	}

	return s.commentRepo.UpdateArticleAllowComment(ctx, req.ArticleID, false)
}

// CacheKey 生成缓存键
func CacheKey(key string, args ...interface{}) string {
	return fmt.Sprintf("comment_service:%s:%v", key, args)
}
