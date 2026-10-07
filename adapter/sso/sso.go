// Package sso 实现飞牛 SSO 免登录适配。
//
// 流程:前端拿飞牛授权 ticket → POST /api/fnos/login {ticket}
// → adapter 校验 ticket 调飞牛"换取用户信息" → 查/建本系统用户
// → 签发本系统 JWT → 返回 token。
//
// 本期为接口抽象桩:ticket 校验/用户映射/JWT 签发的真实实现,
// 待飞牛凭证就绪后按飞牛 Open API 文档补全(见 TODO 注释)。
package sso

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"

	"github.com/pigeonbox/core/pkg/logger"

	"github.com/pigeonbox/fnos/adapter/internal/fnosconfig"
)

// Mount 在飞牛路由组上挂载 SSO 子路由。
func Mount(g *route.RouterGroup, cfg fnosconfig.Config) {
	g.POST("/login", func(ctx context.Context, c *app.RequestContext) {
		// TODO(凭证就绪后实现):
		//   1. 解析请求体 ticket
		//   2. client.ExchangeTicket(cfg, ticket) → 飞牛用户信息(uid/nickname)
		//   3. 按飞牛 uid 查/建本系统用户(复用 api/app/user 或 api/repo/db)
		//   4. 用 api/pkg/auth 签发本系统 JWT
		//   5. 返回 {token, user}
		logger.Warn("飞牛 SSO 登录尚未实现(凭证就绪后补全),ticket 已收到")
		c.JSON(consts.StatusNotImplemented, map[string]any{
			"code":    501,
			"message": "飞牛 SSO 免登录尚未实现,请使用账号密码登录",
		})
	})
}
