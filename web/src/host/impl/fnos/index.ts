// fnOS(fnNAS)宿主适配器——PigeonBox 前端壳的平台实现。
//
// 接入模型(developer.fnnas.com 官方开放平台):
//   - 统一网关:页面经 /app/pigeonbox 跑在宿主 iframe,sdk.isWeb&&!sdk.isStandaloneWeb
//     为宿主态;manifest 已声明 micro_app=true,api-scope 见 fnos 仓 config/resource
//   - 后端能力探测:/api/fnos/capabilities(fnos 适配层,任何环境可探测)
//   - 非本平台部署:SDK 加载短路/探测失败,全部方法静默——能力按默认关闭上报
import type { HostAdapter, HostCapabilities, HostDirItem, HostFollowHandlers } from '@/host'
import type TrimApp from '@trimjs/web-app'
import type { user as userContract } from '@pigeonbox/contracts'
import { request } from '@/utils/request'

// ---- 后端能力探测(一次缓存) ----

interface FnosCapabilities {
  gateway: boolean
  sso: boolean
  shares: boolean
  trimapi: boolean
  prefix: string
  appName: string
  disabled: boolean
  reason: string
}

let capsPromise: Promise<FnosCapabilities | null> | null = null

function getFnosCapabilities(force = false): Promise<FnosCapabilities | null> {
  if (force || !capsPromise) {
    capsPromise = request<{ code: number; data: FnosCapabilities }>({
      url: '/api/fnos/capabilities',
      method: 'GET',
      timeout: 5000,
    })
      .then(
        (res: { code: number; data: FnosCapabilities } | null) =>
          res && res.code === 200 ? res.data : null
      )
      .catch(() => null)
  }
  return capsPromise
}

// ---- SDK 加载(仅宿主 iframe 内) ----

let sdkPromise: Promise<TrimApp | null> | null = null

function loadSdk(): Promise<TrimApp | null> {
  if (!sdkPromise) {
    sdkPromise = (async () => {
      // 廉价预检:非 iframe(独立浏览器直连)必然无宿主,不 import SDK
      if (typeof window === 'undefined' || window.parent === window) return null
      try {
        const mod = await import('@trimjs/web-app')
        return new mod.TrimApp()
      } catch (e) {
        console.debug('[fnos] SDK 加载失败(非宿主环境?)', e)
        return null
      }
    })()
  }
  return sdkPromise
}

function inFnosHost(sdk: TrimApp | null): boolean {
  return !!sdk && sdk.isWeb === true && sdk.isStandaloneWeb === false
}

/** 统一封装:能力门控 → 取 SDK → 宿主判定 → try/catch 静默 */
async function withSdk<T>(fn: (sdk: TrimApp) => Promise<T>): Promise<T | null> {
  const caps = await getFnosCapabilities()
  if (!caps?.gateway) return null
  const sdk = await loadSdk()
  if (!sdk || !inFnosHost(sdk)) return null
  try {
    return await fn(sdk)
  } catch (e) {
    console.debug('[fnos] 宿主调用失败', e)
    return null
  }
}

// ---- 适配器 ----

const capabilities: HostCapabilities = {
  sso: false,
  sharedDirs: false,
  fileActions: false,
  appSettings: false,
}

export const fnosHostAdapter: HostAdapter = {
  name: '飞牛',
  capabilities,

  async init() {
    const [caps, sdk] = await Promise.all([getFnosCapabilities(true), loadSdk()])
    const inHost = inFnosHost(sdk)
    Object.assign(capabilities, {
      sso: !!caps?.sso && inHost,
      sharedDirs: !!caps?.shares,
      fileActions: inHost,
      appSettings: inHost,
    } satisfies HostCapabilities)
  },

  initFollow(handlers: HostFollowHandlers): Promise<void> {
    return withSdk(async (sdk) => {
      try {
        const cfg = await sdk.getPlatformConfig()
        handlers.onTheme?.(cfg.theme === 'dark')
        handlers.onLanguage?.(cfg.language)
      } catch {
        /* 平台配置缺失不阻塞事件监听 */
      }
      try {
        // $on 仅 Web 宿主支持(withSdk 内已判定);老宿主静默
        await sdk.$on('os/theme', (theme: 'dark' | 'light') => handlers.onTheme?.(theme === 'dark'))
        await sdk.$on('os/language', (language: string) => handlers.onLanguage?.(language))
      } catch {
        /* no-op */
      }
    }).then(() => undefined)
  },

  setTitle(title: string): Promise<void> {
    return withSdk((sdk) => sdk.setTitle(title)).then(() => undefined)
  },

  async ssoLogin(): Promise<userContract.UserData | null> {
    const caps = await getFnosCapabilities()
    if (!caps?.sso) return null
    const sdk = await loadSdk()
    if (!sdk || !inFnosHost(sdk)) return null
    try {
      // 网关注入 X-Trim-* 身份头的前提是请求携带飞牛长效令牌(fnos-long-token);
      // web 会话 cookie(ost)不被网关认作凭据——先经宿主桥接换令牌
      await sdk.refreshToken()
    } catch {
      /* 宿主不支持换令牌时仍尝试 */
    }
    try {
      const res = await request<{
        code: number
        message: string
        data: { token: string; user: userContract.UserData }
      }>({
        url: '/api/fnos/login',
        method: 'POST',
        timeout: 8000,
      })
      return res.code === 200 ? res.data.user : null
    } catch {
      return null
    }
  },

  async listAuthorizedDirs(): Promise<HostDirItem[] | null> {
    const caps = await getFnosCapabilities()
    if (!caps?.shares) return null
    try {
      // convertPath 的 language 官方必传:显式带当前界面语言(部分 WebView 无
      // Accept-Language,缺省头会让后端跳过语义化转换)。语言取 html.lang
      // (core locale store 启动时同步),不反向 import core 内部模块
      const lang = document.documentElement.lang || 'zh-CN'
      const res = await request<{ code: number; message: string; data: HostDirItem[] }>({
        url: '/api/fnos/shares',
        method: 'GET',
        params: { lang },
        timeout: 8000,
      })
      return res.code === 200 ? (res.data ?? []) : null
    } catch {
      return null
    }
  },

  async pickAuthorizedDir(): Promise<string[] | null> {
    return withSdk((sdk) => sdk.pickSharedFile({ title: '选择授权目录', okText: '确认授权' })).then((r) => {
      const resp = r as { code: number; data?: string[] } | null
      return resp && resp.code === 0 && Array.isArray(resp.data) ? resp.data : null
    })
  },

  openFile(path: string): Promise<boolean> {
    return withSdk((sdk) => sdk.openFile(path)).then((r) => r !== null)
  },

  openDir(path: string): Promise<boolean> {
    return withSdk((sdk) => sdk.openFileManager(path)).then((r) => r !== null)
  },

  showFileDetails(paths: string[]): Promise<boolean> {
    return withSdk((sdk) => sdk.showFileDetails(paths)).then((r) => r !== null)
  },

  openAppSettings(): Promise<boolean> {
    return withSdk((sdk) => sdk.openAppSetting()).then((r) => r !== null)
  },

  openExternal(url: string, target = '_blank'): Promise<boolean> {
    return withSdk((sdk) => sdk.openURL(url, target)).then((r) => r !== null)
  },
}
