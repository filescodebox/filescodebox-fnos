// Package main 是飞牛(fnOS)应用的容器入口。
//
// 运行时形态:单容器单进程。本二进制以库调用方式拉起 FileCodeBox 全部业务
// (bootstrap.Bootstrap),并在同一进程内挂载飞牛 Open API 适配层(SSO/通知/目录/穿透)。
// 适配层以反向代理 + HTTP 中间件方式包裹 *server.Hertz,不修改其内部路由。
//
// 凭证缺失时进入降级模式:飞牛能力关闭,FileCodeBox 业务正常运行。
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/filescodebox/core/bootstrap"
	"github.com/filescodebox/core/pkg/logger"
	"github.com/filescodebox/fnos/adapter"
	"github.com/filescodebox/kit/version"

	"go.uber.org/zap"
)

// 版本信息:kit/version 包内变量,由 Dockerfile -ldflags -X 注入,缺省为 dev。

// ensureJWTSecret 保证 FCB_JWT_SECRET 存在:未显式配置时自动生成强密钥,
// 持久化到数据目录(.jwt_secret,权限 0600),重启复用(已签发 token 不失效)。
// 依据:core 的 validateSecrets 在 secret 缺失/弱值时拒绝启动(安全基线);
// NAS 场景用户不应被迫手工生成密钥,故在库拉起前注入。
func ensureJWTSecret() {
	if os.Getenv("FCB_JWT_SECRET") != "" {
		return
	}
	dataDir := os.Getenv("FCB_DATA_PATH")
	if dataDir == "" {
		dataDir = "./data"
	}
	secretPath := filepath.Join(dataDir, ".jwt_secret")
	if b, err := os.ReadFile(secretPath); err == nil {
		if s := strings.TrimSpace(string(b)); len(s) >= 32 {
			_ = os.Setenv("FCB_JWT_SECRET", s)
			return
		}
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		// 加密源不可用属系统级故障,直接失败
		_, _ = os.Stderr.WriteString("无法生成 JWT 密钥(crypto/rand 不可用): " + err.Error() + "\n")
		os.Exit(1)
	}
	secret := hex.EncodeToString(raw)
	_ = os.MkdirAll(dataDir, 0o755)
	if err := os.WriteFile(secretPath, []byte(secret), 0o600); err != nil {
		_, _ = os.Stderr.WriteString("无法持久化 JWT 密钥(" + secretPath + "): " + err.Error() + "\n")
		os.Exit(1)
	}
	_ = os.Setenv("FCB_JWT_SECRET", secret)
}

func main() {
	// --config 指定 FileCodeBox 配置文件路径(透传给 bootstrap)。
	configPath := flag.String("config", "", "FileCodeBox 配置文件路径(默认 configs/config.yaml)")
	// --static 前端静态资源目录:存在则以 WithStaticDir 注入 core(同端口服务
	// 内嵌前端 SPA);不存在保持 core 默认(./static,缺目录时优雅降级为纯 API)。
	staticDir := flag.String("static", "/app/www", "前端静态资源目录(不存在时回退 core 默认)")
	flag.Parse()

	// 注意:logger 必须先经 bootstrap.Init() 初始化后才能使用(logger 全局变量初始为 nil)。
	// 故飞牛适配配置的日志输出放到 bootstrap 之后。

	// 0. 保证 JWT 密钥存在(自动生成 + 数据卷持久化),必须在 bootstrap 读配置前注入。
	ensureJWTSecret()

	// 1. 以库调用方式拉起 FileCodeBox 全部业务。
	//    返回的 *server.Hertz 已完成:读配置→初始化logger→建DB→建storage→装路由→装中间件。
	bootOpts := make([]bootstrap.Option, 0, 1)
	if st, err := os.Stat(*staticDir); err == nil && st.IsDir() {
		bootOpts = append(bootOpts, bootstrap.WithStaticDir(*staticDir))
	}
	h, err := bootstrap.BootstrapWithOptions(*configPath, bootOpts...)
	if err != nil {
		// 此时 logger 可能未初始化,fallback 到标准错误输出。
		_, _ = os.Stderr.WriteString("FileCodeBox bootstrap 失败: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer bootstrap.Cleanup()

	// 2. 初始化飞牛适配配置(从环境变量读凭证,缺省=降级模式)。
	//    此时 logger 已就绪,可安全调用。
	fnosCfg := adapter.LoadConfig()
	if !fnosCfg.Enabled {
		logger.Warn("飞牛 Open API 适配层未启用(降级模式):业务正常运行,飞牛能力关闭",
			zap.String("reason", fnosCfg.DisabledReason))
	} else {
		logger.Info("飞牛 Open API 适配层已启用",
			zap.String("appid", fnosCfg.AppID),
			zap.String("api_base", fnosCfg.APIBase))
	}

	// 3. 装配飞牛适配层(在 Hertz server 上挂载 /api/fnos/* 路由组)。
	//    不影响 FileCodeBox 原有路由,飞牛能力以独立路由组注入。
	adapter.Mount(h, fnosCfg)

	// 4. 启动 HTTP 服务。
	go func() {
		logger.Info("FileCodeBox 飞牛应用启动中...",
			zap.String("version", version.Version),
			zap.String("commit", version.BuildCommit))
		h.Spin()
	}()

	// 5. 优雅退出:给在途请求 5s 排空窗口,超时强退(避免卡死容器停止流程,
	//    docker stop 默认 10s 后 SIGKILL,留足余量)。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Shutdown(shutdownCtx); err != nil {
		logger.Warn("优雅关闭超时,存在未完成的在途请求", zap.Error(err))
	}
	logger.Info("服务已停止")
}
