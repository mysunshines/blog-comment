// Package v1 存放 comment-service 的 HTTP API 处理器（v1 版本）。
// 后续迭代 v2 接口时，新增 internal/handler/v2 包即可，互不干扰。
package v1

import (
	"strconv"

	"github.com/mysunshines/blog-comment/internal/model"
	"github.com/mysunshines/blog-comment/internal/service"
	"github.com/mysunshines/blog-comment/pkg/response"

	"github.com/mysunshines/gocommon/constants"
	commonmiddleware "github.com/mysunshines/gocommon/middleware"

	"github.com/gin-gonic/gin"
)

// CommentHandler 评论处理器
type CommentHandler struct {
	svc service.CommentService
}

// NewCommentHandler 创建评论处理器
func NewCommentHandler(svc service.CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

// CreateComment 创建评论
// @Summary 创建评论
// @Tags comment
// @Accept json
// @Produce json
// @Param comment body model.CreateCommentRequest true "评论信息"
// @Success 200 {object} response.Response
// @Router /api/v1/comment [post]
func (h *CommentHandler) CreateComment(c *gin.Context) {
	var req model.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}
	req.UserID = userID

	comment, err := h.svc.CreateComment(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.Success(c, comment)
}

// GetComment 获取评论详情
// @Summary 获取评论详情
// @Tags comment
// @Produce json
// @Param id path int true "评论ID"
// @Success 200 {object} response.Response
// @Router /api/v1/comment/{id} [get]
func (h *CommentHandler) GetComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的评论ID")
		return
	}

	comment, err := h.svc.GetComment(c.Request.Context(), uint(id))
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.Success(c, comment)
}

// UpdateComment 更新评论
// @Summary 更新评论
// @Tags comment
// @Accept json
// @Produce json
// @Param id path int true "评论ID"
// @Param comment body model.UpdateCommentRequest true "评论信息"
// @Success 200 {object} response.Response
// @Router /api/v1/comment/{id} [put]
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的评论ID")
		return
	}

	var req model.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}
	req.UserID = userID

	comment, err := h.svc.UpdateComment(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.Success(c, comment)
}

// DeleteComment 删除评论
// @Summary 删除评论
// @Tags comment
// @Produce json
// @Param id path int true "评论ID"
// @Success 200 {object} response.Response
// @Router /api/v1/comment/{id} [delete]
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的评论ID")
		return
	}

	// 获取用户ID和角色
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	role := getUserRole(c)
	isAdmin := uint(0)
	if role == uint(constants.RoleAdmin) { // 2 为管理员角色
		isAdmin = 1
	}

	req := &model.DeleteCommentRequest{
		UserID:  userID,
		IsAdmin: isAdmin,
	}

	if err := h.svc.DeleteComment(c.Request.Context(), uint(id), req); err != nil {
		response.FailWithError(c, err)
		return
	}

	response.SuccessWithMessage(c, "删除成功")
}

// ListComments 获取用户评论列表
// @Summary 获取用户评论列表
// @Tags comment
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param user_id query int false "用户ID"
// @Success 200 {object} response.PageResponse
// @Router /api/v1/comment [get]
func (h *CommentHandler) ListComments(c *gin.Context) {
	var req model.ListCommentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}

	comments, total, err := h.svc.ListComments(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.SuccessPage(c, comments, int64(total), int(req.Page), int(req.Size))
}

// GetArticleComments 获取文章评论
// @Summary 获取文章评论
// @Tags comment
// @Produce json
// @Param article_id path int true "文章ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param include_replies query bool false "包含回复" default(false)
// @Success 200 {object} response.Response
// @Router /api/v1/comment/article/{article_id} [get]
func (h *CommentHandler) GetArticleComments(c *gin.Context) {
	articleIDStr := c.Param("article_id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	var req model.GetArticleCommentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	req.ArticleID = uint(articleID)

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}

	comments, total, commentEnabled, err := h.svc.GetArticleComments(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.Success(c, gin.H{
		"comments":        comments,
		"total":           total,
		"page":            req.Page,
		"size":            req.Size,
		"comment_enabled": commentEnabled,
	})
}

// ReplyComment 回复评论
// @Summary 回复评论
// @Tags comment
// @Accept json
// @Produce json
// @Param id path int true "评论ID"
// @Param reply body model.ReplyCommentRequest true "回复内容"
// @Success 200 {object} response.Response
// @Router /api/v1/comment/{id}/reply [post]
func (h *CommentHandler) ReplyComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的评论ID")
		return
	}

	var req model.ReplyCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 获取用户ID
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}
	req.UserID = userID

	reply, err := h.svc.ReplyComment(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.Success(c, reply)
}

// LikeComment 点赞评论
// @Summary 点赞评论
// @Tags comment
// @Produce json
// @Param id path int true "评论ID"
// @Success 200 {object} response.Response
// @Router /api/v1/comment/{id}/like [post]
func (h *CommentHandler) LikeComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的评论ID")
		return
	}

	// 获取用户ID
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	req := &model.LikeCommentRequest{
		UserID: userID,
	}

	likeCount, err := h.svc.LikeComment(c.Request.Context(), uint(id), req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.Success(c, gin.H{"like_count": likeCount})
}

// GetCommentReplies 获取评论回复
// @Summary 获取评论回复
// @Tags comment
// @Produce json
// @Param id path int true "评论ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} response.PageResponse
// @Router /api/v1/comment/{id}/replies [get]
func (h *CommentHandler) GetCommentReplies(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的评论ID")
		return
	}

	var req model.GetCommentRepliesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	req.CommentID = uint(id)

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}

	replies, total, err := h.svc.GetCommentReplies(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.SuccessPage(c, replies, int64(total), int(req.Page), int(req.Size))
}

// EnableComment 开启文章评论
// @Summary 开启文章评论
// @Tags comment
// @Accept json
// @Produce json
// @Param article_id path int true "文章ID"
// @Success 200 {object} response.Response
// @Router /api/v1/comment/article/{article_id}/enable [post]
func (h *CommentHandler) EnableComment(c *gin.Context) {
	articleIDStr := c.Param("article_id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	// 获取用户ID
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	req := &model.EnableCommentRequest{
		UserID:    userID,
		ArticleID: uint(articleID),
	}

	if err := h.svc.EnableComment(c.Request.Context(), req); err != nil {
		response.FailWithError(c, err)
		return
	}

	response.SuccessWithMessage(c, "评论功能已开启")
}

// DisableComment 关闭文章评论
// @Summary 关闭文章评论
// @Tags comment
// @Accept json
// @Produce json
// @Param article_id path int true "文章ID"
// @Success 200 {object} response.Response
// @Router /api/v1/comment/article/{article_id}/disable [post]
func (h *CommentHandler) DisableComment(c *gin.Context) {
	articleIDStr := c.Param("article_id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	// 获取用户ID
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	req := &model.DisableCommentRequest{
		UserID:    userID,
		ArticleID: uint(articleID),
	}

	if err := h.svc.DisableComment(c.Request.Context(), req); err != nil {
		response.FailWithError(c, err)
		return
	}

	response.SuccessWithMessage(c, "评论功能已关闭")
}

// 辅助函数：从上下文获取用户ID
func getUserID(c *gin.Context) uint {
	if userID, exists := c.Get(commonmiddleware.UserIDContextKey); exists {
		switch v := userID.(type) {
		case uint:
			return v
		case uint32:
			return uint(v)
		case uint64:
			return uint(v)
		case float64:
			return uint(v)
		case int:
			return uint(v)
		case int32:
			return uint(v)
		case int64:
			return uint(v)
		}
	}
	return 0
}

// 辅助函数：从上下文获取用户角色
func getUserRole(c *gin.Context) uint {
	if role, exists := c.Get(commonmiddleware.RoleContextKey); exists {
		switch v := role.(type) {
		case uint:
			return v
		case uint32:
			return uint(v)
		case uint64:
			return uint(v)
		case float64:
			return uint(v)
		case int:
			return uint(v)
		case int32:
			return uint(v)
		case int64:
			return uint(v)
		}
	}
	return 0
}
