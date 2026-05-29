package errors

import (
	"fmt"
)

// Error 自定义错误
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewError 创建错误
func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// 错误码定义 (区间: 40000-49999, 与 proto/comment.proto 中 CommentErrorCode 保持一致)
const (
	// 成功
	COMMENT_SUCCESS = 0

	// 业务错误 4000x
	ErrInternal      = 40011 // 内部错误
	ErrCreateFailed  = 40017 // 创建失败
	ErrUpdateFailed  = 40012 // 更新失败
	ErrDeleteFailed  = 40013 // 删除失败
	ErrListFailed    = 40014 // 列表获取失败
	ErrEnableFailed  = 40015 // 开启评论失败
	ErrDisableFailed = 40016 // 关闭评论失败

	// 资源/权限错误 4000x
	ErrNotFound        = 40001
	ErrCommentNotFound = 40018 // 评论未找到
	ErrForbidden       = 40020
)

// AppError AppError 是 Error 的别名，保持向后兼容
type AppError = Error

// New 创建错误
func New(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Internal 创建内部错误
func Internal(message string) *Error {
	return New(ErrInternal, message)
}

// NotFound 创建未找到错误
func NotFound(message string) *Error {
	return New(ErrNotFound, message)
}

// Forbidden 创建禁止错误
func Forbidden(message string) *Error {
	return New(ErrForbidden, message)
}

// BadRequest 创建请求错误
func BadRequest(message string) *Error {
	return New(40002, message)
}

// CommentDisabled 评论已关闭
func CommentDisabled() *Error {
	return New(40003, "评论已关闭")
}

// InBlacklist 用户在黑名单中
func InBlacklist() *Error {
	return New(40004, "您在黑名单中，无法评论")
}

// CommentCreateFailed 评论创建失败
func CommentCreateFailed(err error) *Error {
	return New(ErrCreateFailed, fmt.Sprintf("评论创建失败: %v", err))
}

// PermissionDenied 权限不足
func PermissionDenied() *Error {
	return New(ErrForbidden, "权限不足")
}

// AlreadyLiked 已点赞
func AlreadyLiked() *Error {
	return New(40005, "您已经点赞过该评论")
}
