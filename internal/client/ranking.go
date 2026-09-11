package client

import (
	"context"
	"fmt"
	"strconv"

	v0pb "github.com/mysunshines/blog-ranking/proto/pb/v0"
	pb "github.com/mysunshines/blog-ranking/proto/pb/v1"
	"github.com/mysunshines/gocommon/grpcclient"
)

// BoardArticleComments 评论数榜单键（member = 文章 ID，装饰器 = article-service）。
// 评论榜的「配置(RegisterBoard)」由 article-service 持有；本服务只负责推送分数变更。
const BoardArticleComments = "board:article:comments"

// IncrCommentScore 对文章评论数榜单增量推送（新增评论 +1 / 删除评论 -1）。best-effort。
func IncrCommentScore(ctx context.Context, articleID uint, delta float64) error {
	var resp pb.RecordScoreResponse
	if err := grpcclient.SendRequest(ctx, v0pb.RankingService_RecordScore_FullMethodName, &pb.RecordScoreRequest{
		Board:  BoardArticleComments,
		Member: strconv.FormatUint(uint64(articleID), 10),
		Op:     pb.ScoreOp_SCORE_OP_INCREMENT,
		Delta:  delta,
	}, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("record comment score failed: code=%d message=%s", resp.Code, resp.Message)
	}
	return nil
}
