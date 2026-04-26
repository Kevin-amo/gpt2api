<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import type { FormInstance } from 'element-plus'
import { ElMessage } from 'element-plus'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useSiteStore } from '@/stores/site'

const router = useRouter()
const route = useRoute()
const store = useUserStore()
const site = useSiteStore()

const siteName = computed(() => site.get('site.name', 'GPT2API'))
const siteDesc = computed(() =>
  site.get('site.description', '基于 chatgpt.com 的 OpenAI 兼容网关 · 多账号池 · IMG2 终稿直出 · 批量出图'),
)
const siteLogo = computed(() => site.get('site.logo_url', ''))
const siteFooter = computed(() => site.get('site.footer', ''))
const allowRegister = computed(() => site.allowRegister())

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  email: '',
  password: '',
})

const rules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '至少 6 位', trigger: 'blur' },
  ],
}

async function onSubmit() {
  if (!formRef.value) return
  const ok = await formRef.value.validate().catch(() => false)
  if (!ok) return
  loading.value = true
  try {
    await store.login(form.email, form.password)
    ElMessage.success('登录成功')
    const redirect = (route.query.redirect as string) || '/personal/dashboard'
    router.replace(redirect)
  } catch {
    // 错误已由 axios 拦截器 toast
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="auth-shell">
      <div class="hero">
        <div class="brand">
          <img v-if="siteLogo" :src="siteLogo" class="logo-img" alt="logo" />
          <div v-else class="mark">{{ (siteName[0] || 'G').toUpperCase() }}</div>
          <h1>{{ siteName }} 控制台</h1>
        </div>
        <p class="tagline">{{ siteDesc }}</p>
        <ul class="features">
          <li><el-icon><Lightning /></el-icon> 多账号池 / 多代理池 · IMG2 终稿直出 · 批量出图 · 高并发调度</li>
          <li><el-icon><Lock /></el-icon> RBAC 权限 · 全链路审计 · 数据库一键备份 / 恢复</li>
          <li><el-icon><Medal /></el-icon> 积分钱包 · 预扣结算 · 易支付接入 · 用量透明</li>
        </ul>
      </div>

      <el-card class="form-card" shadow="never">
        <div class="form-title">欢迎回来</div>
        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          size="large"
          label-position="top"
          @submit.prevent="onSubmit"
        >
          <el-form-item label="邮箱" prop="email">
            <el-input v-model="form.email" placeholder="you@example.com" autocomplete="username">
              <template #prefix><el-icon><Message /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="form.password" type="password" show-password placeholder="至少 6 位"
                      autocomplete="current-password" @keyup.enter="onSubmit">
              <template #prefix><el-icon><Lock /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-button type="primary" :loading="loading" class="submit" @click="onSubmit">登录</el-button>
          <div class="foot">
            <template v-if="allowRegister">
              还没有账号?<router-link to="/register">立即注册</router-link>
            </template>
            <template v-else>
              <span class="muted">管理员已关闭自助注册,请联系管理员创建账号</span>
            </template>
          </div>
        </el-form>
      </el-card>
    </div>
    <div v-if="siteFooter" class="site-footer">{{ siteFooter }}</div>
  </div>
</template>

<style scoped lang="scss">
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
  background:
    radial-gradient(circle at top left, rgba(138, 122, 101, 0.08), transparent 22%),
    radial-gradient(circle at bottom right, rgba(138, 122, 101, 0.05), transparent 28%),
    var(--gp-bg);
}

.auth-shell {
  width: min(1040px, 100%);
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(360px, 420px);
  gap: 32px;
  align-items: center;
}

.hero {
  padding: 24px 8px;

  .brand {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  .logo-img {
    width: 48px;
    height: 48px;
    border-radius: 14px;
    object-fit: contain;
    background: rgba(255, 255, 255, 0.92);
  }

  .mark {
    width: 48px;
    height: 48px;
    border-radius: 14px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: var(--gp-accent);
    color: #fff;
    font-size: 16px;
    font-weight: 700;
    box-shadow: 0 10px 24px rgba(138, 122, 101, 0.2);
  }

  h1 {
    margin: 0;
    font-size: 28px;
    font-weight: 600;
    letter-spacing: -0.02em;
    color: var(--gp-text);
  }

  .tagline {
    margin: 18px 0 0;
    max-width: 520px;
    color: var(--gp-text-soft);
    line-height: 1.85;
  }

  .features {
    list-style: none;
    margin: 32px 0 0;
    padding: 0;
    display: grid;
    gap: 14px;
    color: var(--gp-text);

    li {
      display: flex;
      align-items: flex-start;
      gap: 10px;
      padding: 14px 16px;
      border: 1px solid var(--gp-border);
      border-radius: 18px;
      background: rgba(255, 255, 255, 0.42);
      box-shadow: var(--gp-shadow-sm);
      line-height: 1.75;

      .el-icon {
        margin-top: 4px;
        color: var(--gp-accent);
      }
    }
  }
}

.form-card {
  width: 100%;

  .form-title {
    font-size: 24px;
    font-weight: 600;
    letter-spacing: -0.02em;
    margin-bottom: 6px;
  }

  .form-sub {
    margin-bottom: 22px;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .submit {
    width: 100%;
    margin-top: 4px;
  }

  .foot {
    margin-top: 18px;
    text-align: center;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
}

.site-footer {
  position: fixed;
  left: 16px;
  right: 16px;
  bottom: 12px;
  text-align: center;
  font-size: 12px;
  color: var(--gp-text-mute);
}

.foot .muted {
  color: var(--gp-text-mute);
}

@media (max-width: 960px) {
  .login-page {
    padding: 28px 20px;
  }

  .auth-shell {
    grid-template-columns: 1fr;
    gap: 20px;
  }

  .hero {
    padding: 0;
  }
}

@media (max-width: 640px) {
  .login-page {
    padding: 16px;
  }

  .auth-shell {
    gap: 16px;
  }

  .hero {
    display: none;
  }

  .site-footer {
    position: static;
    margin-top: 16px;
  }
}
</style>