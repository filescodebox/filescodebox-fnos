# wizard · 安装/配置向导

已按官方规范落地(docs/core-concepts/wizard/):向导收集的字段**不带前缀**直接成为
环境变量,注入生命周期脚本(cmd/*_callback 经 `$FNOS_APPID` 等直接读取)。

- `install` 安装时收集;`config` 安装后从应用设置随时改(两份字段一致)
- 字段即 env 契约(compose/adapter 直接消费,故未用 wizard_ 前缀):
  `FNOS_APPID` / `FNOS_APPSECRET`(留空=降级模式,分享功能完整)、
  `FCB_JWT_SECRET`(留空=adapter 自动生成并持久化到数据卷)
- `FNOS_ENABLED` 不向用户收集,由 callback 按凭证是否齐全推导
- 凭证经 cmd/install_callback、cmd/config_callback 写入 `app/docker/.env`
  (umask 077),compose 插值 `${FNOS_*:-}` 消费——官方未文档化向导变量是否
  直接参与 compose 插值,.env 是标准 Docker Compose 机制,确定性最高

## 待真机验证

- fnpack 打包通过,但向导 UI/`.env` 桥接/redis 数据卷归属(uid 999)需在
  fnOS 设备实测一轮安装→配置→升级流程
