package client

import (
	"context"
	"fmt"

	notification "github.com/mysunshines/blog-notification/proto/pb"
	"github.com/mysunshines/gocommon/grpcclient"
)

// CreateNotification 写入一条站内消息（调 notification-service 的 CreateMessage）。
//
// 语义化封装：调用方只传业务字段，无需关心 proto 拼装。通知是 best-effort，
// 失败仅返回 error，由调用方决定是否忽略（不应影响主业务成功路径）。
// 注意：下游别名 notification.v1.NotificationService 由 consul resolver 解析为
// Consul 服务名 notification-service，无需在此硬编码地址。
func CreateNotification(ctx context.Context, userID uint, typ notification.NotificationType,
	title, content, link string, actorID uint, actorName string) error {

	var resp notification.CreateMessageResponse
	err := grpcclient.SendRequest(ctx, notification.NotificationService_CreateMessage_FullMethodName,
		&notification.CreateMessageRequest{
			UserId:    uint64(userID),
			Type:      typ,
			Title:     title,
			Content:   content,
			Link:      link,
			ActorId:   uint64(actorID),
			ActorName: actorName,
		}, &resp)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("create notification failed: code=%d message=%s", resp.Code, resp.Message)
	}
	return nil
}
