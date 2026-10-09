package fnosconfig

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setEnvs 设置一批环境变量并注册清理(还原原值),确保测试隔离。
// 相关变量全集先清空再按 kv 注入,避免宿主环境污染用例。
func setEnvs(t *testing.T, kv map[string]string) {
	t.Helper()
	keys := []string{
		"TRIM_APPDEST", "TRIM_APPNAME",
		"PB_GATEWAY_SOCKET", "PB_GATEWAY_PREFIX", "PB_SERVER_PORT", "PB_TRIM_API_SOCKET",
	}
	for _, k := range keys {
		orig, had := os.LookupEnv(k)
		os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, orig)
			} else {
				_ = os.Unsetenv(k)
			}
		})
	}
	for k, v := range kv {
		_ = os.Setenv(k, v)
	}
}

// fakeTrimAPISocket 造一个真实 unix socket 文件(LoadConfig 以 ModeSocket 判定可用性)。
func fakeTrimAPISocket(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "apiscope.sock")
	ln, err := net.Listen("unix", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	return path
}

// 非 fnOS 环境(无 TRIM_* 变量、无官方 socket):全降级,业务不受影响。
func TestLoadConfig_NonFnosEnvironment(t *testing.T) {
	setEnvs(t, nil)
	cfg := LoadConfig()
	assert.False(t, cfg.GatewayEnabled, "无 TRIM_APPDEST 不应启用网关")
	assert.False(t, cfg.TrimAPIEnabled, "无官方 socket 不应启用 trimapi")
	assert.Equal(t, DefaultGatewayPrefix, cfg.GatewayPrefix)
	assert.Equal(t, DefaultAppName, cfg.AppName)
	assert.Equal(t, "12345", cfg.ServerPort)
	assert.Contains(t, cfg.DisabledReason, "TRIM_APPDEST")
}

// fnOS 原生环境:TRIM_APPDEST 注入 → 网关启用,socket 落 target 目录。
func TestLoadConfig_GatewayEnvironment(t *testing.T) {
	setEnvs(t, map[string]string{
		"TRIM_APPDEST": "/var/apps/pigeonbox/target",
		"TRIM_APPNAME": "pigeonbox",
	})
	cfg := LoadConfig()
	assert.True(t, cfg.GatewayEnabled)
	assert.Equal(t, "/var/apps/pigeonbox/target/app.sock", cfg.SocketPath)
	assert.False(t, cfg.TrimAPIEnabled)
	assert.Contains(t, cfg.DisabledReason, "官方后端 API")
}

// 全能力环境:网关 + 官方 API socket 俱在,降级原因为空。
func TestLoadConfig_FullEnabled(t *testing.T) {
	sock := fakeTrimAPISocket(t)
	setEnvs(t, map[string]string{
		"TRIM_APPDEST":       "/var/apps/pigeonbox/target",
		"PB_TRIM_API_SOCKET": sock,
	})
	cfg := LoadConfig()
	assert.True(t, cfg.GatewayEnabled)
	assert.True(t, cfg.TrimAPIEnabled)
	assert.Equal(t, sock, cfg.TrimAPISocket)
	assert.Empty(t, cfg.DisabledReason)
}

// 覆盖变量:PB_GATEWAY_SOCKET / PB_GATEWAY_PREFIX / PB_SERVER_PORT 均可注入。
func TestLoadConfig_Overrides(t *testing.T) {
	setEnvs(t, map[string]string{
		"TRIM_APPDEST":      "/var/apps/pigeonbox/target",
		"PB_GATEWAY_SOCKET": "/tmp/custom.sock",
		"PB_GATEWAY_PREFIX": "/app/mybox",
		"PB_SERVER_PORT":    "8080",
		"TRIM_APPNAME":      "mybox",
	})
	cfg := LoadConfig()
	assert.Equal(t, "/tmp/custom.sock", cfg.SocketPath)
	assert.Equal(t, "/app/mybox", cfg.GatewayPrefix)
	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "mybox", cfg.AppName)
}

// 普通(非 socket)文件不得误判为官方 API 可用。
func TestLoadConfig_RegularFileNotSocket(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "not-a-socket")
	require.NoError(t, os.WriteFile(regular, []byte("x"), 0o600))
	setEnvs(t, map[string]string{"PB_TRIM_API_SOCKET": regular})
	cfg := LoadConfig()
	assert.False(t, cfg.TrimAPIEnabled, "普通文件不应判为官方 API socket")
}
