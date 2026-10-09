# PigeonBox 飞牛(fnOS)应用镜像
#
# 单容器单进程:fnos-adapter 二进制以库调用方式拉起 PigeonBox 全部业务,
# 并挂载飞牛 Open API 适配层(SSO/共享目录/通知/内网穿透),同端口服务内嵌前端。
#
# 构建上下文为本仓库即可(core/contracts 经 go.mod 正式版本从 module proxy 拉取;
# 前端自本仓 web/ 构建——2026-10-09 拆仓后适配器+构建自包含,依赖 frontend-core
# tgz 钉版在 web/package.json;架构无关,只在构建机原生平台跑一次)。
#   cd fnos && docker build -t fnos:latest .
# GOPROXY 可用 --build-arg GOPROXY=... 覆盖(默认国内加速;海外 CI 传空走默认)。

# ========== Stage 1: 前端构建产物(架构无关,BUILDPLATFORM 原生跑一次) ==========
# v1.2.7 起内嵌前端:fpk 桌面图标指向 http://<nas>:12345/,纯后端镜像只会给 404
# (2026-10-07 真机事故)。web/ 自包含构建(类型检查由 fnos CI 把守)。
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend

ARG NPM_REGISTRY=https://registry.npmjs.org
ENV NPM_CONFIG_REGISTRY=${NPM_REGISTRY}

WORKDIR /src
COPY web/package.json web/package-lock.json* ./
RUN npm ci --no-audit --no-fund
COPY web/ .
RUN npx vite build --outDir /frontend-dist --emptyOutDir

# ========== Stage 2: 构建 fnos-adapter(含 PigeonBox 库) ==========
FROM golang:1.26-alpine AS builder

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

# 构建依赖:原项目 SQLite 用纯 Go 驱动 glebarez/sqlite(无需 CGO);
# 此处保留 gcc/musl-dev 以备未来切换 CGO 驱动,不影响当前纯 Go 构建。
RUN apk add --no-cache gcc musl-dev sqlite-dev git ca-certificates tzdata

WORKDIR /workspace/fnos

# 先复制模块描述再下载依赖(利用层缓存)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 构建参数(版本信息)
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# 编译(静态链接 musl;CGO 仅预留,当前为纯 Go sqlite 驱动)
RUN CGO_ENABLED=1 GOWORK=off go build \
    -ldflags="-w -s \
    -X 'github.com/pigeonbox/kit/version.Version=${VERSION}' \
    -X 'github.com/pigeonbox/kit/version.BuildCommit=${COMMIT}' \
    -X 'github.com/pigeonbox/kit/version.BuildTime=${BUILD_TIME}'" \
    -o /out/fnos-adapter ./cmd/fnos-adapter

# ========== Stage 3: 运行时镜像 ==========
FROM alpine:3.24

# su-exec:entrypoint 以 root 修正数据卷归属后降权 uid 1000(见 entrypoint.sh)
RUN apk --no-cache add ca-certificates tzdata sqlite su-exec

# 应用用户(与原项目对齐 uid 1000);容器以 root 启动由 entrypoint 降权
RUN addgroup -g 1000 app && \
    adduser -D -s /bin/sh -u 1000 -G app app

WORKDIR /app

COPY --from=builder /out/fnos-adapter ./
# 随镜像携带默认配置:裸 docker run 的开箱可用性(open_upload/存储路径
# 在 core 无零值可用默认);fnOS 应用包模式由 compose env 覆盖同名项。
COPY configs/config.yaml ./configs/config.yaml
# 内嵌前端(WithStaticDir 指向此目录;目录缺失时 adapter 回退 core 默认并优雅降级)
COPY --from=frontend /frontend-dist /app/www
COPY entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/entrypoint.sh && mkdir -p data www && chown -R app:app /app

# wget --spider 发 HEAD,/health 未注册 HEAD 恒 404 → 健康检查永不通过,须显式 GET
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:12345/health || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]
EXPOSE 12345
