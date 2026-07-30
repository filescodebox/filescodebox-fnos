# FileCodeBox 飞牛(fnOS)应用镜像
#
# 单容器单进程:fnos-adapter 二进制以库调用方式拉起 FileCodeBox 全部业务,
# 并挂载飞牛 Open API 适配层(SSO/共享目录/通知/内网穿透)。
#
# 构建上下文需同时包含本仓库与 FileCodeBox 仓库(因 go.mod 用 replace 指向 ../FileCodeBox)。
# 推荐在父目录构建:
#   docker build -f filecodebox-fnos/Dockerfile -t filecodebox-fnos:latest \
#     --build-arg VERSION=$(git describe --tags) .

# ========== Stage 1: 构建 fnos-adapter(含 FileCodeBox 库) ==========
FROM golang:1.26-alpine AS builder

# 构建依赖:原项目 SQLite 用纯 Go 驱动 glebarez/sqlite(无需 CGO);
# 此处保留 gcc/musl-dev 以备未来切换 CGO 驱动,不影响当前纯 Go 构建。
RUN apk add --no-cache gcc musl-dev sqlite-dev git ca-certificates tzdata

WORKDIR /workspace

# 先复制 FileCodeBox 后端(被 replace 指向的库)
COPY FileCodeBox/backend ./FileCodeBox/backend

# 再复制本仓库
COPY filecodebox-fnos ./filecodebox-fnos

WORKDIR /workspace/filecodebox-fnos

# 构建参数(版本信息)
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# 编译(静态链接 musl,CGO_ENABLED=1 for sqlite)
RUN CGO_ENABLED=1 go build \
    -ldflags="-w -s \
    -X 'main.Version=${VERSION}' \
    -X 'main.Commit=${COMMIT}' \
    -X 'main.BuildTime=${BUILD_TIME}'" \
    -o /out/fnos-adapter ./cmd/fnos-adapter

# ========== Stage 2: 运行时镜像 ==========
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata sqlite

# 非 root 用户(与原项目对齐 uid 1000)
RUN addgroup -g 1000 app && \
    adduser -D -s /bin/sh -u 1000 -G app app

WORKDIR /app

COPY --from=builder /out/fnos-adapter ./

# 运行时配置通过环境变量(FCB_* / FNOS_*)注入,与原项目 Dockerfile 一致,
# 无需内置配置文件。默认配置加载逻辑见 backend/cmd/server/bootstrap。

RUN mkdir -p data && chown -R app:app /app

USER app

EXPOSE 12345

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:12345/health || exit 1

CMD ["./fnos-adapter"]
