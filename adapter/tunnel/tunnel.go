// Package tunnel 实现飞牛"内网穿透/外网分享"适配。
//
// 用途:分享生成时,可选调用飞牛内网穿透接口,为本分享生成本机端口的外网链接。
// 默认关闭,配置开启后生效。
//
// 本期为桩:真实映射待凭证就绪后补全。
package tunnel

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"

	"github.com/zy84338719/filecodebox-fnos/adapter/internal/fnosconfig"
)

// Mount 在飞牛路由组上挂载内网穿透子路由。
func Mount(g *route.RouterGroup, cfg fnosconfig.Config) {
	// 为指定分享生成外网链接。
	g.POST("/share/:code/tunnel", func(ctx context.Context, c *app.RequestContext) {
		// TODO(凭证就绪后实现):
		//   client.CreateTunnel(cfg, shareCode) → 外网链接
		c.JSON(consts.StatusNotImplemented, map[string]any{
			"code":    501,
			"message": "飞牛内网穿透尚未实现(凭证就绪后补全)",
		})
	})
}
