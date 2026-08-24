package repository

import (
	"context"

	"github.com/mysunshines/blog-comment/internal/model"
	"github.com/mysunshines/gocommon/pool"

	"gorm.io/gorm"
)

// maxRepliesPerQuery 楼中楼回复单次加载上限：GetByArticleID 只加载"当前页主评论"
// 线程的回复，且不超过该上限，避免热门文章整篇回复（可达数千条）一次性全量拉取
// 造成内存与延迟放大。超过上限时截断展示，主评论分页本身不受影响。
const maxRepliesPerQuery = 1000

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
	AdminList(ctx context.Context, articleID, userID uint, keyword string, page, pageSize int) ([]*model.Comment, int64, error)
	GetReplies(ctx context.Context, commentID uint, page, pageSize int) ([]*model.Comment, int64, error)
	GetRepliesByParentID(ctx context.Context, parentID uint, page, pageSize int) ([]*model.Comment, int64, error)
	GetByIDWithUser(ctx context.Context, id uint) (*model.Comment, error)
	GetByArticleID(ctx context.Context, articleID uint, page, pageSize int, includeReplies bool, sort string) ([]*model.Comment, int64, error)

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

// Update 更新评论（只更新非零值字段，不更新 created_at）
func (r *commentRepository) Update(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Select("*").Omit("created_at").Updates(comment).Error
}

// Delete 删除评论
func (r *commentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Comment{}, id).Error
}

// ListByArticle 获取文章评论列表
func (r *commentRepository) ListByArticle(ctx context.Context, articleID uint, parentID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("article_id = ? AND status = 1", articleID)
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

// AdminList 管理端评论列表：支持按文章/用户/关键字过滤（全量，不受 status 限制）。
func (r *commentRepository) AdminList(ctx context.Context, articleID, userID uint, keyword string, page, pageSize int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&model.Comment{})
	if articleID > 0 {
		baseQuery = baseQuery.Where("article_id = ?", articleID)
	}
	if userID > 0 {
		baseQuery = baseQuery.Where("user_id = ?", userID)
	}
	if keyword != "" {
		baseQuery = baseQuery.Where("content LIKE ?", "%"+keyword+"%")
	}
	offset := (page - 1) * pageSize

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
// sort: latest(默认,按创建时间倒序) / oldest(按创建时间正序) / hot(按点赞数倒序)
func (r *commentRepository) GetByArticleID(ctx context.Context, articleID uint, page, pageSize int, includeReplies bool, sort string) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	// 按排序方式确定 ORDER BY 子句
	var orderClause string
	switch sort {
	case "oldest":
		orderClause = "created_at ASC"
	case "hot":
		orderClause = "like_count DESC, created_at DESC"
	default: // latest 及未知值
		orderClause = "created_at DESC"
	}

	baseQuery := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("article_id = ? AND parent_id = 0 AND status = 1", articleID)
	offset := (page - 1) * pageSize

	// 并行：COUNT + SELECT
	results := pool.Go(ctx,
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).Count(&total).Error
		},
		func(ctx context.Context) (interface{}, error) {
			return nil, baseQuery.Session(&gorm.Session{}).
				Preload("User").
				Order(orderClause).
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
		// 主评论 ID 集合，用于判定某回复属于哪个根线程
		rootSet := make(map[uint]bool, len(comments))
		for _, c := range comments {
			rootSet[c.ID] = true
		}

		// 只加载"当前页主评论"的回复线程（root_id 命中当前页主评论，或 parent 直接是
		// 主评论——兼容历史 root_id 缺失数据），而不是整篇文章的所有回复，
		// 避免热门文章回复一次性全量加载；并设单次查询上限保护极端大回复量场景。
		rootIDs := make([]uint, 0, len(comments))
		for _, c := range comments {
			rootIDs = append(rootIDs, c.ID)
		}
		var allReplies []*model.Comment
		r.db.WithContext(ctx).
			Where("article_id = ? AND parent_id != 0 AND status = 1 AND (root_id IN ? OR parent_id IN ?)", articleID, rootIDs, rootIDs).
			Preload("User").
			Order("created_at ASC").
			Limit(maxRepliesPerQuery).
			Find(&allReplies)

		// 按真实 parent_id 关系构建嵌套树。
		// 注意：model.Comment.Replies 是值类型切片，解引用拷贝会丢失后续挂载的子级，
		// 因此分两轮：先让所有回复在其上一级（同为回复）下挂好子树，
		// 再把「直接挂主评论的回复」（第 2 级）以完整对象拷贝到主评论，
		// 确保其 Replies（第 3 级）一并带出。
		byID := make(map[uint]*model.Comment, len(allReplies)+len(comments))
		// 先把主评论与所有回复放入索引（作为潜在父节点）
		for _, c := range comments {
			byID[c.ID] = c
		}
		for _, rply := range allReplies {
			byID[rply.ID] = rply
		}

		// 清空主评论旧的 Replies，避免重复拼接（re-query 时）
		for _, c := range comments {
			c.Replies = nil
		}

		// 第一轮：父节点是「另一条回复」（非主评论）时，挂到 allReplies 内部父对象，
		// 形成完整子树（如第 3 级挂到第 2 级）。
		for _, rply := range allReplies {
			if parent, ok := byID[rply.ParentID]; ok && !rootSet[rply.ParentID] {
				parent.Replies = append(parent.Replies, *rply)
			}
		}

		// 第二轮：父节点是「主评论」（第 2 级回复），从 byID 取已挂好子级的完整对象，
		// 挂到主评论；若父节点既不是主评论也找不到（历史越级数据），兜底挂到根主评论。
		for _, rply := range allReplies {
			if rootSet[rply.ParentID] {
				if full, ok := byID[rply.ID]; ok {
					if root, ok2 := byID[rply.ParentID]; ok2 {
						root.Replies = append(root.Replies, *full)
					}
				}
			} else if _, ok := byID[rply.ParentID]; !ok {
				// 兜底：上一级评论不在本批次（如历史数据 root_id 缺失），挂到根主评论
				if rootSet[rply.RootID] {
					if root, ok := byID[rply.RootID]; ok {
						root.Replies = append(root.Replies, *rply)
					}
				}
			}
		}
	}

	return comments, total, nil
}

// UpdateArticleAllowComment 更新文章评论开关
func (r *commentRepository) UpdateArticleAllowComment(ctx context.Context, articleID uint, allow bool) error {
	return r.UpdateArticleCommentEnabled(ctx, articleID, allow)
}
