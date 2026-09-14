package model

import (
	"time"
)

// Comment 评论模型
type Comment struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	ArticleID  uint   `gorm:"not null;index" json:"article_id"`
	UserID     uint   `gorm:"not null;index" json:"user_id"`
	ParentID   uint   `gorm:"default:0;index" json:"parent_id"` // 上一级评论 ID（回复谁就指向谁，保留真实层级）
	RootID     uint   `gorm:"default:0;index" json:"root_id"`   // 根主评论 ID（同线程所有回复共享，便于一次性取全楼中楼）
	Content    string `gorm:"type:text;not null" json:"content"`
	LikeCount  uint   `gorm:"default:0" json:"like_count"`
	ReplyCount uint   `gorm:"default:0" json:"reply_count"`
	Status     uint   `gorm:"default:1" json:"status"` // 1=正常, 2=已删除

	// 行内批注锚点（Confluence 式：选中正文片段后评论）。
	// 采用 W3C TextQuoteSelector 思路：anchor_text 为精确选中文本，
	// anchor_prefix/anchor_suffix 为其前后上下文（增强抗漂移与消歧义），
	// paragraph_index/anchor_offset 为兼容旧实现的辅助定位。
	// 文章大改导致失配时回退为普通评论（anchor_text 仍展示为引用块）。
	ParagraphIndex int    `gorm:"default:-1" json:"paragraph_index"` // 选中文字所在段落序号，-1 表示非行内批注
	AnchorText     string `gorm:"size:500" json:"anchor_text"`       // 被选中的精确原文片段
	AnchorOffset   int    `gorm:"default:-1" json:"anchor_offset"`   // 片段在段落内的字符偏移，-1 表示未记录
	AnchorPrefix   string `gorm:"size:128" json:"anchor_prefix"`     // 选中文本前的上下文（消歧义/抗漂移）
	AnchorSuffix   string `gorm:"size:128" json:"anchor_suffix"`     // 选中文本后的上下文（消歧义/抗漂移）

	CreatedAt time.Time `gorm:"<-:create" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

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

	// 行内批注入参（可选）：选中正文片段时一并提交，用于后端注入高亮标记。
	// 四者要么同时有效，要么都为空（空表示普通评论）。
	ParagraphIndex int    `json:"paragraph_index"` // 选中片段所在段落序号，-1 表示非行内批注
	AnchorText     string `json:"anchor_text"`     // 被选中的精确原文片段（最多 500 字）
	AnchorOffset   int    `json:"anchor_offset"`   // 片段在段落内的字符偏移，-1 表示未记录
	AnchorPrefix   string `json:"anchor_prefix"`   // 选中文本前上下文（最多 128 字）
	AnchorSuffix   string `json:"anchor_suffix"`   // 选中文本后上下文（最多 128 字）
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
	// 排序方式: latest(默认,按创建时间倒序) / oldest(按创建时间正序) / hot(按点赞数倒序)
	Sort string `form:"sort"`
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
