<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useUserStore } from '@/stores/user'
import { useUIStore } from '@/stores/ui'
import { useSiteStore } from '@/stores/site'
import { APP_VERSION } from '@/version'
import type { MenuItem } from '@/api/auth'
import AnnouncementContent from '@/components/AnnouncementContent.vue'

const SITE_ANNOUNCEMENT_TITLE = 'site.announcement_title'
const SITE_ANNOUNCEMENT_HTML = 'site.announcement_html'
const SITE_ANNOUNCEMENT_POPUP_ENABLED = 'site.announcement_popup_enabled'
const SITE_ANNOUNCEMENT_VERSION = 'site.announcement_version'

const store = useUserStore()
const ui = useUIStore()
const site = useSiteStore()
const router = useRouter()
const route = useRoute()

const siteName = computed(() => site.siteName)
const siteLogo = computed(() => site.get('site.logo_url', ''))
const noticeTitle = computed(() => site.get(SITE_ANNOUNCEMENT_TITLE, '系统公告'))
const noticeHTML = computed(() => site.get(SITE_ANNOUNCEMENT_HTML, ''))
const noticePopupEnabled = computed(() => site.getBool(SITE_ANNOUNCEMENT_POPUP_ENABLED, false))
const noticeVersion = computed(() => Number(site.get(SITE_ANNOUNCEMENT_VERSION, '0')) || 0)
const hasAnnouncement = computed(() => !!(noticeTitle.value.trim() || noticeHTML.value.trim()))

const { menu, user, role } = storeToRefs(store)
const collapsed = ref(false)
const drawerOpen = ref(false)
const isMobile = ref(false)
const loadingMenu = ref(false)
const noticeDialogVisible = ref(false)
const noticeMuteToday = ref(false)
const noticeReadVersion = ref(0)
const noticeMutedToday = ref(false)

const MOBILE_BP = 768

const activePath = computed(() => route.path)
const isPersonalDashboard = computed(() => activePath.value === '/personal/dashboard')
const showAnnouncementButton = computed(() => isPersonalDashboard.value && hasAnnouncement.value)
const hasUnreadAnnouncement = computed(() => {
  if (!hasAnnouncement.value || noticeVersion.value <= 0) return false
  return noticeReadVersion.value < noticeVersion.value
})
const noticeUserKey = computed(() => String(user.value?.id || 0))
const noticeReadStorageKey = computed(() => `gpt2api.notice.read.${noticeUserKey.value}`)
const noticeMuteStorageKey = computed(() => `gpt2api.notice.mute.${noticeUserKey.value}`)
const noticeAutoShownSessionKey = computed(() => `gpt2api.notice.auto.${noticeUserKey.value}.${noticeVersion.value}`)

function checkMobile() {
  const mobile = window.innerWidth < MOBILE_BP
  if (mobile !== isMobile.value) {
    isMobile.value = mobile
    if (!mobile) drawerOpen.value = false
  }
}

function toggleSidebar() {
  if (isMobile.value) {
    drawerOpen.value = !drawerOpen.value
  } else {
    collapsed.value = !collapsed.value
  }
}

const menuIcon = computed(() => {
  if (isMobile.value) return 'Menu'
  return collapsed.value ? 'Expand' : 'Fold'
})
const sideCollapsed = computed(() => isMobile.value ? false : collapsed.value)
const asideWidth = computed(() => isMobile.value ? '0px' : (collapsed.value ? '64px' : '220px'))

const titleMap = computed(() => {
  const m = new Map<string, string>()
  function walk(items: MenuItem[]) {
    for (const it of items) {
      if (it.path) m.set(it.path, it.title)
      if (it.children) walk(it.children)
    }
  }
  walk(menu.value)
  return m
})

const currentTitle = computed(() => titleMap.value.get(activePath.value) || (route.meta.title as string) || '')

async function loadMenu() {
  if (menu.value.length > 0) return
  loadingMenu.value = true
  try {
    await store.fetchMenu()
  } finally {
    loadingMenu.value = false
  }
}

async function logout() {
  await store.logout()
  router.replace('/login')
}

function goto(path?: string) {
  if (path) router.push(path)
}

function todayToken() {
  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

function readStorageInt(key: string): number {
  const raw = localStorage.getItem(key)
  const n = Number(raw || 0)
  return Number.isFinite(n) ? n : 0
}

function parseMuteRecord(raw: string | null): { version: number; date: string } | null {
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw)
    if (typeof parsed?.version === 'number' && typeof parsed?.date === 'string') {
      return parsed as { version: number; date: string }
    }
  } catch {
    return null
  }
  return null
}

function syncAnnouncementState() {
  noticeReadVersion.value = readStorageInt(noticeReadStorageKey.value)
  const record = parseMuteRecord(localStorage.getItem(noticeMuteStorageKey.value))
  noticeMutedToday.value = !!record
    && record.version === noticeVersion.value
    && record.date === todayToken()
  if (!noticeDialogVisible.value) {
    noticeMuteToday.value = noticeMutedToday.value
  }
}

function markAnnouncementRead() {
  if (noticeVersion.value <= 0) return
  localStorage.setItem(noticeReadStorageKey.value, String(noticeVersion.value))
  noticeReadVersion.value = noticeVersion.value
}

function applyMutePreference() {
  const record = parseMuteRecord(localStorage.getItem(noticeMuteStorageKey.value))
  if (noticeMuteToday.value && noticeVersion.value > 0) {
    localStorage.setItem(noticeMuteStorageKey.value, JSON.stringify({
      version: noticeVersion.value,
      date: todayToken(),
    }))
  } else if (!record || record.version === noticeVersion.value) {
    localStorage.removeItem(noticeMuteStorageKey.value)
  }
  syncAnnouncementState()
}

function sessionAlreadyAutoShown() {
  return sessionStorage.getItem(noticeAutoShownSessionKey.value) === '1'
}

function openAnnouncement(auto = false) {
  if (!hasAnnouncement.value) return
  syncAnnouncementState()
  noticeDialogVisible.value = true
  markAnnouncementRead()
  if (auto) {
    sessionStorage.setItem(noticeAutoShownSessionKey.value, '1')
  }
}

function maybeAutoOpenAnnouncement() {
  syncAnnouncementState()
  if (!site.loaded || !showAnnouncementButton.value || !noticePopupEnabled.value) return
  if (!hasUnreadAnnouncement.value || noticeMutedToday.value || sessionAlreadyAutoShown()) return
  openAnnouncement(true)
}

watch(() => route.path, () => {
  if (isMobile.value) drawerOpen.value = false
})

watch(
  [
    () => route.path,
    () => user.value?.id,
    () => site.loaded,
    noticeVersion,
    noticeTitle,
    noticeHTML,
  ],
  () => {
    syncAnnouncementState()
    maybeAutoOpenAnnouncement()
  },
  { immediate: true },
)

watch(noticeDialogVisible, (open, wasOpen) => {
  if (open) {
    syncAnnouncementState()
    markAnnouncementRead()
    return
  }
  if (wasOpen) {
    applyMutePreference()
  }
})

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
  loadMenu()
  syncAnnouncementState()
})
onUnmounted(() => window.removeEventListener('resize', checkMobile))
watch(() => store.isLoggedIn, (v) => { if (v) loadMenu() })
</script>

<template>
  <el-container class="layout-root">
    <!-- 移动端遮罩层 -->
    <transition name="overlay-fade">
      <div v-if="isMobile && drawerOpen" class="sidebar-overlay" @click="drawerOpen = false" />
    </transition>

    <!-- 侧栏:桌面 inline / 移动端 fixed drawer -->
    <el-aside
      :width="asideWidth"
      class="sidebar"
      :class="{ 'sidebar-mobile': isMobile, 'sidebar-open': isMobile && drawerOpen }"
    >
      <div class="logo">
        <img v-if="siteLogo" :src="siteLogo" class="logo-img" alt="logo" />
        <span v-else class="mark">{{ (siteName[0] || 'G').toUpperCase() }}</span>
        <span v-if="!sideCollapsed" class="title">{{ siteName }}</span>
      </div>
      <el-menu
        :default-active="activePath"
        :collapse="sideCollapsed"
        background-color="transparent"
        :text-color="'var(--gp-sidebar-text-soft)'"
        :active-text-color="'var(--gp-sidebar-active-text)'"
        class="side-menu"
        router
      >
        <template v-for="group in menu" :key="group.key">
          <el-menu-item v-if="!group.children?.length && group.path" :index="group.path">
            <el-icon v-if="group.icon"><component :is="group.icon" /></el-icon>
            <template #title>{{ group.title }}</template>
          </el-menu-item>
          <el-sub-menu v-else-if="group.children?.length" :index="group.key">
            <template #title>
              <el-icon v-if="group.icon"><component :is="group.icon" /></el-icon>
              <span>{{ group.title }}</span>
            </template>
            <el-menu-item
              v-for="child in group.children"
              :key="child.key"
              :index="child.path!"
            >
              <el-icon v-if="child.icon"><component :is="child.icon" /></el-icon>
              <template #title>{{ child.title }}</template>
            </el-menu-item>
          </el-sub-menu>
        </template>
      </el-menu>

      <div class="sidebar-version" :class="{ collapsed: sideCollapsed }">
        <span class="ver-text">{{ sideCollapsed ? APP_VERSION.replace('v','') : APP_VERSION }}</span>
      </div>
    </el-aside>

    <el-container class="right-container">
      <el-header class="topbar">
        <div class="left">
          <el-button link @click="toggleSidebar">
            <el-icon :size="18"><component :is="menuIcon" /></el-icon>
          </el-button>
          <span class="crumb">{{ currentTitle }}</span>
        </div>
        <div class="right">
          <el-tooltip v-if="showAnnouncementButton" content="查看公告" placement="bottom">
            <el-badge :is-dot="hasUnreadAnnouncement" class="notice-badge">
              <el-button link class="theme-btn notice-btn" @click="openAnnouncement()">
                <el-icon :size="18"><Bell /></el-icon>
              </el-button>
            </el-badge>
          </el-tooltip>
          <el-tooltip :content="ui.isDark ? '切换到亮色' : '切换到暗色'" placement="bottom">
            <el-button link class="theme-btn" @click="ui.toggleDark()">
              <el-icon :size="18">
                <component :is="ui.isDark ? 'Sunny' : 'Moon'" />
              </el-icon>
            </el-button>
          </el-tooltip>
          <el-dropdown trigger="click" @command="(c: string) => c === 'logout' ? logout() : goto(c)">
            <span class="user-entry">
              <el-avatar :size="28" style="background:var(--gp-accent);color:#fff">
                {{ (user?.nickname || user?.email || 'U').slice(0, 1).toUpperCase() }}
              </el-avatar>
              <span class="nick">{{ user?.nickname || user?.email }}</span>
              <el-tag v-if="role === 'admin' && !isMobile" type="warning" size="small">管理员</el-tag>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="/personal/dashboard">
                  <el-icon><User /></el-icon> 个人中心
                </el-dropdown-item>
                <el-dropdown-item command="/personal/billing">
                  <el-icon><Wallet /></el-icon> 账单
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  <el-icon><SwitchButton /></el-icon> 退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="main">
        <div v-if="loadingMenu" class="main-loading-hint">正在加载菜单与页面框架…</div>
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>

      <el-dialog
        v-model="noticeDialogVisible"
        title="系统公告"
        width="720px"
        top="10vh"
        class="announcement-dialog"
      >
        <template #header>
          <div class="announcement-dialog-title">
            <span>{{ noticeTitle || '系统公告' }}</span>
            <el-tag v-if="noticeVersion > 0" size="small" effect="plain">版本 {{ noticeVersion }}</el-tag>
          </div>
        </template>
        <AnnouncementContent :html="noticeHTML" empty-text="暂无公告内容" />
        <template #footer>
          <div class="announcement-dialog-footer">
            <el-checkbox v-model="noticeMuteToday">今日不再提醒</el-checkbox>
            <el-button @click="noticeDialogVisible = false">关闭</el-button>
          </div>
        </template>
      </el-dialog>

    </el-container>
  </el-container>
</template>

<style scoped lang="scss">
.layout-root {
  height: 100vh;
  overflow: hidden;
  background: var(--gp-bg);
}

.right-container {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  background: transparent;
}

.sidebar {
  position: relative;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.02), transparent 14%),
    var(--gp-sidebar-bg);
  border-right: 1px solid var(--gp-sidebar-border);
  transition: width 0.22s ease;
  overflow-x: hidden;
  display: flex !important;
  flex-direction: column;

  .side-menu {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 8px 10px 16px;
  }
}

.sidebar-mobile {
  position: fixed !important;
  left: 0;
  top: 0;
  height: 100vh;
  width: 248px !important;
  z-index: 1001;
  transform: translateX(-100%);
  transition: transform 0.25s ease, box-shadow 0.25s ease;
  box-shadow: none;
}

.sidebar-mobile.sidebar-open {
  transform: translateX(0);
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.24);
}

.sidebar-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.28);
  z-index: 1000;
  backdrop-filter: blur(2px);
}

.overlay-fade-enter-active,
.overlay-fade-leave-active {
  transition: opacity 0.25s;
}

.overlay-fade-enter-from,
.overlay-fade-leave-to {
  opacity: 0;
}

.logo {
  height: 72px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 18px;
  color: var(--gp-sidebar-text);
  font-weight: 700;
  letter-spacing: 0.02em;
  flex-shrink: 0;

  .logo-img {
    width: 36px;
    height: 36px;
    border-radius: 12px;
    object-fit: contain;
    background: rgba(255, 255, 255, 0.92);
    box-shadow: 0 6px 16px rgba(31, 25, 19, 0.08);
  }

  .mark {
    display: inline-flex;
    width: 36px;
    height: 36px;
    border-radius: 12px;
    background: var(--gp-sidebar-surface);
    border: 1px solid var(--gp-sidebar-border);
    color: var(--gp-sidebar-text);
    align-items: center;
    justify-content: center;
    font-size: 14px;
    flex-shrink: 0;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.35);
  }

  .title {
    font-size: 15px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.side-menu {
  border-right: none !important;
  --el-menu-bg-color: transparent;
  --el-menu-hover-bg-color: var(--gp-sidebar-hover-bg);

  :deep(.el-menu-item),
  :deep(.el-sub-menu__title) {
    height: 44px;
    margin-bottom: 4px;
    border-radius: 12px;
    color: var(--gp-sidebar-text-soft);
    font-weight: 500;
  }

  :deep(.el-menu-item:hover),
  :deep(.el-sub-menu__title:hover) {
    color: var(--gp-sidebar-text);
    background: var(--gp-sidebar-hover-bg);
  }

  :deep(.el-menu-item.is-active) {
    color: var(--gp-sidebar-active-text);
    background: var(--gp-sidebar-active-bg);
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.22);
  }

  :deep(.el-sub-menu.is-active > .el-sub-menu__title) {
    color: var(--gp-sidebar-text);
  }

  :deep(.el-sub-menu .el-menu-item) {
    min-width: 0;
  }
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 72px;
  min-height: 72px;
  margin: 12px 12px 0;
  padding: 0 18px;
  background: rgba(255, 255, 255, 0.58);
  color: var(--el-text-color-primary);
  border: 1px solid var(--gp-border);
  border-radius: 18px;
  box-shadow: var(--gp-shadow-sm);
  backdrop-filter: blur(18px);
  flex-shrink: 0;

  .left {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }

  .crumb {
    font-size: 16px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    letter-spacing: 0.01em;
  }

  .user-entry {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    cursor: pointer;
    color: var(--el-text-color-primary);
    padding: 6px 8px;
    border-radius: 999px;

    .nick {
      font-size: 14px;
      max-width: 120px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .right {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .notice-badge {
    display: inline-flex;
    align-items: center;
  }

  .notice-badge :deep(.el-badge__content) {
    border: 0;
    box-shadow: none;
  }

  .theme-btn {
    padding: 0 6px;
  }
}

:deep(html.dark) .topbar {
  background: rgba(28, 25, 22, 0.72);
}

.main {
  background: transparent;
  padding: 12px 0 0;
  overflow-y: auto;
  overflow-x: hidden;
}

.main-loading-hint {
  margin: 0 12px 12px;
  padding: 10px 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  background: var(--el-fill-color-blank);
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

:deep(.announcement-dialog .el-dialog) {
  max-width: calc(100vw - 24px);
  border-radius: 22px;
}

:deep(.announcement-dialog .el-dialog__body) {
  padding-top: 8px;
}

.announcement-dialog-title,
.announcement-dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.sidebar-version {
  position: sticky;
  bottom: 0;
  padding: 12px 16px 16px;
  text-align: center;
  border-top: 1px solid var(--gp-sidebar-border);
  background: var(--gp-sidebar-bg);
  flex-shrink: 0;

  .ver-text {
    display: inline-block;
    font-size: 11px;
    color: var(--gp-sidebar-version-text);
    letter-spacing: 0.04em;
    user-select: none;
    white-space: nowrap;
  }

  &.collapsed .ver-text {
    font-size: 9px;
    letter-spacing: 0;
  }
}

@media (max-width: 767px) {
  .topbar {
    height: 64px;
    min-height: 64px;
    margin: 8px 8px 0;
    padding: 0 12px;

    .crumb {
      font-size: 14px;
    }

    .nick {
      display: none;
    }
  }

  .announcement-dialog-footer {
    flex-direction: column;
    align-items: flex-start;
  }

  .main {
    padding-top: 8px;
  }
}
</style>
