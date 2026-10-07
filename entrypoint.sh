#!/bin/sh
# PigeonBox 飞牛应用镜像入口:数据目录归属兜底 + 降权运行。
#
# 为什么需要(2026-10-07 fnOS 真机回归):Docker 绑定挂载的宿主目录不存在时由
# 守护进程(root)代建,属主 root:root——容器内 uid 1000 无法写入(SQLite/JWT
# 密钥/上传文件全挂)。故以 root 起容器,仅做一次归属修正后立即降权,
# 业务进程恒以 uid 1000 运行(与既有数据卷语义一致)。
set -e

DATA_DIR=${FCB_DATA_PATH:-/app/data}

if [ "$(id -u)" = "0" ]; then
    mkdir -p "$DATA_DIR"
    if [ "$(stat -c %u "$DATA_DIR")" != "1000" ]; then
        chown -R 1000:1000 "$DATA_DIR"
    fi
    exec su-exec 1000:1000 /app/fnos-adapter "$@"
fi

# 非 root 运行(用户显式 --user):无权修正归属,直接运行,目录须调用方自备。
exec /app/fnos-adapter "$@"
