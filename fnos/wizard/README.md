# wizard · 安装向导(待补)

飞牛应用安装向导,向导收集的字段会**不带前缀**直接成为环境变量注入应用进程与 compose
(见 developer.fnnas.com/docs/core-concepts/environment-variables/)。

当前 app/docker/docker-compose.yaml 依赖以下向导变量(均留空可降级):

- `FNOS_ENABLED` / `FNOS_APPID` / `FNOS_APPSECRET`:飞牛 Open API 凭证,
  缺省=降级模式(业务全功能,飞牛 SSO/通知关闭)
- `FCB_JWT_SECRET`:JWT 密钥,缺省时 adapter 自动生成强密钥并持久化到数据卷

端口无需向导:由 manifest.service_port(12345)固定并经 ${TRIM_SERVICE_PORT} 注入。

按官方向导规范(docs/core-concepts/wizard/)补全后,本目录需提供向导定义文件。
