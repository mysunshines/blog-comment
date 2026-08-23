package repository

import (
	"context"
	"errors"

	"github.com/mysunshines/blog-comment/internal/model"

	"gorm.io/gorm"
)

// CommentLikeRepository 点赞数据访问层接口
type CommentLikeRepository interface {
	GetByCommentAndUser(ctx context.Context, commentID, userID uint) (*model.CommentLike, error)
	GetLikeCount(ctx context.Context, commentID uint) (uint, error)
	DeleteByCommentAndUser(ctx context.Context, commentID, userID uint) error
}

// commentLikeRepository 点赞数据访问层实现
type commentLikeRepository struct {
	db *gorm.DB
}

// NewCommentLikeRepository 创建点赞仓储
func NewCommentLikeRepository(db *gorm.DB) CommentLikeRepository {
	return &commentLikeRepository{db: db}
}

// GetByCommentAndUser 根据评论和用户获取点赞记录
func (r *commentLikeRepository) GetByCommentAndUser(ctx context.Context, commentID, userID uint) (*model.CommentLike, error) {
	var like model.CommentLike
	if err := r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		First(&like).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &like, nil
}

// GetLikeCount 获取点赞数
func (r *commentLikeRepository) GetLikeCount(ctx context.Context, commentID uint) (uint, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.CommentLike{}).
		Where("comment_id = ?", commentID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return uint(count), nil
}

// DeleteByCommentAndUser 删除指定评论下某用户的点赞记录（用于取消点赞）
func (r *commentLikeRepository) DeleteByCommentAndUser(ctx context.Context, commentID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Delete(&model.CommentLike{}).Error
}
