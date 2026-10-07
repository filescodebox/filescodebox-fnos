# fnos

[![CI](https://github.com/pigeonbox/fnos/actions/workflows/ci.yml/badge.svg)](https://github.com/pigeonbox/fnos/actions/workflows/ci.yml)
[![Tag](https://img.shields.io/github/v/tag/pigeonbox/fnos)](https://github.com/pigeonbox/fnos/tags)
[![License](https://img.shields.io/github/license/pigeonbox/fnos)](LICENSE)

> [PigeonBox](https://github.com/pigeonbox/pigeonbox)（文件快递柜）的飞牛 fnOS 应用适配层——单容器库式集成 PigeonBox 全部业务，包装为可在飞牛 NAS 应用中心安装的第三方应用，并接入飞牛 Open API。

> 🗂️ [PigeonBox 生态](https://github.com/orgs/pigeonbox)成员仓 · 应用包统一发布在 [hub 仓 Releases](https://github.com/pigeonbox/pigeonbox/releases)（`fnos-v*` 资产）

## 特性

| 能力 | 说明 | 状态 |
|------|------|------|
| **一键安装** | `fnpack` 标准应用包，应用中心托管启停/升级 | ✅ fnpack 1.2.3 打包通过 |
| **业务全功能** | 单进程库式调用 [core](https://github.com/pigeonbox/core) `bootstrap.Bootstrap()`，文本/文件分享、取件码、多云存储全部可用 | ✅ |
| **数据落 NAS** | 上传文件 + SQLite 全部落在用户可见的共享目录，文件管理器可直接查看/备份 | ✅ |
| **开箱即用** | JWT 密钥自动生成并持久化；默认免 Redis（core v0.14.0 单机内存模式，取件码映射存进程内、重启/升级失效）；安装向导收集可选凭证 | ✅ |
| **降级模式** | 未配置飞牛凭证时，飞牛集成关闭、业务完整运行 | ✅ |
| **SSO 免登录** | 飞牛账号一键登录映射为本系统用户 | 🔜 待凭证 |
| **通知中心** | 分享事件推送飞牛通知 | 🔜 待凭证 |
| **共享目录存储** | 读取 NAS 共享文件夹作为存储后端选项 | 🔜 待凭证 |

## 运行时架构（单容器单进程）

```
                ┌──────────────── fnos-adapter 二进制(容器入口)────────────────┐
                │                                                              │
  HTTP 12345 ──►│  bootstrap.Bootstrap() ──► *server.Hertz (PigeonBox 全业务)│
                │         │                              ▲                      │
                │         └── adapter.Mount(h, cfg) ─────┘ 挂载 /api/fnos/*     │
                │                          (SSO/目录/通知/穿透)                 │
                └──────────────────────────────────────────────────────────────┘
                     │                                            │
        /app/data ◄──┘ (上传文件+SQLite+JWT密钥)      取件码映射 ◄── 进程内内存
                     └────────────── ${TRIM_PKGVAR} ──────┘ (NAS 共享目录)
```

- **库式调用**：同进程 `import` + 函数调用 `bootstrap.Bootstrap()`，零跨进程开销
- **不侵入**：不修改 core 原有路由，飞牛能力以独立路由组 `/api/fnos/*` 注入（降级模式统一 503）

## 安装（飞牛 fnOS）

1. 从 hub 仓 [Releases](https://github.com/pigeonbox/pigeonbox/releases) 下载 `fnos-v*` 资产中的 `pigeonbox.fpk`（或自行 [打包](#打包fpk)）
2. 飞牛应用中心 → 手动安装 → 上传 fpk，按向导完成安装（飞牛凭证可留空，随时在应用设置补填）
3. 桌面入口打开即用；数据在应用数据目录（NAS 共享路径）下的 `data/`（上传文件/SQLite/JWT 密钥；取件码映射存进程内存，无独立文件）

> 要求：fnOS 设备可拉取 `ghcr.io/pigeonbox/fnos`（x86/ARM 均可，镜像多架构）。

## Docker 直接部署（不经飞牛应用中心）

```bash
docker run -d --name pigeonbox -p 12345:12345 \
  -v ./data:/app/data \
  -e FCB_SERVER_HOST=0.0.0.0 -e FCB_PRODUCTION=1 \
  ghcr.io/pigeonbox/fnos:1.14
```

## 打包 .fpk

```bash
# 安装 fnpack: https://developer.fnnas.com/docs/cli/fnpack/
cd fnos && fnpack build     # 产物 pigeonbox.fpk
```

`fnos/` 目录即应用包定义，**已对齐飞牛官方规范**（manifest / app/docker / app/ui 入口+图标 / cmd 生命周期脚本 / wizard 向导 / config 资源与权限）。CI 每次推送都会跑 `fnpack build` 校验并产出 fpk 构件。

注意：发版版本真相源=仓根 `VERSION` 文件(由 hub 发布列车 `scripts/release-train.sh bump` 统一维护)，`fnos/manifest` 的 `version` 与之同步,勿单手改一处。

## 本地开发

本仓已纳入 [pigeonbox](https://github.com/pigeonbox/pigeonbox) 装配仓的 `go.work`：

```bash
# 工作区内(hub 根 make setup 拉齐全部模块仓后):联编本地 core main
cd fnos && go build ./... && go test ./...

# 独立构建:钉 go.mod 正式版本(与 CI/Docker 一致)
GOWORK=off go test ./...

# 降级模式运行(业务全功能,飞牛能力关闭)
go run ./cmd/fnos-adapter

# 启用飞牛能力(需凭证)
FNOS_ENABLED=true FNOS_APPID=xxx FNOS_APPSECRET=yyy go run ./cmd/fnos-adapter
```

或使用 `make build / test / run / fpack / docker`。

## 目录结构

```
fnos/
├─ cmd/fnos-adapter/            容器入口:JWT 密钥引导 + 库式拉起 + 挂载 adapter
├─ adapter/                     飞牛 Open API 适配层(/api/fnos/*)
│  ├─ internal/fnosconfig/      配置加载(独立子包,无环)
│  ├─ internal/client/          飞牛 API 统一 HTTP client(签名/重试待文档)
│  └─ sso/ storage/ notify/ tunnel/   各能力模块(凭证就绪后填实)
├─ fnos/                        飞牛 .fpk 应用包定义(官方规范)
│  ├─ manifest                  应用元数据(version/platform/入口/端口)
│  ├─ app/docker/               容器编排(app 单服务,官方 TRIM_* 占位符)
│  ├─ app/ui/                   桌面入口(config)与图标(images/)
│  ├─ cmd/                      生命周期脚本(install/upgrade/uninstall/config)
│  ├─ wizard/                   安装/配置向导(飞牛凭证与 JWT 密钥)
│  └─ config/                   权限(privilege)与资源(resource: docker-project)
└─ Dockerfile                   多架构镜像构建(amd64/arm64;内嵌前端+entrypoint 降权)
```

> 前端内嵌（v1.2.7 起）：Dockerfile 现场构建 `pigeonbox/frontend` 产物入镜像
> `www/`，adapter 经 `bootstrap.WithStaticDir` 同端口服务 SPA——桌面图标打开即用。
> `FRONTEND_REF` 构建参数可钉前端分支/tag（默认 main）。

## 版本对应

| 本仓 | core | 说明 |
|------|------|------|
| v1.2.7 | v0.14.2 | 修复 fpk 真机安装失败（目录准备改尽力而为 + entrypoint 兜底归属）；镜像内嵌前端（桌面图标打开即用）；core 补文件/分片上传分享链接 0.0.0.0 漏网路径 |
| v1.2.6 | v0.14.1 | 跟随 core v0.14.1（env-only 缺 notifies 表修复；分享链接 0.0.0.0 修复） |
| v1.2.3 | v0.11.0 | 跟随 core v0.11.0（Cookie 会话）；默认关闭开放注册（管理员建号） |
| v1.2.2 | v0.10.0 | 跟随 core v0.10.0 安全审计加固；测试 -race 门禁；版本注入上收 kit/version |
| v1.2.1 | v0.8.0 | core 对齐（P2P 联邦接入；fnos 依赖面仅 bootstrap+logger，行为无变化） |
| v1.2.0 | v0.7.6 | 版本号与 desktop/charts 统一起始版；compose 镜像 tag 对齐 major.minor `:1.2` |
| v0.3.x | v0.7.6 | 仓改名 fnos——镜像路径切换 `ghcr.io/pigeonbox/fnos`，go module path 同步 |
| v0.2.x | v0.5.0 → v0.7.6 | 上传治理/多云存储/P0 修复；fnpack 规范化 + 内置 Redis + 向导；0.2.6 升 core v0.7.6 |

镜像：`ghcr.io/pigeonbox/fnos`（tag 跟随 Release；`0.2.x` 及更早镜像名为 `pigeonbox-fnos`、`0.1.x` 为 `pigeonbox-fnos`，均已冻结）。

## 路线图

- [ ] SSO：`sso/sso.go` ticket → 飞牛用户 → 本系统用户 → JWT（待飞牛凭证与 Open API 文档）
- [ ] 通知中心、共享目录存储、外网分享链接（同上）
- [ ] fnOS 真机全流程验证（安装→向导→升级→卸载）
- [ ] 飞牛应用中心上架（开发者后台未开放前经官方交流群提交）

## 相关仓库

[pigeonbox](https://github.com/pigeonbox/pigeonbox)（装配仓）· [core](https://github.com/pigeonbox/core)（业务核心）· [server](https://github.com/pigeonbox/server)（独立部署壳）· [frontend](https://github.com/pigeonbox/frontend) · [charts](https://github.com/pigeonbox/charts)（Helm）

## License

[Apache-2.0](LICENSE)
