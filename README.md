# filescodebox-fnos

> [FileCodeBox](https://github.com/zy84338719/FileCodeBox) 的飞牛(fnOS)应用适配层 —— 把 FileCodeBox 当作库复用,包装为可在飞牛 NAS 应用商店上架的第三方应用,并接入飞牛 Open API。

---

## 它是什么

本项目是 **独立的 Go 模块**,通过 `go.mod` 的 `replace` 把 [FileCodeBox](../FileCodeBox) 当作库复用(运行时单进程库式调用 `bootstrap.Bootstrap()`),并在此之上叠加飞牛能力:

| 能力 | 说明 | 状态 |
|------|------|------|
| **SSO 免登录** | 飞牛账号一键登录,映射为本系统用户 | 接口桩(凭证就绪后补全) |
| **共享目录列表** | 读取 NAS 共享文件夹作为存储目录选项 | 接口桩 |
| **通知中心** | 推送到飞牛通知中心 | 接口桩 |
| **内网穿透/外网分享** | 为分享生成外网链接 | 接口桩 |
| **降级模式** | 无凭证时飞牛能力关闭,业务正常运行 | ✅ 已实现 |

> **凭证策略**:飞牛 Open API 需 `appid/appsecret`。本项目先抽象接口,凭证留作环境变量(`FNOS_APPID`/`FNOS_APPSECRET`)后填。缺省即降级模式。

## 运行时架构(单容器单进程)

```
                ┌──────────────── fnos-adapter 二进制(容器入口)────────────────┐
                │                                                              │
  HTTP 12345 ──►│  bootstrap.Bootstrap() ──► *server.Hertz (FileCodeBox 全业务) │
                │         │                              ▲                      │
                │         └── adapter.Mount(h, cfg) ─────┘ 挂载 /api/fnos/*     │
                │                          (SSO/目录/通知/穿透)                 │
                └──────────────────────────────────────────────────────────────┘
                                            │
                            /app/data  ◄──── ┴────►  NAS 共享文件夹(${TRIM_PKGVAR}/data)
                            (上传文件 + SQLite)
```

- **库式调用**:不通过 HTTP 调 FileCodeBox,而是同进程 `import` + 函数调用,零跨进程开销。
- **不侵入**:不修改 FileCodeBox 原有路由与业务逻辑,飞牛能力以独立路由组 `/api/fnos/*` 注入。

## 目录结构

```
filecodebox-fnos/
├─ cmd/fnos-adapter/main.go     容器入口:库式拉起 + 挂载 adapter
├─ adapter/                     飞牛 Open API 适配层
│  ├─ adapter.go                Mount:在 Hertz 上挂载 /api/fnos/*
│  ├─ internal/fnosconfig/      配置类型(独立子包,打破循环依赖)
│  ├─ sso/                      SSO 免登录
│  ├─ storage/                  共享目录列表
│  ├─ notify/                   通知中心
│  └─ tunnel/                   内网穿透
├─ fnos/                        飞牛 .fpk 打包定义
│  ├─ manifest                  应用元数据
│  ├─ config/privilege          运行权限声明
│  ├─ docker/docker-compose.yaml 飞牛容器编排(飞牛变量占位)
│  ├─ ui/ cmd/ wizard/          桌面入口/生命周期/安装向导(凭证期补全)
│  └─ FileCodeBox.sc            端口转发声明
└─ Dockerfile                   单容器构建
```

## 前置:原项目改造

FileCodeBox 的业务逻辑原在 `backend/internal/`,Go 的 internal 规则禁止跨模块 import。为支持库式复用,原项目已做**机械改造**:`backend/internal` → `backend/api`(仅目录改名 + import 路径替换,零业务逻辑变更,`go build/vet/test` 全绿)。

两个仓库需置于同级目录:
```
my_project/
├─ FileCodeBox/          (原项目,已改造 internal→api)
└─ filecodebox-fnos/     (本项目)
```

## 本地开发

```bash
# 在 filecodebox-fnos 目录
go build ./cmd/fnos-adapter      # 编译验证(已通过)
go run ./cmd/fnos-adapter         # 降级模式启动(FileCodeBox 全功能可用)

# 启用飞牛能力(需凭证)
FNOS_ENABLED=true FNOS_APPID=xxx FNOS_APPSECRET=yyy go run ./cmd/fnos-adapter
```

## 构建 Docker 镜像

构建上下文需同时包含两个仓库(因 replace 指向 `../FileCodeBox`):
```bash
cd my_project   # 父目录
docker build -f filecodebox-fnos/Dockerfile -t filecodebox-fnos:latest .
```

## 飞牛应用打包

`fnos/` 目录即 `.fpk` 包定义,参照 [conversun/fnos-apps](https://github.com/conversun/fnos-apps) 真实仓库的格式:
- `manifest`:应用元数据(`source = thirdparty`,`service_port = 12345`)
- `docker/docker-compose.yaml`:用 `${TRIM_PKGVAR}` 挂载 NAS 共享文件夹到 `/app/data`
- `config/privilege`:运行权限(`run-as: package`)
- `ui/cmd/wizard`:桌面入口与生命周期脚本(待凭证期按飞牛规范补全)

> 打包成 `.fpk` 需飞牛官方打包工具(见飞牛开发者文档)。`fnos/` 目录结构已就绪。

## 待办(凭证就绪后)

- [ ] SSO:`sso/sso.go` 实现 ticket→飞牛用户→本系统用户→JWT
- [ ] 共享目录:`storage/storage.go` 调飞牛"列出共享目录"接口
- [ ] 通知:`notify/notify.go` 推送到飞牛通知中心
- [ ] 内网穿透:`tunnel/tunnel.go` 生成分享外网链接
- [ ] `fnos/ui` `fnos/cmd` `fnos/wizard` 按飞牛桌面规范补全
- [ ] 飞牛 Open API 签名算法(凭证下发后从文档核实)
