<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import {
  NConfigProvider,
  NDialogProvider,
  NMessageProvider,
  darkTheme,
  dateZhCN,
  zhCN,
  type GlobalThemeOverrides,
} from 'naive-ui'
import { usePanelStore } from '@/store/panel'
import { useAuthStore } from '@/store/auth'

const panel = usePanelStore()
const auth = useAuthStore()
const router = useRouter()
const systemDark = ref(window.matchMedia('(prefers-color-scheme: dark)').matches)
const media = window.matchMedia('(prefers-color-scheme: dark)')
const onMedia = (e: MediaQueryListEvent) => (systemDark.value = e.matches)

const isDark = computed(
  () =>
    panel.config.themeMode === 'dark' ||
    (panel.config.themeMode === 'auto' && systemDark.value),
)

const theme = computed(() => (isDark.value ? darkTheme : null))

const overrides = computed<GlobalThemeOverrides>(() => ({
  common: {
    primaryColor: panel.config.accentColor,
    primaryColorHover: panel.config.accentColor,
    primaryColorPressed: panel.config.accentColor,
    primaryColorSuppl: panel.config.accentColor,
    borderRadius: '10px',
    borderRadiusSmall: '8px',
    fontFamily: "'Inter', 'Noto Sans SC', 'Microsoft YaHei', system-ui, sans-serif",
  },
  Button: {
    textColorPrimary: '#ffffff',
    textColorHoverPrimary: '#ffffff',
    textColorPressedPrimary: '#ffffff',
    textColorFocusPrimary: '#ffffff',
  },
  Slider: {
    fillColor: panel.config.accentColor,
    fillColorHover: panel.config.accentColor,
    handleColor: panel.config.accentColor,
  },
  Switch: {
    railColorActive: panel.config.accentColor,
  },
  Tabs: {
    tabTextColorActiveLine: panel.config.accentColor,
    barColor: panel.config.accentColor,
  },
}))

watchEffect(() => {
  document.documentElement.classList.toggle('dark', isDark.value)
  document.documentElement.style.setProperty('--accent', panel.config.accentColor)
})

function onUnauthorized() {
  auth.reset()
  if (router.currentRoute.value.name !== 'login') router.push('/login')
}

onMounted(async () => {
  media.addEventListener('change', onMedia)
  window.addEventListener('firefly:unauthorized', onUnauthorized)
  if (auth.isLoggedIn) await auth.fetchInfo()
})

onUnmounted(() => {
  media.removeEventListener('change', onMedia)
  window.removeEventListener('firefly:unauthorized', onUnauthorized)
})
</script>

<template>
  <n-config-provider :theme="theme" :theme-overrides="overrides" :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
