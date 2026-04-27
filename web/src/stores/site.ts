import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { fetchSiteInfo } from '@/api/settings'

/**
 * Site store 缓存站点公开信息:
 *   site.name / site.description / site.logo_url / site.footer / site.contact_email
 *   site.announcement_title / site.announcement_html / site.announcement_popup_enabled / site.announcement_version
 *   auth.allow_register   — 用于登录/注册页判定是否展示注册入口
 *
 * 页面启动时 refresh() 一次即可;管理员改完设置会再触发一次 refresh。
 * 不依赖 token,匿名也能拿。
 */
export const useSiteStore = defineStore('site', () => {
  const info = ref<Record<string, string>>({
    'site.name': 'GPT2API',
    'site.description': '企业级 OpenAI 兼容网关',
    'site.logo_url': '',
    'site.footer': '',
    'site.contact_email': '',
    'site.announcement_title': '',
    'site.announcement_html': '',
    'site.announcement_popup_enabled': 'false',
    'site.announcement_version': '0',
    'auth.allow_register': 'true',
  })
  const loaded = ref(false)

  const siteName = computed(() => info.value['site.name'] || 'GPT2API')

  async function refresh() {
    try {
      const d = await fetchSiteInfo()
      info.value = { ...info.value, ...d }
    } catch {
      // 静默失败,保持默认值。后端未起或权限中间件变化时,前端仍可渲染。
    } finally {
      loaded.value = true
      applyDocumentTitle()
      applyFavicon()
    }
  }

  function applyDocumentTitle() {
    document.title = `${siteName.value} 控制台`
  }

  function applyFavicon() {
    const url = info.value['site.logo_url']
    if (!url) return
    let link = document.querySelector<HTMLLinkElement>('link[rel~="icon"]')
    if (!link) {
      link = document.createElement('link')
      link.rel = 'icon'
      document.head.appendChild(link)
    }
    link.href = url
  }

  function get(key: string, fallback = ''): string {
    const v = info.value[key]
    return v == null || v === '' ? fallback : v
  }
  function getBool(key: string, fallback = false): boolean {
    const raw = get(key, fallback ? 'true' : 'false').toLowerCase()
    return raw === 'true' || raw === '1' || raw === 'yes' || raw === 'on'
  }
  function allowRegister(): boolean {
    return getBool('auth.allow_register', true)
  }

  return { info, loaded, refresh, get, getBool, allowRegister, siteName }
})
