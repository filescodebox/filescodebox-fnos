package fnosconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setEnvs 设置一批环境变量并注册清理(还原原值),确保测试隔离。
func setEnvs(t *testing.T, kv map[string]string) {
	t.Helper()
	// 先清空所有相关变量,确保隔离;并注册还原。
	for _, k := range []string{"FNOS_ENABLED", "FNOS_APPID", "FNOS_APPSECRET", "FNOS_API_BASE"} {
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

func TestLoadConfig_DisabledByDefault(t *testing.T) {
	setEnvs(t, nil) // 全部缺失
	cfg := LoadConfig()
	assert.False(t, cfg.Enabled, "无任何配置应降级")
	assert.Equal(t, "https://open.fnnas.com", cfg.APIBase, "缺省 APIBase 应为占位地址")
	assert.Contains(t, cfg.DisabledReason, "FNOS_ENABLED", "降级原因应指向总开关")
}

func TestLoadConfig_EnabledButMissingSecret(t *testing.T) {
	setEnvs(t, map[string]string{"FNOS_ENABLED": "true", "FNOS_APPID": "app1"})
	cfg := LoadConfig()
	assert.False(t, cfg.Enabled, "开启但缺 secret 应降级")
	assert.Contains(t, cfg.DisabledReason, "APPSECRET", "降级原因应指向缺失的 secret")
}

func TestLoadConfig_Enabled(t *testing.T) {
	setEnvs(t, map[string]string{
		"FNOS_ENABLED":   "true",
		"FNOS_APPID":     "app1",
		"FNOS_APPSECRET": "sec1",
		"FNOS_API_BASE":  "https://api.example.com",
	})
	cfg := LoadConfig()
	assert.True(t, cfg.Enabled, "总开关开启且凭证齐全应启用")
	assert.Equal(t, "app1", cfg.AppID)
	assert.Equal(t, "sec1", cfg.AppSecret)
	assert.Equal(t, "https://api.example.com", cfg.APIBase)
	assert.Empty(t, cfg.DisabledReason, "启用时降级原因应为空")
}
