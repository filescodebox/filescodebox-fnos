# ui · 桌面入口定义

飞牛桌面应用的 UI 声明文件(参照 conversun/fnos-apps 各应用的 ui/ 目录)。

待凭证期按飞牛桌面规范补全:
- 桌面图标点击行为(`desktop_applaunchname = filecodebox.Application`)
- 打开方式:浏览器跳转 `http://<nas-ip>:${TRIM_SERVICE_PORT}`

manifest 中 `desktop_uidir = ui` 指向本目录。
