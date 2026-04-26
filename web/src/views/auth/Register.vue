<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import type { FormInstance } from 'element-plus'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useSiteStore } from '@/stores/site'

const router = useRouter()
const store = useUserStore()
const site = useSiteStore()

const siteName = computed(() => site.get('site.name', 'GPT2API'))
const allowRegister = computed(() => site.allowRegister())

const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive({ email: '', password: '', confirm: '', nickname: '' })

const rules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 64, message: '6~64 位', trigger: 'blur' },
  ],
  confirm: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (_r: unknown, v: string, cb: (e?: Error) => void) => {
        if (v !== form.password) cb(new Error('两次密码不一致'))
        else cb()
      },
      trigger: 'blur',
    },
  ],
}

async function onSubmit() {
  if (!formRef.value) return
  const ok = await formRef.value.validate().catch(() => false)
  if (!ok) return
  loading.value = true
  try {
    await store.register(form.email, form.password, form.nickname)
    ElMessage.success('注册成功,正在登录…')
    await store.login(form.email, form.password)
    router.replace('/personal/dashboard')
  } catch {
    // toast 由拦截器处理
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="register-page">
    <div class="auth-shell">
      <div class="intro-card">
        <div class="intro-kicker">创建账号</div>
        <h1>{{ siteName }}</h1>
        <p>使用一套干净一致的控制台体验，完成注册后即可进入个人中心开始使用。</p>
      </div>

      <el-card class="form-card" shadow="never">
        <div class="form-title">创建 {{ siteName }} 账号</div>
        <div class="form-sub">免费注册，立即体验</div>
        <el-alert
          v-if="!allowRegister"
          type="warning"
          :closable="false"
          title="当前站点已关闭自助注册"
          description="请联系管理员开通账号，或改用已有账号登录。"
          style="margin-bottom:16px"
        />
        <el-form ref="formRef" :model="form" :rules="rules" label-position="top" size="large"
                 :disabled="!allowRegister" @submit.prevent="onSubmit">
          <el-form-item label="邮箱" prop="email">
            <el-input v-model="form.email" placeholder="you@example.com" autocomplete="username" />
          </el-form-item>
          <el-form-item label="昵称" prop="nickname">
            <el-input v-model="form.nickname" placeholder="选填" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="form.password" type="password" show-password autocomplete="new-password" />
          </el-form-item>
          <el-form-item label="确认密码" prop="confirm">
            <el-input v-model="form.confirm" type="password" show-password autocomplete="new-password"
                      @keyup.enter="onSubmit" />
          </el-form-item>
          <el-button type="primary" class="submit" :loading="loading" :disabled="!allowRegister"
                     @click="onSubmit">
            注册
          </el-button>
          <div class="foot">
            已有账号?<router-link to="/login">直接登录</router-link>
          </div>
        </el-form>
      </el-card>
    </div>
  </div>
</template>

<style scoped lang="scss">
.register-page {
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
  width: min(960px, 100%);
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(360px, 420px);
  gap: 32px;
  align-items: center;
}

.intro-card {
  padding: 28px;
  border: 1px solid var(--gp-border);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.42);
  box-shadow: var(--gp-shadow-sm);

  .intro-kicker {
    margin-bottom: 12px;
    color: var(--gp-text-mute);
    font-size: 12px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  h1 {
    margin: 0 0 12px;
    font-size: 30px;
    font-weight: 600;
    letter-spacing: -0.02em;
    color: var(--gp-text);
  }

  p {
    margin: 0;
    color: var(--gp-text-soft);
    line-height: 1.85;
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
    color: var(--el-text-color-secondary);
    margin-bottom: 18px;
    font-size: 13px;
  }
}

.submit {
  width: 100%;
}

.foot {
  margin-top: 16px;
  text-align: center;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

@media (max-width: 900px) {
  .register-page {
    padding: 24px 16px;
  }

  .auth-shell {
    grid-template-columns: 1fr;
    gap: 16px;
  }
}

@media (max-width: 640px) {
  .intro-card {
    display: none;
  }
}
</style>