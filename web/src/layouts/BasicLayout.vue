<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useUserStore } from '@/stores/user'
import { useUIStore } from '@/stores/ui'
import { useSiteStore } from '@/stores/site'
import { APP_VERSION } from '@/version'
import type { MenuItem } from '@/api/auth'

const store = useUserStore()
const ui = useUIStore()
const site = useSiteStore()
const router = useRouter()
const route = useRoute()

const siteName = computed(() => site.get('site.name', 'GPT2API'))
const siteLogo = computed(() => site.get('site.logo_url', ''))

const { menu, user, role } = storeToRefs(store)
const collapsed = ref(false)      // 桌面端折叠状态
const drawerOpen = ref(false)     // 移动端抽屉展开状态
const isMobile = ref(false)
const loadingMenu = ref(false)

const MOBILE_BP = 768

function checkMobile() {
  const mobile = window.innerWidth < MOBILE_BP
  if (mobile !== isMobile.value) {
    isMobile.value = mobile
    if (!mobile) drawerOpen.value = false  // 切换到桌面时自动关抽屉
  }
}

// 顶栏汉堡按钮行为:移动端控制抽屉,桌面端控制折叠
function toggleSidebar() {
  if (isMobile.value) {
    drawerOpen.value = !drawerOpen.value
  } else {
    collapsed.value = !collapsed.value
  }
}

// 侧栏图标:移动端始终用 Menu 图标,桌面端跟随折叠状态
const menuIcon = computed(() => {
  if (isMobile.value) return 'Menu'
  return collapsed.value ? 'Expand' : 'Fold'
})

// 侧栏实际是否折叠(移动端抽屉展开时不折叠)
const sideCollapsed = computed(() => isMobile.value ? false : collapsed.value)

// 侧栏宽度(桌面端动态;移动端固定 240px 由 CSS 管理)
const asideWidth = computed(() => isMobile.value ? '0px' : (collapsed.value ? '64px' : '220px'))

const activePath = computed(() => route.path)

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

// 路由切换时关闭移动端抽屉
watch(() => route.path, () => {
  if (isMobile.value) drawerOpen.value = false
})

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
  loadMenu()
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
        text-color="#cfd3dc"
        active-text-color="#ffffff"
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
          <el-tooltip :content="ui.isDark ? '切换到亮色' : '切换到暗色'" placement="bottom">
            <el-button link class="theme-btn" @click="ui.toggleDark()">
              <el-icon :size="18">
                <component :is="ui.isDark ? 'Sunny' : 'Moon'" />
              </el-icon>
            </el-button>
          </el-tooltip>
          <el-dropdown trigger="click" @command="(c: string) => c === 'logout' ? logout() : goto(c)">
            <span class="user-entry">
              <el-avatar :size="28" style="background:#409eff">
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

      <el-main class="main" v-loading="loadingMenu">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>

    </el-container>
  </el-container>
</template>

<style scoped lang="scss">
// ─── 根容器 ──────────────────────────────────────────────────────────────────
.layout-root { height: 100vh; overflow: hidden; }

.right-container { min-width: 0; flex: 1; overflow: hidden; }

// ─── 侧栏 ────────────────────────────────────────────────────────────────────
.sidebar {
  background: var(--gp-sidebar-bg);
  transition: width .22s ease;
  overflow-x: hidden;
  display: flex !important;
  flex-direction: column;
  .side-menu {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
  }
}

// 移动端：侧栏脱离文档流变成 fixed overlay drawer
.sidebar-mobile {
  position: fixed !important;
  left: 0;
  top: 0;
  height: 100vh;
  width: 240px !important;  // 覆盖 :width 绑定
  z-index: 1001;
  transform: translateX(-100%);
  transition: transform .25s ease, box-shadow .25s ease;
  box-shadow: none;
}
.sidebar-mobile.sidebar-open {
  transform: translateX(0);
  box-shadow: 4px 0 24px rgba(0, 0, 0, 0.35);
}

// 移动端遮罩
.sidebar-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 1000;
}
.overlay-fade-enter-active, .overlay-fade-leave-active { transition: opacity .25s; }
.overlay-fade-enter-from, .overlay-fade-leave-to { opacity: 0; }

// ─── Logo ─────────────────────────────────────────────────────────────────────
.logo {
  height: 60px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  color: #fff;
  font-weight: 700;
  letter-spacing: 1px;
  flex-shrink: 0;
  .logo-img {
    width: 32px; height: 32px; border-radius: 8px;
    object-fit: contain; background: #fff;
  }
  .mark {
    display: inline-flex;
    width: 32px; height: 32px;
    border-radius: 8px;
    background: linear-gradient(135deg,#409eff,#67c23a);
    align-items: center; justify-content: center;
    font-size: 14px;
    flex-shrink: 0;
  }
  .title { font-size: 16px; white-space: nowrap; overflow: hidden; }
}

.side-menu {
  border-right: none !important;
  --el-menu-hover-bg-color: rgba(255,255,255,0.06);
}

// ─── 顶栏 ─────────────────────────────────────────────────────────────────────
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  min-height: 56px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--el-border-color-light);
  padding: 0 18px;
  flex-shrink: 0;
  .left { display: flex; align-items: center; gap: 10px; min-width: 0; }
  .crumb {
    font-size: 16px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .user-entry {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    color: var(--el-text-color-primary);
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
    gap: 10px;
    flex-shrink: 0;
  }
  .theme-btn { padding: 0 6px; }
}

// ─── 主区 ─────────────────────────────────────────────────────────────────────
.main {
  background: var(--gp-bg);
  padding: 0;
  overflow-y: auto;
  overflow-x: hidden;
}

// ─── 过渡 ─────────────────────────────────────────────────────────────────────
.fade-enter-active, .fade-leave-active { transition: opacity .15s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

// ─── 版本号 ───────────────────────────────────────────────────────────────────
.sidebar-version {
  position: sticky;
  bottom: 0;
  padding: 10px 16px;
  text-align: center;
  border-top: 1px solid rgba(255,255,255,0.07);
  background: var(--gp-sidebar-bg);
  flex-shrink: 0;
  .ver-text {
    display: inline-block;
    font-size: 11px;
    color: rgba(255,255,255,0.28);
    letter-spacing: 0.5px;
    user-select: none;
    white-space: nowrap;
  }
  &.collapsed .ver-text { font-size: 9px; letter-spacing: 0; }
}

// ─── 移动端补丁 ───────────────────────────────────────────────────────────────
@media (max-width: 767px) {
  .topbar {
    padding: 0 12px;
    .crumb { font-size: 14px; }
    .nick { display: none; }
  }
}
</style>
