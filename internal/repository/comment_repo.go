package repository

import (
	"context"

	"github.com/mysunshines/blog-comment/internal/model"
	"github.com/mysunshines/gocommon/pool"

	"gorm.io/gorm"
)

// CommentRepository 评论数据访问层接口
type CommentRepository interface {
	// 评论基础操作
	Create(ctx context.Context, comment *model.Comment) error
	GetByID(ctx context.Context, id uint) (*model.Comment, error)
	Update(ctx context.Context, comment *model.Comment) error
	Delete(ctx context.Context, id uint) error

	// 查询操作
	ListByArticle(ctx context.Context, articleID uint, parentID uint, page, pageSize int) ([]*model.Comment, int64, error)
	ListByUser(ctx context.Context, userID uint, page, pageSize int) ([]*model.Comment, int64, error)
	GetReplies(ctx context.Context, commentID uint, page, pageSize int) ([]*model.Comment, int64, error)
	GetRepliesByParentID(ctx context.Context, parentID uint, page, pageSize int) ([]*model.Comment, int64, error)
	GetByIDWithUser(ctx context.Context, id uint) (*model.Comment, error)
	GetByArticleID(ctx context.Context, articleID uint, page, pageSize int, includeReplies bool) ([]*model.Comment, int64, error)

	// 点赞操作
	CreateLike(ctx context.Context, like *model.CommentLike) error
	DeleteLike(ctx context.Context, commentID, userID uint) error
	GetLike(ctx context.Context, commentID, userID uint) (*model.CommentLike, error)

	// 文章操作
	GetArticle(ctx context.Context, articleID uint) (*model.Article, error)
	UpdateArticleCommentCount(ctx context.Context, articleID uint, delta int) error
	UpdateArticleCommentEnabled(ctx context.Context, articleID uint, enabled bool) error
	UpdateArticleAllowComment(ctx context.Context, articleID uint, allow bool) error

	// 统计操作
	UpdateReplyCount(ctx context.Context, commentID uint, delta int) error
	UpdateLikeCount(ctx context.Context, commentID uint, delta int) error
}

// commentRepository 评论数据访问层实现
type commentRepository struct {
	db *gorm.DB
}

// NewCommentRepository 创建评论仓储
func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

// Create 创建评论
func (r *commentRepository) Create(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

// GetByID 根据ID获取评论
func (r *commentRepository) GetByID(ctx context.Context, id uint) (*model.Comment, error) {
	var comment model.Comment
	if err := r.db.WithContext(ctx).First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetByIDWithUser 根据ID获取评论（包含用户信息）
func (r *commentRepository) GetByIDWithUser(ctx context.Context, id uint) (*model.Comment, error) {
	var comment model.Comment
	if err := r.db.WithContext(ctx).Preload("User").First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// Update 更新评论
func (r *commentRepository) Update(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Save(comment).Error
}

// Delete 删除评论
func (r *commentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Comment{}, id).Error
}

// ListByArticle 获取文章评论列表
func (r *commentRepository) ListByArticle(ctx context.Context, articleID uint, parentID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&model.Comment{}).Where("article_id = ?", articleID)
	if parentID > 0 {
		baseQuery = baseQuery.Where("parent_id = ?", parentID)
	} else {
		baseQuery = baseQuery.Where("parent_id = 0")
	}

	offset := (page - 1) * pageSize

	// 并行：COUNT + SELECT
	results := pool.Go(ctx,
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).Count(&total).Error
		},
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).
				Preload("User").
				Order("created_at DESC").
				Offset(offset).
				Limit(pageSize).
				Find(&comments).Error
		},
	)

	if results[0].Err != nil {
		return nil, 0, results[0].Err
	}
	if results[1].Err != nil {
		return nil, 0, results[1].Err
	}

	return comments, total, nil
}

// ListByUser 获取用户评论列表
func (r *commentRepository) ListByUser(ctx context.Context, userID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&model.Comment{}).Where("user_id = ?", userID)
	offset := (page - 1) * pageSize

	// 并行：COUNT + SELECT
	results := pool.Go(ctx,
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).Count(&total).Error
		},
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).
				Preload("User").Preload("Article").
				Order("created_at DESC").
				Offset(offset).
				Limit(pageSize).
				Find(&comments).Error
		},
	)

	if results[0].Err != nil {
		return nil, 0, results[0].Err
	}
	if results[1].Err != nil {
		return nil, 0, results[1].Err
	}

	return comments, total, nil
}

// GetReplies 获取评论回复列表
func (r *commentRepository) GetReplies(ctx context.Context, commentID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	var replies []*model.Comment
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&model.Comment{}).Where("parent_id = ? AND status = 1", commentID)
	offset := (page - 1) * pageSize

	// 并行：COUNT + SELECT
	results := pool.Go(ctx,
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).Count(&total).Error
		},
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).
				Preload("User").
				Order("created_at ASC").
				Offset(offset).
				Limit(pageSize).
				Find(&replies).Error
		},
	)

	if results[0].Err != nil {
		return nil, 0, results[0].Err
	}
	if results[1].Err != nil {
		return nil, 0, results[1].Err
	}

	return replies, total, nil
}

// CreateLike 创建点赞
func (r *commentRepository) CreateLike(ctx context.Context, like *model.CommentLike) error {
	return r.db.WithContext(ctx).Create(like).Error
}

// DeleteLike 删除点赞
func (r *commentRepository) DeleteLike(ctx context.Context, commentID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Delete(&model.CommentLike{}).Error
}

// GetLike 获取点赞记录
func (r *commentRepository) GetLike(ctx context.Context, commentID, userID uint) (*model.CommentLike, error) {
	var like model.CommentLike
	if err := r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		First(&like).Error; err != nil {
		return nil, err
	}
	return &like, nil
}

// GetArticle 获取文章
func (r *commentRepository) GetArticle(ctx context.Context, articleID uint) (*model.Article, error) {
	var article model.Article
	if err := r.db.WithContext(ctx).First(&article, articleID).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

// UpdateArticleCommentCount 更新文章评论数
func (r *commentRepository) UpdateArticleCommentCount(ctx context.Context, articleID uint, delta int) error {
	return r.db.WithContext(ctx).
		Model(&model.Article{}).
		Where("id = ?", articleID).
		Update("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

// UpdateArticleCommentEnabled 更新文章评论开关
func (r *commentRepository) UpdateArticleCommentEnabled(ctx context.Context, articleID uint, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&model.Article{}).
		Where("id = ?", articleID).
		Update("allow_comment", enabled).Error
}

// UpdateReplyCount 更新回复数
func (r *commentRepository) UpdateReplyCount(ctx context.Context, commentID uint, delta int) error {
	return r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("reply_count", gorm.Expr("reply_count + ?", delta)).Error
}

// UpdateLikeCount 更新点赞数
func (r *commentRepository) UpdateLikeCount(ctx context.Context, commentID uint, delta int) error {
	return r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// GetRepliesByParentID 根据父评论ID获取回复列表
func (r *commentRepository) GetRepliesByParentID(ctx context.Context, parentID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	return r.GetReplies(ctx, parentID, page, pageSize)
}

// GetByArticleID 根据文章ID获取评论列表
func (r *commentRepository) GetByArticleID(ctx context.Context, articleID uint, page, pageSize int, includeReplies bool) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&model.Comment{}).Where("article_id = ? AND parent_id = 0", articleID)
	offset := (page - 1) * pageSize

	// 并行：COUNT + SELECT
	results := pool.Go(ctx,
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).Count(&total).Error
		},
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).
				Preload("User").
				Order("created_at DESC").
				Offset(offset).
				Limit(pageSize).
				Find(&comments).Error
		},
	)

	if results[0].Err != nil {
		return nil, 0, results[0].Err
	}
	if results[1].Err != nil {
		return nil, 0, results[1].Err
	}

	// 如果需要加载回复（依赖 comments ID，必须在 SELECT 之后）
	if includeReplies && len(comments) > 0 {
		parentIDs := make([]uint, len(comments))
		for i, c := range comments {
			parentIDs[i] = c.ID
		}
		var replies []*model.Comment
		r.db.WithContext(ctx).
			Where("parent_id IN ? AND status = 1", parentIDs).
			Preload("User").
			Order("created_at ASC").
			Find(&replies)

		// 建立映射
		replyMap := make(map[uint][]model.Comment)
		for _, reply := range replies {
			replyMap[reply.ParentID] = append(replyMap[reply.ParentID], *reply)
		}
		// 填充回复
		for _, comment := range comments {
			comment.Replies = replyMap[comment.ID]
		}
	}

	return comments, total, nil
}

// UpdateArticleAllowComment 更新文章评论开关
func (r *commentRepository) UpdateArticleAllowComment(ctx context.Context, articleID uint, allow bool) error {
	return r.UpdateArticleCommentEnabled(ctx, articleID, allow)
}
