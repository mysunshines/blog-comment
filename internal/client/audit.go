package client

import (
	"context"
	"fmt"

	"github.com/mysunshines/gocommon/grpcclient"
	user "github.com/mysunshines/blog-user/proto/pb"
)

// RecordAudit 记录后台审计日志（调 user-service 的 RecordLog）。
//
// 把 proto 请求拼装与错误映射收拢在此，调用方只传业务字段，无需关心
// user.RecordLogRequest 的字段细节。失败时返回 error（由调用方决定是否告警忽略）。
//
// 注意：下游别名 user.v1.UserService 由 consul resolver 解析为 Consul 服务名
// user-service，无需在此硬编码地址。
func RecordAudit(ctx context.Context, operatorID uint, operator string, action user.AuditAction,
	targetType string, targetID uint, targetTitle, detail string) error {

	var resp user.RecordLogResponse
	err := grpcclient.SendRequest(ctx, user.UserService_RecordLog_FullMethodName, &user.RecordLogRequest{
		OperatorId:  uint32(operatorID),
		Operator:    operator,
		Action:      action,
		TargetType:  targetType,
		TargetId:    uint32(targetID),
		TargetTitle: targetTitle,
		Detail:      detail,
	}, &resp)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("audit record failed: code=%d message=%s", resp.Code, resp.Message)
	}
	return nil
}
