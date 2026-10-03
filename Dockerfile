# FileCodeBox 飞牛(fnOS)应用镜像
#
# 单容器单进程:fnos-adapter 二进制以库调用方式拉起 FileCodeBox 全部业务,
# 并挂载飞牛 Open API 适配层(SSO/共享目录/通知/内网穿透)。
#
# 构建上下文为本仓库即可(core/contracts 经 go.mod 正式版本从 module proxy 拉取)。
#   cd filecodebox-fnos && docker build -t filecodebox-fnos:latest .
# GOPROXY 可用 --build-arg GOPROXY=... 覆盖(默认国内加速;海外 CI 传空走默认)。

# ========== Stage 1: 构建 fnos-adapter(含 FileCodeBox 库) ==========
FROM golang:1.26-alpine AS builder

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

# 构建依赖:原项目 SQLite 用纯 Go 驱动 glebarez/sqlite(无需 CGO);
# 此处保留 gcc/musl-dev 以备未来切换 CGO 驱动,不影响当前纯 Go 构建。
RUN apk add --no-cache gcc musl-dev sqlite-dev git ca-certificates tzdata

WORKDIR /workspace/filecodebox-fnos

# 先复制模块描述再下载依赖(利用层缓存)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

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
# 随镜像携带默认配置:裸 docker run 的开箱可用性(open_upload/存储路径
# 在 core 无零值可用默认);fnOS 应用包模式由 compose env 覆盖同名项。
COPY configs/config.yaml ./configs/config.yaml

RUN mkdir -p data && chown -R app:app /app

USER app

EXPOSE 12345

# wget --spider 发 HEAD,/health 未注册 HEAD 恒 404 → 健康检查永不通过,须显式 GET
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:12345/health || exit 1

CMD ["./fnos-adapter"]
