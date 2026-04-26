<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useUIStore } from '@/stores/ui'
import { useSiteStore } from '@/stores/site'

const router = useRouter()
const user = useUserStore()
const ui = useUIStore()
const site = useSiteStore()

const siteName = computed(() => site.get('site.name', 'GPT2API'))
const siteLogo = computed(() => site.get('site.logo_url', ''))
const allowRegister = computed(() => site.allowRegister())
const loggedIn = computed(() => user.isLoggedIn)

function goPlay() {
  if (loggedIn.value) router.push('/personal/play')
  else router.push('/login?redirect=/personal/play')
}
function goDashboard() { router.push('/personal/dashboard') }
function goLogin() { router.push('/login') }
function goRegister() { router.push('/register') }
function scrollTop() { window.scrollTo({ top: 0, behavior: 'smooth' }) }

// 滚动监听,nav 加实体背景
const scrolled = ref(false)
onMounted(() => {
  const onScroll = () => { scrolled.value = window.scrollY > 24 }
  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll()
})

// 三张卖点卡,全部围绕"图"
const features = [
  {
    icon: 'MagicStick',
    color: '#409eff',
    title: 'IMG2 正式版直出',
    desc: '全面对齐 <code>picture_v2</code> 正式协议,SSE 够数即返回,60s 短轮询补齐。<b>速度优先 · 不悄悄重试</b>,出错第一时间暴露给调用方。',
  },
  {
    icon: 'Picture',
    color: '#a855f7',
    title: '批量 · 多比例 · 预设',
    desc: '10 种常用宽高比一键切换(21:9 / 16:9 / 4:3 / 1:1 / 9:16 …),<b>N 张批量成图</b>,提示词预设库,浏览器里直接出图。',
  },
  {
    icon: 'Connection',
    color: '#6f685e',
    title: 'OpenAI 零改造接入',
    desc: '<code>/v1/images/generations</code> · <code>/v1/images/edits</code> 原样对齐官方 SDK,<b>切网关只改 base_url</b>,一行代码即可接入。',
  },
]
</script>

<template>
  <div class="landing" :class="{ dark: ui.isDark }">
    <!-- ============= 顶部导航 ============= -->
    <header class="nav" :class="{ scrolled }">
      <div class="nav-inner">
        <a class="logo" @click="scrollTop">
          <img v-if="siteLogo" :src="siteLogo" class="logo-img" alt="logo" />
          <span v-else class="logo-mark">{{ (siteName[0] || 'G').toUpperCase() }}</span>
          <span class="logo-name">{{ siteName }}</span>
        </a>
        <div class="nav-actions">
          <el-button
            link :title="ui.isDark ? '切换到亮色' : '切换到暗色'"
            class="theme-btn" @click="ui.toggleDark()"
          >
            <el-icon :size="18"><component :is="ui.isDark ? 'Sunny' : 'Moon'" /></el-icon>
          </el-button>
          <template v-if="!loggedIn">
            <el-button text class="btn-login" @click="goLogin">登录</el-button>
            <el-button v-if="allowRegister" type="primary" round @click="goRegister">免费注册</el-button>
          </template>
          <template v-else>
            <el-button type="primary" round @click="goDashboard">
              进入控制台 <el-icon><ArrowRight /></el-icon>
            </el-button>
          </template>
        </div>
      </div>
    </header>

    <!-- ============= Hero:只讲 GPT IMAGE2 出图 ============= -->
    <section id="hero" class="hero">
      <div class="hero-bg"></div>
      <div class="hero-inner">
        <div class="hero-text">
          <div class="eyebrow">
            <span class="dot"></span>
            gpt-image-2 · 官方级终稿直出
          </div>
          <h1 class="hero-title">
            <span class="gradient-text">GPT IMAGE2</span><br/>
            一键出高清终稿
          </h1>
          <p class="hero-sub">
            基于 chatgpt.com 逆向的 <b>gpt-image-2</b> 网关<br/>
            <b>IMG2 终稿直出</b> · 多比例 / 批量 N 张 / OpenAI SDK 零改造
          </p>
          <div class="hero-cta">
            <el-button size="large" type="primary" round @click="goPlay">
              <el-icon><VideoPlay /></el-icon> 立即体验在线生图
            </el-button>
          </div>
        </div>

        <div class="hero-preview">
          <div class="preview-glow"></div>
          <div class="preview-frame">
            <div class="frame-bar">
              <span class="dot red"></span>
              <span class="dot yellow"></span>
              <span class="dot green"></span>
              <span class="frame-url">/personal/play · gpt-image-2 终稿直出</span>
            </div>
            <img src="/screenshots/playground-xiaoqiao.png" alt="gpt-image-2 单次调用产出多张高清终稿" />
          </div>
        </div>
      </div>
    </section>

    <!-- ============= 三张卖点卡(全围绕"图") ============= -->
    <section class="section features">
      <div class="feature-grid">
        <div v-for="f in features" :key="f.title" class="feature-card">
          <div class="feature-icon" :style="{ background: f.color + '1A', color: f.color }">
            <el-icon :size="22"><component :is="f.icon" /></el-icon>
          </div>
          <div class="feature-title">{{ f.title }}</div>
          <div class="feature-desc" v-html="f.desc"></div>
        </div>
      </div>
      <div class="features-cta">
        <el-button size="large" type="primary" round @click="goPlay">
          立即开始出图 <el-icon><ArrowRight /></el-icon>
        </el-button>
      </div>
    </section>

    <!-- ============= Footer(极简) ============= -->
    <footer class="footer">
      <div class="footer-inner">
        <span>© {{ new Date().getFullYear() }} {{ siteName }} · gpt-image-2 终稿直出网关</span>
      </div>
    </footer>
  </div>
</template>

<style scoped lang="scss">
.landing {
  --lp-bg: #f6f3ee;
  --lp-bg-soft: rgba(255, 255, 255, 0.78);
  --lp-surface: rgba(255, 255, 255, 0.72);
  --lp-surface-strong: rgba(255, 255, 255, 0.9);
  --lp-text: #1f1d19;
  --lp-text-soft: #6f685e;
  --lp-text-mute: #9c9488;
  --lp-border: rgba(31, 29, 25, 0.08);
  --lp-shadow: 0 20px 48px rgba(26, 22, 16, 0.08);
  --lp-shadow-soft: 0 12px 28px rgba(26, 22, 16, 0.05);
  --lp-accent: #8a7a65;
  --lp-accent-soft: rgba(138, 122, 101, 0.12);

  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background:
    radial-gradient(circle at top left, rgba(138, 122, 101, 0.08), transparent 24%),
    radial-gradient(circle at bottom right, rgba(138, 122, 101, 0.05), transparent 28%),
    var(--lp-bg);
  color: var(--lp-text);
}

.landing.dark {
  --lp-bg: #141311;
  --lp-bg-soft: rgba(28, 25, 22, 0.72);
  --lp-surface: rgba(28, 25, 22, 0.74);
  --lp-surface-strong: rgba(34, 30, 27, 0.92);
  --lp-text: #f1ece4;
  --lp-text-soft: #b1aa9f;
  --lp-text-mute: #8a8378;
  --lp-border: rgba(255, 255, 255, 0.08);
  --lp-shadow: 0 24px 56px rgba(0, 0, 0, 0.28);
  --lp-shadow-soft: 0 14px 32px rgba(0, 0, 0, 0.18);
  --lp-accent: #b29c82;
  --lp-accent-soft: rgba(178, 156, 130, 0.14);
}

.gradient-text {
  color: var(--lp-text);
}

.nav {
  position: sticky;
  top: 0;
  z-index: 50;
  padding: 18px 0;
  background: transparent;
  border-bottom: 1px solid transparent;
  transition: background 0.25s ease, border-color 0.25s ease, box-shadow 0.25s ease, padding 0.25s ease;
}

.nav.scrolled {
  padding: 12px 0;
  background: var(--lp-bg-soft);
  border-bottom-color: var(--lp-border);
  box-shadow: var(--lp-shadow-soft);
  backdrop-filter: blur(18px);
}

.nav-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.logo {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  color: var(--lp-text);
  cursor: pointer;
  text-decoration: none;

  .logo-img {
    width: 36px;
    height: 36px;
    border-radius: 12px;
    object-fit: contain;
    background: rgba(255, 255, 255, 0.9);
  }

  .logo-mark {
    width: 36px;
    height: 36px;
    border-radius: 12px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: var(--lp-accent);
    color: #fff;
    font-size: 14px;
    font-weight: 700;
    box-shadow: 0 10px 22px rgba(138, 122, 101, 0.2);
  }

  .logo-name {
    font-size: 16px;
    font-weight: 600;
    letter-spacing: 0.01em;
  }
}

.nav-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;

  .theme-btn {
    padding: 4px 8px;
  }

  .btn-login {
    font-weight: 500;
  }
}

.hero {
  position: relative;
  padding: 48px 28px 56px;
}

.hero-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    radial-gradient(circle at 18% 12%, rgba(138, 122, 101, 0.12), transparent 22%),
    radial-gradient(circle at 88% 12%, rgba(138, 122, 101, 0.08), transparent 24%);
}

.hero-inner {
  position: relative;
  z-index: 1;
  max-width: 1200px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: minmax(0, 1.02fr) minmax(0, 0.98fr);
  gap: 40px;
  align-items: center;
}

.eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 7px 12px;
  border-radius: 999px;
  border: 1px solid var(--lp-border);
  background: var(--lp-surface);
  color: var(--lp-text-soft);
  font-size: 12px;
  backdrop-filter: blur(16px);

  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--lp-accent);
    box-shadow: 0 0 0 4px var(--lp-accent-soft);
  }
}

.hero-title {
  margin: 24px 0 16px;
  font-size: clamp(38px, 5vw, 62px);
  line-height: 1.06;
  font-weight: 700;
  letter-spacing: -0.03em;
}

.hero-sub {
  margin: 0 0 28px;
  font-size: 16px;
  line-height: 1.85;
  color: var(--lp-text-soft);

  b {
    color: var(--lp-text);
    font-weight: 600;
  }
}

.hero-cta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.hero-preview {
  position: relative;

  .preview-glow {
    position: absolute;
    inset: 24px;
    background: radial-gradient(circle, rgba(138, 122, 101, 0.16), transparent 68%);
    filter: blur(28px);
    pointer-events: none;
  }

  .preview-frame {
    position: relative;
    overflow: hidden;
    border: 1px solid var(--lp-border);
    border-radius: 24px;
    background: var(--lp-surface-strong);
    box-shadow: var(--lp-shadow);
  }

  .frame-bar {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 14px 18px;
    border-bottom: 1px solid var(--lp-border);
    background: rgba(255, 255, 255, 0.4);

    .dot {
      width: 9px;
      height: 9px;
      border-radius: 50%;
      background: rgba(31, 29, 25, 0.18);

      &.red,
      &.yellow,
      &.green {
        background: rgba(31, 29, 25, 0.16);
      }
    }

    .frame-url {
      margin-left: 10px;
      font-size: 12px;
      color: var(--lp-text-mute);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }

  img {
    display: block;
    width: 100%;
    height: auto;
  }
}

.section {
  position: relative;
  padding: 12px 28px 72px;
}

.feature-grid {
  max-width: 1200px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
}

.feature-card {
  padding: 28px;
  border-radius: 22px;
  border: 1px solid var(--lp-border);
  background: var(--lp-surface);
  box-shadow: var(--lp-shadow-soft);
  backdrop-filter: blur(16px);
  transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease;
}

.feature-card:hover {
  transform: translateY(-2px);
  border-color: rgba(138, 122, 101, 0.18);
  box-shadow: var(--lp-shadow);
}

.feature-icon {
  width: 44px;
  height: 44px;
  margin-bottom: 18px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.feature-title {
  margin-bottom: 10px;
  font-size: 17px;
  font-weight: 600;
}

.feature-desc {
  font-size: 14px;
  line-height: 1.8;
  color: var(--lp-text-soft);

  :deep(code) {
    padding: 2px 6px;
    border-radius: 8px;
    background: var(--lp-accent-soft);
    color: var(--lp-accent);
    font-family: 'JetBrains Mono', Menlo, Consolas, monospace;
    font-size: 12px;
  }

  :deep(b) {
    color: var(--lp-text);
    font-weight: 600;
  }
}

.features-cta {
  margin-top: 36px;
  text-align: center;
}

.footer {
  margin-top: auto;
  padding: 22px 28px 28px;
}

.footer-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding-top: 18px;
  border-top: 1px solid var(--lp-border);
  text-align: center;
  font-size: 12.5px;
  color: var(--lp-text-mute);
}

@media (max-width: 1100px) {
  .hero-inner {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .feature-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .nav-inner,
  .hero,
  .section,
  .footer {
    padding-left: 16px;
    padding-right: 16px;
  }

  .nav-inner {
    gap: 12px;
  }

  .nav-actions .btn-login {
    display: none;
  }

  .hero {
    padding-top: 28px;
    padding-bottom: 36px;
  }

  .hero-title {
    margin-top: 18px;
    font-size: 34px;
  }

  .hero-cta .el-button {
    width: 100%;
  }

  .feature-card {
    padding: 22px;
  }

  .section {
    padding-bottom: 48px;
  }
}
</style>

