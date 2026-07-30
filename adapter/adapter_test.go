package adapter

import (
	"context"
	"os"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/stretchr/testify/assert"

	fclogger "github.com/zy84338719/fileCodeBox/backend/api/pkg/logger"

	"github.com/zy84338719/filecodebox-fnos/adapter/internal/fnosconfig"
)

// TestMain 初始化最小 logger,避免 adapter.Mount 内 logger.Warn 在 logger 全局变量为 nil 时 panic。
func TestMain(m *testing.M) {
	_ = fclogger.Init(&fclogger.Config{Level: "error"}) // 测试只需 error 级,降噪
	os.Exit(m.Run())
}

// newTestEngine 建一个空 Hertz + 探针路由,挂载指定 cfg 的 adapter。
// 不调用 bootstrap.Bootstrap(),零 DB/Redis/JWT 副作用。
func newTestEngine(t *testing.T, cfg fnosconfig.Config) *server.Hertz {
	t.Helper()
	h := server.Default()
	// 探针路由:模拟 FileCodeBox 业务路由,验证 adapter 不影响它
	h.GET("/api/v1/ping", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, map[string]string{"msg": "pong"})
	})
	Mount(h, cfg)
	return h
}

// 降级模式:/api/fnos/* 全部返回 503。
func TestMount_DisabledMode_Returns503(t *testing.T) {
	cfg := fnosconfig.Config{Enabled: false, DisabledReason: "FNOS_ENABLED 未设为 true"}
	h := newTestEngine(t, cfg)

	w := ut.PerformRequest(h.Engine, consts.MethodPost, "/api/fnos/login", nil)
	resp := w.Result()
	assert.Equal(t, consts.StatusServiceUnavailable, resp.StatusCode(), "降级模式飞牛路由应返回 503")
}

// 降级模式:不影响业务路由。
func TestMount_DisabledMode_DoesNotAffectBusinessRoutes(t *testing.T) {
	cfg := fnosconfig.Config{Enabled: false}
	h := newTestEngine(t, cfg)

	w := ut.PerformRequest(h.Engine, consts.MethodGet, "/api/v1/ping", nil)
	resp := w.Result()
	assert.Equal(t, consts.StatusOK, resp.StatusCode(), "业务探针路由应正常 200")
}

// 启用模式:飞牛子路由应注册(桩返回非 404)。
func TestMount_EnabledMode_RegistersFnosRoutes(t *testing.T) {
	cfg := fnosconfig.Config{
		Enabled:   true,
		AppID:     "app1",
		AppSecret: "sec1",
		APIBase:   "https://api.example.com",
	}
	h := newTestEngine(t, cfg)

	// SSO 桩返回 501(未实现),但路由已注册,不应是 404
	w := ut.PerformRequest(h.Engine, consts.MethodPost, "/api/fnos/login", nil)
	resp := w.Result()
	assert.NotEqual(t, consts.StatusNotFound, resp.StatusCode(), "启用模式路由应已注册,不应 404")

	// 共享目录桩
	w2 := ut.PerformRequest(h.Engine, consts.MethodGet, "/api/fnos/shares", nil)
	assert.NotEqual(t, consts.StatusNotFound, w2.Result().StatusCode(), "shares 路由应已注册")
}
