package service

import (
	"context"
	"fmt"
	"log"

	"github.com/mysunshines/blog-comment/internal/client"
	"github.com/mysunshines/blog-comment/internal/errors"
	"github.com/mysunshines/blog-comment/internal/model"
	"github.com/mysunshines/blog-comment/internal/repository"
	notification "github.com/mysunshines/blog-notification/proto/pb/v1"
	user "github.com/mysunshines/blog-user/proto/pb/v1"
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
	LikeComment(ctx context.Context, commentID uint, req *model.LikeCommentRequest) (uint, bool, error)
	GetCommentReplies(ctx context.Context, req *model.GetCommentRepliesRequest) ([]*model.Comment, int64, error)
	GetArticleAnnotations(ctx context.Context, articleID uint) ([]*model.Comment, error)
	EnableComment(ctx context.Context, req *model.EnableCommentRequest) error
	DisableComment(ctx context.Context, req *model.DisableCommentRequest) error

	// 管理员操作
	AdminListComments(ctx context.Context, articleID, userID uint, keyword string, page, pageSize int) ([]*model.Comment, int64, error)
	AdminDeleteComment(ctx context.Context, commentID uint) error
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

	// 行内批注：当三字段同时有效时记录锚点，供前端渲染高亮。
	// 父评论（被回复对象）上记录的锚点会作为整条批注线程的锚点，
	// 因此回复时若父评论是批注，回复自动继承其锚点（不覆盖）。
	if req.ParagraphIndex >= 0 && req.AnchorText != "" && req.AnchorOffset >= 0 {
		comment.ParagraphIndex = req.ParagraphIndex
		comment.AnchorText = req.AnchorText
		comment.AnchorOffset = req.AnchorOffset
		comment.AnchorPrefix = req.AnchorPrefix
		comment.AnchorSuffix = req.AnchorSuffix
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

	// 新评论 → 评论榜 +1（best-effort，失败不影响主流程）
	if perr := client.IncrCommentScore(ctx, req.ArticleID, 1); perr != nil {
		log.Printf("[ranking] push comment score failed article=%d: %v", req.ArticleID, perr)
	}

	// 通知文章作者收到新评论（自己评论自己的文章不打扰）
	if article.UserID != req.UserID {
		s.notifyUser(ctx, article.UserID, notification.NotificationType_ARTICLE_COMMENTED,
			fmt.Sprintf("你的文章《%s》收到了新评论", article.Title),
			truncateRunes(comment.Content, 200),
			articleLink(article.ID), req.UserID)
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

// DeleteComment 逻辑删除评论（status 置 2，不物理删除）。
// 被删节点若存在直接子级（楼中楼回复），这些子级自动上提一级，
// parent_id 改为被删节点的父级，从而保证子孙回复仍然可见、不产生孤儿。
// 若该评论本身是主评论，则其回复会升级为顶层主评论。
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

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 取出被删节点的所有直接子级（仅正常状态）
		var children []*model.Comment
		if err := tx.Where("parent_id = ? AND status = 1", id).Find(&children).Error; err != nil {
			return err
		}

		// 2. 上提：直接子级的 parent_id 改为被删节点的父级（保留可见、不丢数据）
		if len(children) > 0 {
			if err := tx.Model(&model.Comment{}).
				Where("parent_id = ? AND status = 1", id).
				Update("parent_id", comment.ParentID).Error; err != nil {
				return err
			}
		}

		// 3. 逻辑删除本节点（status=2），不物理删除
		if err := tx.Model(&model.Comment{}).
			Where("id = ?", id).
			Update("status", uint(2)).Error; err != nil {
			return err
		}

		// 4. 计数调整
		// 4.1 文章评论数 -1（本条评论不再可见）
		if err := tx.Model(&model.Article{}).
			Where("id = ?", comment.ArticleID).
			UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - 1, 0)")).Error; err != nil {
			return err
		}

		// 4.2 若有父节点：父级 reply_count 变化 = 原直接子(被删) -1 + 上提子级(+len)
		//     净变化 = len(children) - 1；无父节点（主评论）无需调整 reply_count。
		if comment.ParentID > 0 {
			if err := tx.Model(&model.Comment{}).
				Where("id = ?", comment.ParentID).
				UpdateColumn("reply_count", gorm.Expr("GREATEST(reply_count + ?, 0)", len(children)-1)).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	// 逻辑删除成功 → 评论榜 -1（best-effort，失败不影响主流程）
	if perr := client.IncrCommentScore(ctx, comment.ArticleID, -1); perr != nil {
		log.Printf("[ranking] push comment score failed article=%d: %v", comment.ArticleID, perr)
	}
	return nil
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

// GetArticleAnnotations 获取文章的行内批注锚点（供 article-service 渲染时注入高亮标记）
func (s *commentService) GetArticleAnnotations(ctx context.Context, articleID uint) ([]*model.Comment, error) {
	var comments []*model.Comment
	if err := s.db.WithContext(ctx).
		Select("id, article_id, paragraph_index, anchor_text, anchor_offset, anchor_prefix, anchor_suffix").
		Where("article_id = ? AND paragraph_index >= 0 AND anchor_text != '' AND status = 1", articleID).
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
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

	comments, total, err := s.commentRepo.GetByArticleID(ctx, req.ArticleID, int(req.Page), int(req.Size), req.IncludeReplies, req.Sort)
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

	// 创建回复。
	// 保留真实层级：reply.ParentID = 被回复评论(parentID)，
	// 无论回复的是主评论还是其第 N 级子评论，都原样指向它的上一级。
	// 同时记录 root_id = 根主评论，便于 GetByArticleID 一次性取全整个线程。
	// 注意：root_id 应沿父链追溯到真正顶层主评论，而不是只向上取一代。
	rootID := parentID
	if parentComment.RootID != 0 {
		rootID = parentComment.RootID
	}

	reply := &model.Comment{
		ArticleID: parentComment.ArticleID,
		UserID:    req.UserID,
		ParentID:  parentID, // 真实上一级评论
		RootID:    rootID,   // 根主评论线程
		Content:   req.Content,
		Status:    1,
	}

	// 使用事务
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建回复
		if err := tx.Create(reply).Error; err != nil {
			return errors.CommentCreateFailed(err)
		}

		// 增加根主评论的回复数
		tx.Model(&model.Comment{}).Where("id = ?", rootID).
			UpdateColumn("reply_count", gorm.Expr("reply_count + 1"))

		// 增加文章评论数
		tx.Model(&model.Article{}).Where("id = ?", parentComment.ArticleID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1"))

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 新回复 → 评论榜 +1（best-effort，失败不影响主流程）
	if perr := client.IncrCommentScore(ctx, parentComment.ArticleID, 1); perr != nil {
		log.Printf("[ranking] push comment score failed article=%d: %v", parentComment.ArticleID, perr)
	}

	// 通知被回复者（自己回复自己的评论不打扰）
	if parentComment.UserID != req.UserID {
		s.notifyUser(ctx, parentComment.UserID, notification.NotificationType_COMMENT_REPLIED,
			"你的评论收到了新回复",
			truncateRunes(reply.Content, 200),
			articleLink(parentComment.ArticleID), req.UserID)
	}
	// 通知文章作者（作者不是回复者、也不是被回复者时才通知，避免同一行为重复打扰）
	if article.UserID != req.UserID && article.UserID != parentComment.UserID {
		s.notifyUser(ctx, article.UserID, notification.NotificationType_ARTICLE_COMMENTED,
			fmt.Sprintf("你的文章《%s》收到了新评论", article.Title),
			truncateRunes(reply.Content, 200),
			articleLink(article.ID), req.UserID)
	}

	return s.commentRepo.GetByID(ctx, reply.ID)
}

// LikeComment 对评论点赞/取消点赞（toggle）。
// 同一用户对同一评论只能点赞一次：已点赞时再次调用则取消点赞并返回 liked=false，
// 未点赞时调用则点赞并返回 liked=true。返回操作后的最新点赞数与点赞状态。
func (s *commentService) LikeComment(ctx context.Context, commentID uint, req *model.LikeCommentRequest) (uint, bool, error) {
	// 并行：检查评论是否存在 + 检查是否已点赞（两个查询互不依赖）
	var target *model.Comment
	results := pool.Go(ctx,
		func(ctx context.Context) (interface{}, error) {
			c, err := s.commentRepo.GetByID(ctx, commentID)
			target = c
			return nil, err
		},
		func(ctx context.Context) (interface{}, error) {
			return s.commentLikeRepo.GetByCommentAndUser(ctx, commentID, req.UserID)
		},
	)

	if results[0].Err != nil {
		return 0, false, results[0].Err
	}
	if results[1].Err != nil {
		return 0, false, results[1].Err
	}

	existingLike, _ := results[1].Value.(*model.CommentLike)

	// 使用事务：已点赞则取消，未点赞则新增
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if existingLike != nil {
			// 取消点赞
			if err := s.commentLikeRepo.DeleteByCommentAndUser(ctx, commentID, req.UserID); err != nil {
				return err
			}
			// 防止 like_count 变负
			tx.Model(&model.Comment{}).Where("id = ? AND like_count > 0", commentID).
				UpdateColumn("like_count", gorm.Expr("like_count - 1"))
			return nil
		}

		// 新增点赞记录
		like := &model.CommentLike{
			CommentID: commentID,
			UserID:    req.UserID,
		}
		if err := tx.Create(like).Error; err != nil {
			return err
		}
		tx.Model(&model.Comment{}).Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1"))
		return nil
	})

	if err != nil {
		return 0, false, errors.Internal("点赞失败", err)
	}

	liked := existingLike == nil

	// 新点赞（非取消）时通知评论作者（自己赞自己不打扰）。
	// 通知是 best-effort，失败不影响点赞结果返回。
	if liked && target != nil && target.UserID != 0 && target.UserID != req.UserID {
		s.notifyUser(ctx, target.UserID, notification.NotificationType_COMMENT_LIKED,
			"你的评论收到了点赞",
			"",
			articleLink(target.ArticleID), req.UserID)
	}

	count, err := s.commentLikeRepo.GetLikeCount(ctx, commentID)
	if err != nil {
		return 0, liked, err
	}
	return count, liked, nil
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

// AdminListComments 管理端评论列表（支持按文章/用户/关键字过滤）
func (s *commentService) AdminListComments(ctx context.Context, articleID, userID uint, keyword string, page, pageSize int) ([]*model.Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return s.commentRepo.AdminList(ctx, articleID, userID, keyword, page, pageSize)
}

// AdminDeleteComment 管理端删除评论（无视作者，直接删除）
func (s *commentService) AdminDeleteComment(ctx context.Context, commentID uint) error {
	return s.DeleteComment(ctx, commentID, &model.DeleteCommentRequest{IsAdmin: 1})
}

// CacheKey 生成缓存键
func CacheKey(key string, args ...interface{}) string {
	return fmt.Sprintf("comment_service:%s:%v", key, args)
}

// notifyUser 向指定用户发送站内消息（best-effort：失败仅打日志，不影响主流程）。
// actorID 为触发者（评论人/点赞人），尽量拉取昵称用于展示，失败降级为"用户"；
// userID 或 actorID 为 0 时直接跳过（0 表示系统/匿名，当前评论场景不适用）。
func (s *commentService) notifyUser(ctx context.Context, userID uint, typ notification.NotificationType,
	title, content, link string, actorID uint) {
	if userID == 0 || actorID == 0 {
		return
	}
	actorName := "用户"
	if u, err := client.GetUser(ctx, actorID); err == nil && u != nil {
		if u.Nickname != "" {
			actorName = u.Nickname
		} else if u.Username != "" {
			actorName = u.Username
		}
	}
	if err := client.CreateNotification(ctx, userID, typ, title, content, link, actorID, actorName); err != nil {
		log.Printf("[notification][warn] notify user %d failed: %v", userID, err)
	}
}

// articleLink 生成文章跳转链接（前端路由，评论服务的本地文章模型无 slug，用 id 定位）
func articleLink(articleID uint) string {
	return fmt.Sprintf("/article.html?id=%d", articleID)
}

// truncateRunes 按 rune 截断字符串到 max 个字符，超出部分以 … 结尾，
// 防止评论正文超出 notification-service 的 content 长度限制（1024）。
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
