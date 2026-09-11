package client

import (
	"context"
	"fmt"

	user "github.com/mysunshines/blog-user/proto/pb/v1"
	"github.com/mysunshines/gocommon/grpcclient"
)

// GetUser 按 userID 拉取用户信息（用于通知场景回填触发者昵称等）。
// 失败仅返回 error，调用方决定是否降级（如昵称留空）。
//
// 注意：下游别名 user.v1.UserService 由 consul resolver 解析为 Consul 服务名
// user-service，无需在此硬编码地址。
func GetUser(ctx context.Context, userID uint) (*user.User, error) {
	var resp user.GetUserResponse
	if err := grpcclient.SendRequest(ctx, user.UserService_GetUser_FullMethodName, &user.GetUserRequest{
		UserId: uint32(userID),
	}, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("get user failed: code=%d message=%s", resp.Code, resp.Message)
	}
	return resp.User, nil
}
