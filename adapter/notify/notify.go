// Package notify 实现飞牛"通知中心"适配。
//
// 设计:以独立 client 形式供 FileCodeBox 通知模块调用(非 HTTP 路由)。
// 不侵入 FileCodeBox 通知抽象:adapter 侧维护一个 fnos channel,
// 业务侧配置通知通道为 "fnos" 时,经此 client 推送到飞牛通知中心。
//
// 本期为桩:Send 的真实推送待凭证就绪后补全。
package notify

import (
	"github.com/filescodebox/core/pkg/logger"
	"go.uber.org/zap"

	"github.com/zy84338719/filecodebox-fnos/adapter/internal/fnosconfig"
)

// Init 初始化飞牛通知 channel。
func Init(cfg fnosconfig.Config) {
	logger.Info("飞牛通知中心 channel 已初始化(桩:凭证就绪后补全推送实现)")
}

// Send 向飞牛通知中心推送一条消息。
// 业务侧在通知配置为 "fnos" 通道时调用。
// TODO(凭证就绪后实现):client 推送到飞牛通知中心接口。
func Send(cfg fnosconfig.Config, title, content string) error {
	logger.Warn("飞牛通知推送尚未实现(桩)",
		zap.String("title", title))
	return nil
}
