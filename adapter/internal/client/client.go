// Package client 是飞牛(fnOS)Open API 的统一 HTTP client。
//
// 职责:签名注入、请求重试、超时控制、错误码映射。
// 所有飞牛 API 调用经此 client,便于凭证注入与降级判断。
//
// 注意:签名算法与真实端点待飞牛官方 Open API 文档到手后填实(见 Do 内 TODO)。
package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zy84338719/filecodebox-fnos/adapter/internal/fnosconfig"
)

// Client 飞牛 Open API client。
type Client struct {
	cfg  fnosconfig.Config
	http *http.Client
	// TODO(凭证+文档到手):签名密钥/nonce 缓存等
}

// New 创建飞牛 client。降级模式(cfg.Enabled=false)返回 nil。
func New(cfg fnosconfig.Config) *Client {
	if !cfg.Enabled {
		return nil
	}
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Do 统一请求入口。
//
// method/path/body 由各能力模块构造;返回响应体、HTTP 状态码与 error。
// HTTP 层成功(收到响应)即返回 nil error;业务错误码(非 2xx)由调用方按 status 判断。
//
// TODO(凭证+文档到手):
//  1. 按飞牛文档实现签名(注入 appid/签名/timestamp 到 header 或 query)
//  2. 实现重试(网络错误重试 2 次)
//  3. 映射飞牛错误码
func (c *Client) Do(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	url := c.cfg.APIBase + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("构造请求失败: %w", err)
	}
	// TODO(文档到手):在此注入飞牛要求的签名 header
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求飞牛失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("读取响应失败: %w", err)
	}
	return data, resp.StatusCode, nil
}
