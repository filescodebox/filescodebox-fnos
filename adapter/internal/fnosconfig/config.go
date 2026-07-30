// Package fnosconfig 定义飞牛适配层的配置类型与加载逻辑。
//
// 独立为子包以打破循环依赖:adapter 父包与 sso/storage/notify/tunnel 子模块
// 都引用本包的 Config,而本包不依赖任何 adapter 子包,从而无环。
package fnosconfig

import (
	"os"
)

// Config 是飞牛适配层的配置,全部来自环境变量。
type Config struct {
	// Enabled 飞牛能力总开关。AppID/AppSecret 均配置且 Enabled=true 时才真正启用。
	Enabled bool
	// AppID 飞牛应用 appid。
	AppID string
	// AppSecret 飞牛应用 secret。
	AppSecret string
	// APIBase 飞牛 Open API 基址。
	APIBase string
	// DisabledReason 降级原因(Enabled=false 时填,供日志输出)。
	DisabledReason string
}

// LoadConfig 从环境变量加载飞牛适配配置。
//
// 环境变量:
//   - FNOS_ENABLED   总开关("true"启用,其余视为关闭)
//   - FNOS_APPID     飞牛应用 appid
//   - FNOS_APPSECRET 飞牛应用 secret
//   - FNOS_API_BASE  飞牛 Open API 基址(缺省用官方地址占位,凭证就绪后确认)
func LoadConfig() Config {
	cfg := Config{
		AppID:     os.Getenv("FNOS_APPID"),
		AppSecret: os.Getenv("FNOS_APPSECRET"),
		APIBase:   os.Getenv("FNOS_API_BASE"),
	}
	if cfg.APIBase == "" {
		// 占位:飞牛官方 Open API 基址,凭证就绪后核实修正。
		cfg.APIBase = "https://open.fnnas.com"
	}

	// 降级判定:总开关关闭,或凭证缺失。
	if os.Getenv("FNOS_ENABLED") != "true" {
		cfg.Enabled = false
		cfg.DisabledReason = "FNOS_ENABLED 未设为 true"
		return cfg
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		cfg.Enabled = false
		cfg.DisabledReason = "FNOS_APPID / FNOS_APPSECRET 未配置"
		return cfg
	}

	cfg.Enabled = true
	return cfg
}
