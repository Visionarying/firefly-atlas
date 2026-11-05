<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { ArrowRight, Eye, EyeOff, Lock, ShieldCheck, UserRound } from 'lucide-vue-next'
import Background from '@/components/Background.vue'
import NoticeBanner from '@/components/NoticeBanner.vue'
import { useAuthStore } from '@/store/auth'
import { systemApi } from '@/api'

const auth = useAuthStore()
const router = useRouter()
const message = useMessage()

const username = ref('')
const password = ref('')
const showPassword = ref(false)
const submitting = ref(false)
const publicMode = ref(false)
const siteTitle = ref('')
const notice = ref<{ title: string; content: string } | null>(null)

onMounted(async () => {
  try {
    const data = await systemApi.site()
    publicMode.value = !!data.publicVisitMode
    siteTitle.value = data.settings.site_title || ''
    if (data.settings.notice_enabled === '1' && data.settings.notice_content) {
      const scope = data.settings.notice_scope || 'home'
      if (scope === 'login' || scope === 'both') {
        notice.value = {
          title: data.settings.notice_title || '公告',
          content: data.settings.notice_content,
        }
      }
    }
  } catch {
    publicMode.value = false
  }
})

async function submit() {
  if (!username.value.trim() || !password.value) {
    message.warning('请输入用户名和密码')
    return
  }
  submitting.value = true
  try {
    await auth.login(username.value.trim(), password.value)
    message.success('欢迎回来')
    router.push('/')
  } catch (e: any) {
    message.error(e.message || '登录失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <Background />

    <div class="brand">
      <span class="brand-mark"><span /></span>
      <span class="brand-text">{{ siteTitle || 'Firefly Panel' }}</span>
    </div>

    <div class="login-wrap">
      <NoticeBanner
        v-if="notice"
        class="login-notice"
        :title="notice.title"
        :content="notice.content"
      />
      <div class="login-card">
        <div class="logo-badge">F</div>
        <h1>欢迎回来</h1>
        <p class="subtitle">登录你的私有云导航面板</p>

        <form class="form" @submit.prevent="submit">
          <div class="input-wrap">
            <UserRound :size="16" class="input-icon" />
            <input
              v-model="username"
              class="login-input"
              type="text"
              placeholder="用户名"
              autocomplete="username"
              autofocus
            />
          </div>

          <div class="input-wrap">
            <Lock :size="16" class="input-icon" />
            <input
              v-model="password"
              class="login-input"
              :type="showPassword ? 'text' : 'password'"
              placeholder="密码"
              autocomplete="current-password"
            />
            <button class="eye" type="button" :title="showPassword ? '隐藏密码' : '显示密码'" @click="showPassword = !showPassword">
              <EyeOff v-if="showPassword" :size="15" />
              <Eye v-else :size="15" />
            </button>
          </div>

          <button class="submit" type="submit" :disabled="submitting">
            <span>{{ submitting ? '登录中...' : '登录' }}</span>
            <ArrowRight v-if="!submitting" :size="16" />
          </button>
        </form>

        <div class="divider"><span>或</span></div>

        <router-link v-if="publicMode" to="/" class="guest-link">
          以访客身份浏览
        </router-link>
        <button v-else class="guest-link disabled" type="button" disabled title="管理员尚未开启公开访客模式">
          访客模式未开启
        </button>

        <div class="hint">
          <ShieldCheck :size="13" />
          <span>首次部署请查看控制台日志 initial password 获取初始密码，登录后请立即修改</span>
        </div>
      </div>

      <p class="foot">Private cloud · NAS · Home lab</p>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  position: relative;
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.brand {
  position: fixed;
  top: 22px;
  left: 26px;
  display: flex;
  align-items: center;
  gap: 9px;
  z-index: 2;
}
.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--accent) 14%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent) 22%, transparent);
}
.brand-mark span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 12px color-mix(in srgb, var(--accent) 80%, transparent);
  animation: glow 2.4s ease-in-out infinite;
}
@keyframes glow {
  0%,
  100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.18);
    opacity: 0.75;
  }
}
.brand-text {
  font-size: 12.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  color: var(--text-soft);
}

.login-wrap {
  width: 100%;
  max-width: 400px;
  animation: fadeUp 0.65s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.login-notice {
  margin-bottom: 12px;
}

.login-card {
  padding: 34px 30px 26px;
  border-radius: 22px;
  background: rgba(var(--glass-rgb), 0.62);
  border: 1px solid var(--glass-border);
  backdrop-filter: blur(30px) saturate(180%);
  -webkit-backdrop-filter: blur(30px) saturate(180%);
  box-shadow: 0 40px 90px -40px rgba(var(--shadow-color), 0.55);
  text-align: center;
}

.logo-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 54px;
  height: 54px;
  border-radius: 16px;
  background: linear-gradient(135deg, var(--accent), color-mix(in srgb, var(--accent) 50%, #3b82f6));
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  box-shadow: 0 18px 36px -16px color-mix(in srgb, var(--accent) 70%, transparent);
  margin-bottom: 18px;
}

h1 {
  margin: 0;
  font-size: 22px;
  font-weight: 650;
  color: var(--text-strong);
  letter-spacing: 0.01em;
}

.subtitle {
  margin: 7px 0 24px;
  font-size: 12.5px;
  color: var(--text-soft);
}

.form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.input-wrap {
  position: relative;
  display: flex;
  align-items: center;
}
.input-icon {
  position: absolute;
  left: 13px;
  color: var(--text-faint);
  pointer-events: none;
}
.login-input {
  width: 100%;
  height: 44px;
  padding: 0 40px 0 38px;
  border-radius: 11px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.5);
  color: var(--text-strong);
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s, background 0.2s;
}
.login-input::placeholder {
  color: var(--text-faint);
}
.login-input:focus {
  border-color: color-mix(in srgb, var(--accent) 55%, transparent);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--accent) 12%, transparent);
  background: rgba(var(--glass-rgb), 0.72);
}

.eye {
  position: absolute;
  right: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-faint);
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}
.eye:hover {
  color: var(--text-soft);
  background: rgba(var(--glass-rgb), 0.5);
}

.submit {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 44px;
  margin-top: 6px;
  border: none;
  border-radius: 11px;
  background: var(--accent);
  color: #fff;
  font-size: 13.5px;
  font-weight: 600;
  letter-spacing: 0.04em;
  cursor: pointer;
  box-shadow: 0 16px 32px -16px color-mix(in srgb, var(--accent) 75%, transparent);
  transition: transform 0.18s, filter 0.18s, box-shadow 0.18s;
}
.submit:hover:not(:disabled) {
  transform: translateY(-1px);
  filter: brightness(1.04);
  box-shadow: 0 20px 38px -16px color-mix(in srgb, var(--accent) 75%, transparent);
}
.submit:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.divider {
  position: relative;
  margin: 18px 0 12px;
  height: 1px;
  background: var(--line);
}
.divider span {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  padding: 0 10px;
  font-size: 10px;
  letter-spacing: 0.16em;
  color: var(--text-faint);
  background: color-mix(in srgb, var(--panel-bg) 40%, transparent);
}

.guest-link {
  display: inline-block;
  font-size: 12px;
  color: var(--text-soft);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  padding-bottom: 1px;
  transition: color 0.16s, border-color 0.16s;
  background: transparent;
  border-top: none;
  border-left: none;
  border-right: none;
  cursor: pointer;
}
.guest-link:hover:not(.disabled) {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 45%, transparent);
}
.guest-link.disabled {
  color: var(--text-faint);
  cursor: not-allowed;
}

.hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px dashed var(--line);
  font-size: 10.5px;
  line-height: 1.6;
  color: var(--text-faint);
}
.hint svg {
  flex-shrink: 0;
  color: color-mix(in srgb, var(--accent) 70%, var(--text-faint));
}

.foot {
  margin: 18px 0 0;
  text-align: center;
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-faint);
  opacity: 0.7;
}

@media (max-width: 480px) {
  .login-card {
    padding: 28px 20px 22px;
  }
  .brand {
    top: 14px;
    left: 16px;
  }
}
</style>
