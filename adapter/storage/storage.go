// Package storage 实现飞牛"共享目录列表"适配。
//
// 用途:前端存储配置页用 GET /api/fnos/shares 填充"存储目录"下拉,
// 替代用户手填路径。
//
// 本期为桩:真实实现待凭证就绪后,调飞牛"列出共享目录"接口,
// 返回 [{name, path, capacity}]。
package storage

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"

	"github.com/pigeonbox/fnos/adapter/internal/fnosconfig"
)

// Mount 在飞牛路由组上挂载共享目录子路由。
func Mount(g *route.RouterGroup, cfg fnosconfig.Config) {
	g.GET("/shares", func(ctx context.Context, c *app.RequestContext) {
		// TODO(凭证就绪后实现):
		//   client.ListShares(cfg) → [{name, path, capacity}]
		//   返回飞牛 NAS 上的共享目录列表。
		c.JSON(consts.StatusNotImplemented, map[string]any{
			"code":    501,
			"message": "飞牛共享目录列表尚未实现(凭证就绪后补全)",
		})
	})
}
