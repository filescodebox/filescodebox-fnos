// Package storage 实现飞牛"应用共享授权目录"适配(2026-10-09 真实现)。
//
// 官方模型(developer.fnnas.com/api/authorization/shared-access):
// 管理员经 JS SDK pickSharedFile 选择器为应用授权目录;应用后端用
// trim.file.getSharedAccessibleFolders 查询授权结果,trim.file.convertPath
// 把 /vol1/... 内部路径转成用户可读展示路径。
//
// 用途:前端存储配置页 GET /api/fnos/shares 填充"存储目录"下拉/快捷选择,
// 替代用户手填路径;目录选择器动作由前端 JS SDK 直调(经网关同源页面)。
package storage

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"go.uber.org/zap"

	"github.com/pigeonbox/core/pkg/logger"

	"github.com/pigeonbox/fnos/adapter/internal/fnosconfig"
	"github.com/pigeonbox/fnos/adapter/internal/trimapi"
)

// ShareItem 授权目录项:path=内部路径(业务可直用),semantic=展示路径。
type ShareItem struct {
	Path     string `json:"path"`
	Semantic string `json:"semantic"`
}

// Mount 在飞牛路由组上挂载共享目录子路由。client=nil(trimapi 不可用)时 503。
func Mount(g *route.RouterGroup, cfg fnosconfig.Config, client *trimapi.Client) {
	g.GET("/shares", func(ctx context.Context, c *app.RequestContext) {
		if client == nil {
			c.JSON(consts.StatusServiceUnavailable, map[string]any{
				"code":    503,
				"message": "飞牛官方 API 不可用:" + cfg.DisabledReason,
			})
			return
		}
		paths, err := client.SharedFolders(ctx)
		if err != nil {
			logger.Warn("查询共享授权目录失败", zap.Error(err))
			c.JSON(consts.StatusBadGateway, map[string]any{
				"code":    502,
				"message": "查询授权目录失败:" + err.Error(),
			})
			return
		}
		items := make([]ShareItem, 0, len(paths))
		for _, p := range paths {
			items = append(items, ShareItem{Path: p, Semantic: p})
		}
		// 展示路径装饰:失败不阻塞(path 兜底原样返回)。
		if lang := displayLanguage(c); lang != "" {
			if converted, err := client.ConvertPaths(ctx, lang, paths); err == nil {
				byPath := make(map[string]string, len(converted))
				for _, cv := range converted {
					byPath[cv.Path] = cv.SemanticPath
				}
				for i := range items {
					if s, ok := byPath[items[i].Path]; ok && s != "" {
						items[i].Semantic = s
					}
				}
			}
		}
		c.JSON(consts.StatusOK, map[string]any{
			"code": 200,
			"data": items,
		})
	})
}

// displayLanguage 展示语言:?lang= 优先,回退 Accept-Language 首段,
// 均无时返回空串(跳过转换,直接展示内部路径)。
func displayLanguage(c *app.RequestContext) string {
	if v := strings.TrimSpace(string(c.QueryArgs().Peek("lang"))); v != "" {
		return v
	}
	al := string(c.GetHeader("Accept-Language"))
	if al == "" {
		return ""
	}
	first := strings.SplitN(al, ",", 2)[0]
	first = strings.SplitN(first, ";", 2)[0]
	return strings.TrimSpace(first)
}
