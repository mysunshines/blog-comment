package model

import (
	"time"
)

// Comment 评论模型
type Comment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ArticleID  uint      `gorm:"not null;index" json:"article_id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	ParentID   uint      `gorm:"default:0;index" json:"parent_id"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	LikeCount  uint      `gorm:"default:0" json:"like_count"`
	ReplyCount uint      `gorm:"default:0" json:"reply_count"`
	Status     uint      `gorm:"default:1" json:"status"` // 1=正常, 2=已删除
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// 关联
	User    User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Article Article   `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
	Replies []Comment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

func (Comment) TableName() string {
	return "comments"
}

// CommentLike 评论点赞模型
type CommentLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CommentID uint      `gorm:"not null;uniqueIndex:idx_comment_user" json:"comment_id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_comment_user" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (CommentLike) TableName() string {
	return "comment_likes"
}

// Article 文章模型（用于关联查询）
type Article struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index" json:"user_id"`
	Title        string    `gorm:"size:256" json:"title"`
	AllowComment bool      `gorm:"default:true" json:"allow_comment"`
	CommentCount int       `gorm:"default:0" json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Article) TableName() string {
	return "articles"
}

// User 用户模型（用于关联查询）
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:64" json:"username"`
	Nickname  string    `gorm:"size:64" json:"nickname"`
	Avatar    string    `gorm:"size:256" json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
}

func (User) TableName() string {
	return "users"
}

// DTO 请求结构

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	UserID    uint   `json:"user_id"`
	ArticleID uint   `json:"article_id" binding:"required"`
	Content   string `json:"content" binding:"required,min=1,max=2000"`
	ParentID  uint   `json:"parent_id"`
}

// UpdateCommentRequest 更新评论请求
type UpdateCommentRequest struct {
	UserID  uint   `json:"user_id"`
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

// ListCommentsRequest 评论列表请求
type ListCommentsRequest struct {
	Page   uint `form:"page"`
	Size   uint `form:"size"`
	UserID uint `form:"user_id"`
}

// GetArticleCommentsRequest 获取文章评论请求
type GetArticleCommentsRequest struct {
	ArticleID      uint `form:"article_id" binding:"required"`
	Page           uint `form:"page"`
	Size           uint `form:"size"`
	IncludeReplies bool `form:"include_replies"`
}

// GetCommentRepliesRequest 获取评论回复请求
type GetCommentRepliesRequest struct {
	CommentID uint `form:"comment_id" binding:"required"`
	Page      uint `form:"page"`
	Size      uint `form:"size"`
}

// ReplyCommentRequest 回复评论请求
type ReplyCommentRequest struct {
	UserID  uint   `json:"user_id"`
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

// LikeCommentRequest 点赞评论请求
type LikeCommentRequest struct {
	UserID uint `json:"user_id"`
}

// EnableCommentRequest 开启评论请求
type EnableCommentRequest struct {
	UserID    uint `json:"user_id"`
	ArticleID uint `json:"article_id" binding:"required"`
}

// DisableCommentRequest 关闭评论请求
type DisableCommentRequest struct {
	UserID    uint `json:"user_id"`
	ArticleID uint `json:"article_id" binding:"required"`
}

// DeleteCommentRequest 删除评论请求
type DeleteCommentRequest struct {
	UserID  uint `json:"user_id"`
	IsAdmin uint `json:"is_admin"`
}
