package v1

import (
	"context"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mysunshines/blog-comment/internal/client"
	"github.com/mysunshines/blog-comment/internal/errors"
	"github.com/mysunshines/blog-comment/internal/model"
	"github.com/mysunshines/blog-comment/internal/service"
	comment "github.com/mysunshines/blog-comment/proto/pb"
	user "github.com/mysunshines/blog-user/proto/pb"

	"github.com/mysunshines/gocommon/constants"
	commonmiddleware "github.com/mysunshines/gocommon/middleware"

	"github.com/sony/gobreaker"
)

// GrpcCommentHandler gRPC 评论处理器（支持熔断）
type GrpcCommentHandler struct {
	comment.UnimplementedCommentServiceServer
	Svc service.CommentService
	Cb  *gobreaker.CircuitBreaker
}

func (h *GrpcCommentHandler) CreateComment(ctx context.Context, req *comment.CreateCommentRequest) (*comment.CreateCommentResponse, error) {
	uid, err := commonmiddleware.RequireGRPCAuth(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.Svc.CreateComment(ctx, &model.CreateCommentRequest{
		UserID:    uid,
		ArticleID: uint(req.ArticleId),
		Content:   req.Content,
		ParentID:  uint(req.ParentId),
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return &comment.CreateCommentResponse{
				Code:    uint32(appErr.Code),
				Message: appErr.Message,
			}, nil
		}
		return &comment.CreateCommentResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_CREATE_FAILED),
			Message: err.Error(),
		}, nil
	}

	recordAudit(ctx, user.AuditAction_AUDIT_ACTION_COMMENT_CREATE, "comment", uint(c.ID), req.Content)
	return &comment.CreateCommentResponse{
		Code:    uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message: "success",
		Comment: ConvertToProtoComment(c),
	}, nil
}

func (h *GrpcCommentHandler) GetComment(ctx context.Context, req *comment.GetCommentRequest) (*comment.GetCommentResponse, error) {
	c, err := h.Svc.GetComment(ctx, uint(req.CommentId))
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return &comment.GetCommentResponse{
				Code:    uint32(appErr.Code),
				Message: appErr.Message,
			}, nil
		}
		return &comment.GetCommentResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_NOT_FOUND),
			Message: "Comment not found",
		}, nil
	}

	return &comment.GetCommentResponse{
		Code:    uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message: "success",
		Comment: ConvertToProtoComment(c),
	}, nil
}

func (h *GrpcCommentHandler) UpdateComment(ctx context.Context, req *comment.UpdateCommentRequest) (*comment.UpdateCommentResponse, error) {
	uid, err := commonmiddleware.RequireGRPCAuth(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.Svc.UpdateComment(ctx, uint(req.CommentId), &model.UpdateCommentRequest{
		UserID:  uid,
		Content: req.Content,
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return &comment.UpdateCommentResponse{
				Code:    uint32(appErr.Code),
				Message: appErr.Message,
			}, nil
		}
		return &comment.UpdateCommentResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_UPDATE_FAILED),
			Message: err.Error(),
		}, nil
	}

	recordAudit(ctx, user.AuditAction_AUDIT_ACTION_COMMENT_UPDATE, "comment", uint(req.CommentId), req.Content)
	return &comment.UpdateCommentResponse{
		Code:    uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message: "success",
		Comment: ConvertToProtoComment(c),
	}, nil
}

func (h *GrpcCommentHandler) DeleteComment(ctx context.Context, req *comment.DeleteCommentRequest) (*comment.DeleteCommentResponse, error) {
	uid, err := commonmiddleware.RequireGRPCAuth(ctx)
	if err != nil {
		return nil, err
	}
	// 从 ctx 提取角色，非普通用户（管理员/编辑）即视为有权限删除评论
	isAdmin := uint(0)
	if roleVal, ok := commonmiddleware.GetGRPCRole(ctx); ok {
		var role uint32
		switch v := roleVal.(type) {
		case float64:
			role = uint32(v)
		case int64:
			role = uint32(v)
		case uint32:
			role = v
		case uint:
			role = uint32(v)
		}
		if role != 1 { // 1=普通用户（constants.RoleNormal）
			isAdmin = 1
		}
	}
	err = h.Svc.DeleteComment(ctx, uint(req.CommentId), &model.DeleteCommentRequest{
		UserID:  uid,
		IsAdmin: isAdmin,
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return &comment.DeleteCommentResponse{
				Code:    uint32(appErr.Code),
				Message: appErr.Message,
			}, nil
		}
		return &comment.DeleteCommentResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_DELETE_FAILED),
			Message: err.Error(),
		}, nil
	}

	recordAudit(ctx, user.AuditAction_AUDIT_ACTION_COMMENT_DELETE, "comment", uint(req.CommentId), "")
	return &comment.DeleteCommentResponse{
		Code:    uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message: "success",
	}, nil
}

func (h *GrpcCommentHandler) ListComments(ctx context.Context, req *comment.ListCommentsRequest) (*comment.ListCommentsResponse, error) {
	comments, total, err := h.Svc.ListComments(ctx, &model.ListCommentsRequest{
		Page:   uint(req.Page),
		Size:   uint(req.PageSize),
		UserID: uint(req.UserId),
	})

	if err != nil {
		return &comment.ListCommentsResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_LIST_FAILED),
			Message: err.Error(),
		}, nil
	}

	protoComments := make([]*comment.Comment, len(comments))
	for i, c := range comments {
		protoComments[i] = ConvertToProtoComment(c)
	}

	return &comment.ListCommentsResponse{
		Code:     uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message:  "success",
		Comments: protoComments,
		Total:    uint32(total),
	}, nil
}

func (h *GrpcCommentHandler) GetArticleComments(ctx context.Context, req *comment.GetArticleCommentsRequest) (*comment.GetArticleCommentsResponse, error) {
	comments, total, commentEnabled, err := h.Svc.GetArticleComments(ctx, &model.GetArticleCommentsRequest{
		ArticleID:      uint(req.ArticleId),
		Page:           uint(req.Page),
		Size:           uint(req.PageSize),
		IncludeReplies: req.IncludeReplies,
		Sort:           req.Sort,
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return &comment.GetArticleCommentsResponse{
				Code:           uint32(appErr.Code),
				Message:        appErr.Message,
				CommentEnabled: false,
			}, nil
		}
		return &comment.GetArticleCommentsResponse{
			Code:           50001,
			Message:        err.Error(),
			CommentEnabled: false,
		}, nil
	}

	protoComments := make([]*comment.Comment, len(comments))
	for i, c := range comments {
		protoComments[i] = ConvertToProtoComment(c)
	}

	return &comment.GetArticleCommentsResponse{
		Code:           0,
		Message:        "success",
		Comments:       protoComments,
		Total:          uint32(total),
		CommentEnabled: commentEnabled,
	}, nil
}

func (h *GrpcCommentHandler) ReplyComment(ctx context.Context, req *comment.ReplyCommentRequest) (*comment.ReplyCommentResponse, error) {
	uid, err := commonmiddleware.RequireGRPCAuth(ctx)
	if err != nil {
		return nil, err
	}
	reply, err := h.Svc.ReplyComment(ctx, uint(req.CommentId), &model.ReplyCommentRequest{
		UserID:  uid,
		Content: req.Content,
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return &comment.ReplyCommentResponse{
				Code:    uint32(appErr.Code),
				Message: appErr.Message,
			}, nil
		}
		return &comment.ReplyCommentResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_INTERNAL_ERROR),
			Message: err.Error(),
		}, nil
	}

	recordAudit(ctx, user.AuditAction_AUDIT_ACTION_COMMENT_REPLY, "comment", uint(reply.ID), req.Content)
	return &comment.ReplyCommentResponse{
		Code:    0,
		Message: "success",
		Reply:   ConvertToProtoComment(reply),
	}, nil
}

func (h *GrpcCommentHandler) LikeComment(ctx context.Context, req *comment.LikeCommentRequest) (*comment.LikeCommentResponse, error) {
	uid, err := commonmiddleware.RequireGRPCAuth(ctx)
	if err != nil {
		return nil, err
	}
	likeCount, liked, err := h.Svc.LikeComment(ctx, uint(req.CommentId), &model.LikeCommentRequest{
		UserID: uid,
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return &comment.LikeCommentResponse{
				Code:    uint32(appErr.Code),
				Message: appErr.Message,
			}, nil
		}
		return &comment.LikeCommentResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_UPDATE_FAILED),
			Message: err.Error(),
		}, nil
	}

	recordAudit(ctx, user.AuditAction_AUDIT_ACTION_COMMENT_LIKE, "comment", uint(req.CommentId), "")
	return &comment.LikeCommentResponse{
		Code:      uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message:   "success",
		LikeCount: uint32(likeCount),
		Liked:     liked,
	}, nil
}

func (h *GrpcCommentHandler) GetCommentReplies(ctx context.Context, req *comment.GetCommentRepliesRequest) (*comment.GetCommentRepliesResponse, error) {
	replies, total, err := h.Svc.GetCommentReplies(ctx, &model.GetCommentRepliesRequest{
		CommentID: uint(req.CommentId),
		Page:      uint(req.Page),
		Size:      uint(req.PageSize),
	})

	if err != nil {
		return &comment.GetCommentRepliesResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_LIST_FAILED),
			Message: err.Error(),
		}, nil
	}

	protoReplies := make([]*comment.Comment, len(replies))
	for i, r := range replies {
		protoReplies[i] = ConvertToProtoComment(r)
	}

	return &comment.GetCommentRepliesResponse{
		Code:    uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message: "success",
		Replies: protoReplies,
		Total:   uint32(total),
	}, nil
}

func (h *GrpcCommentHandler) EnableComment(ctx context.Context, req *comment.EnableCommentRequest) (*comment.EnableCommentResponse, error) {
	if err := requireGRPCAdmin(ctx); err != nil {
		return nil, err
	}
	uid, _ := commonmiddleware.GetGRPCUserID(ctx)
	err := h.Svc.EnableComment(ctx, &model.EnableCommentRequest{
		UserID:    uid,
		ArticleID: uint(req.ArticleId),
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return &comment.EnableCommentResponse{
				Code:    uint32(appErr.Code),
				Message: appErr.Message,
			}, nil
		}
		return &comment.EnableCommentResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_ENABLE_FAILED),
			Message: err.Error(),
		}, nil
	}

	return &comment.EnableCommentResponse{
		Code:    uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message: "success",
	}, nil
}

func (h *GrpcCommentHandler) DisableComment(ctx context.Context, req *comment.DisableCommentRequest) (*comment.DisableCommentResponse, error) {
	if err := requireGRPCAdmin(ctx); err != nil {
		return nil, err
	}
	uid, _ := commonmiddleware.GetGRPCUserID(ctx)
	err := h.Svc.DisableComment(ctx, &model.DisableCommentRequest{
		UserID:    uid,
		ArticleID: uint(req.ArticleId),
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return &comment.DisableCommentResponse{
				Code:    uint32(appErr.Code),
				Message: appErr.Message,
			}, nil
		}
		return &comment.DisableCommentResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_DISABLE_FAILED),
			Message: err.Error(),
		}, nil
	}

	return &comment.DisableCommentResponse{
		Code:    uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message: "success",
	}, nil
}

// AdminListComments 管理端评论列表（仅管理员）。
func (h *GrpcCommentHandler) AdminListComments(ctx context.Context, req *comment.AdminListCommentsRequest) (*comment.AdminListCommentsResponse, error) {
	if err := requireGRPCAdmin(ctx); err != nil {
		return nil, err
	}
	comments, total, err := h.Svc.AdminListComments(ctx, uint(req.ArticleId), uint(req.UserId), req.Keyword, int(req.Page), int(req.PageSize))
	if err != nil {
		return &comment.AdminListCommentsResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_LIST_FAILED),
			Message: err.Error(),
		}, nil
	}
	protoComments := make([]*comment.Comment, len(comments))
	for i, c := range comments {
		protoComments[i] = ConvertToProtoComment(c)
	}
	return &comment.AdminListCommentsResponse{
		Code:     uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message:  "success",
		Comments: protoComments,
		Total:    uint32(total),
	}, nil
}

// AdminDeleteComment 管理端删除评论（无视作者，仅管理员）。
func (h *GrpcCommentHandler) AdminDeleteComment(ctx context.Context, req *comment.AdminDeleteCommentRequest) (*comment.AdminDeleteCommentResponse, error) {
	if err := requireGRPCAdmin(ctx); err != nil {
		return nil, err
	}
	if err := h.Svc.AdminDeleteComment(ctx, uint(req.CommentId)); err != nil {
		return &comment.AdminDeleteCommentResponse{
			Code:    uint32(comment.CommentErrorCode_COMMENT_DELETE_FAILED),
			Message: err.Error(),
		}, nil
	}
	recordAudit(ctx, user.AuditAction_AUDIT_ACTION_COMMENT_DELETE, "comment", uint(req.CommentId), "admin delete")
	return &comment.AdminDeleteCommentResponse{
		Code:    uint32(comment.CommentErrorCode_COMMENT_SUCCESS),
		Message: "success",
	}, nil
}

// requireGRPCAdmin 校验调用方已登录且具备管理员角色。
// 鉴权失败直接返回 gRPC 标准错误，由 Gateway 转换为 4xx。
func requireGRPCAdmin(ctx context.Context) error {
	if _, err := commonmiddleware.RequireGRPCAuth(ctx); err != nil {
		return err
	}
	raw, ok := commonmiddleware.GetGRPCRole(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "未认证")
	}
	var role uint8
	switch v := raw.(type) {
	case float64:
		role = uint8(v)
	case uint8:
		role = v
	case int:
		role = uint8(v)
	case int64:
		role = uint8(v)
	default:
		return status.Error(codes.PermissionDenied, "无效的角色信息")
	}
	if role != constants.RoleAdmin {
		return status.Error(codes.PermissionDenied, "需要管理员权限")
	}
	return nil
}

// recordAudit 上报操作审计（失败仅告警，不影响主流程）。
func recordAudit(ctx context.Context, action user.AuditAction, targetType string, targetID uint, detail string) {
	operatorID, _ := commonmiddleware.GetGRPCUserID(ctx)
	operator, _ := commonmiddleware.GetGRPCUsername(ctx)
	if err := client.RecordAudit(ctx, operatorID, operator, action, targetType, targetID, "", detail); err != nil {
		log.Printf("[audit] comment record failed: %v", err)
	}
}

// ConvertToProtoComment 转换为 proto 评论
func ConvertToProtoComment(c *model.Comment) *comment.Comment {
	if c == nil {
		return nil
	}

	username := ""
	nickname := ""
	avatar := ""
	if c.User.Username != "" {
		username = c.User.Username
		nickname = c.User.Nickname
		avatar = c.User.Avatar
	}

	replies := make([]*comment.Comment, len(c.Replies))
	for i, r := range c.Replies {
		replies[i] = ConvertToProtoComment(&r)
	}

	return &comment.Comment{
		Id:         uint32(c.ID),
		ArticleId:  uint32(c.ArticleID),
		UserId:     uint32(c.UserID),
		Username:   username,
		Nickname:   nickname,
		Avatar:     avatar,
		ParentId:   uint32(c.ParentID),
		Content:    c.Content,
		LikeCount:  uint32(c.LikeCount),
		ReplyCount: uint32(c.ReplyCount),
		Status:     uint32(c.Status),
		CreatedAt:  c.CreatedAt.Format(constants.DateTimeFormat),
		UpdatedAt:  c.UpdatedAt.Format(constants.DateTimeFormat),
		Replies:    replies,
	}
}
