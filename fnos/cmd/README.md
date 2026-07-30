# cmd · 生命周期脚本

飞牛应用生命周期脚本(install / uninstall / upgrade),参照 conversun/fnos-apps 各应用的 cmd/ 目录。

待凭证期按飞牛规范补全:
- `install`:初始化数据目录权限、生成默认配置
- `uninstall`:清理(默认保留用户数据于 ${TRIM_PKGVAR}/data)
- `upgrade`:版本迁移
