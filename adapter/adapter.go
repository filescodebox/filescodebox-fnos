package adapter

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/pigeonbox/core/pkg/logger"

	"github.com/pigeonbox/fnos/adapter/internal/fnosconfig"
	"github.com/pigeonbox/fnos/adapter/notify"
	"github.com/pigeonbox/fnos/adapter/sso"
	"github.com/pigeonbox/fnos/adapter/storage"
	"github.com/pigeonbox/fnos/adapter/tunnel"
)

// 为保持调用方(main.go)接口稳定,在此重导出配置类型与加载函数。
// 真正定义见 adapter/internal/fnosconfig。
type Config = fnosconfig.Config

// LoadConfig 从环境变量加载飞牛适配配置(转发至 fnosconfig 包)。
func LoadConfig() Config {
	return fnosconfig.LoadConfig()
}

// Mount 在已启动的 PigeonBox Hertz server 上挂载飞牛适配路由组。
//
// 设计要点:
//   - 不修改 PigeonBox 原有路由,飞牛能力以独立路由组 /api/fnos/* 注入。
//   - 凭证缺失(降级模式)时,/api/fnos/* 统一返回 503,提示"未配置飞牛凭证",
//     其余 API 正常工作。
//   - 凭证就绪时,各能力模块各自注册子路由。
func Mount(h *server.Hertz, cfg Config) {
	if !cfg.Enabled {
		// 降级模式:飞牛路由组统一返回 503。
		// 注意:Hertz 的 Use 中间件仅对已注册路由生效;若 group 下无任何具体路由,
		// 中间件不会被触发(请求将 404)。故降级模式必须注册一个 catch-all 路由,
		// 由其 handler 直接返回 503。
		group := h.Group("/api/fnos")
		group.Any("/*any", func(ctx context.Context, c *app.RequestContext) {
			c.JSON(consts.StatusServiceUnavailable, map[string]any{
				"code":    503,
				"message": "飞牛 Open API 未启用:" + cfg.DisabledReason,
			})
		})
		logger.Warn("飞牛适配路由组以降级模式挂载(/api/fnos/* 返回 503)")
		return
	}

	// 启用模式:各能力模块各自挂载子路由。
	group := h.Group("/api/fnos")

	// ① SSO 免登录。
	sso.Mount(group, cfg)
	// ② 共享目录列表。
	storage.Mount(group, cfg)
	// ③ 通知中心(以 client 形式供业务侧调用,无 HTTP 路由)。
	notify.Init(cfg)
	// ④ 内网穿透/外网分享(可选,本期桩)。
	tunnel.Mount(group, cfg)

	logger.Info("飞牛适配路由组已挂载:SSO / 共享目录 / 通知 / 内网穿透")
}
