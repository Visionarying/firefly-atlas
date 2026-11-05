<script setup lang="ts">
import { computed } from 'vue'
import { Bot, House, Globe2 } from 'lucide-vue-next'
import { usePanelStore } from '@/store/panel'

const panel = usePanelStore()

const label = computed(() => {
  if (panel.lanMode === 'lan') return '内网'
  if (panel.lanMode === 'wan') return '公网'
  return panel.lanReachable ? '自动 · 内网' : '自动 · 公网'
})

const icon = computed(() => {
  if (panel.lanMode === 'lan') return House
  if (panel.lanMode === 'wan') return Globe2
  return Bot
})

const tip = computed(() =>
  panel.lanMode === 'auto'
    ? '自动感知网络环境，点击切换为强制内网'
    : panel.lanMode === 'lan'
      ? '当前强制使用内网地址，点击切换为强制公网'
      : '当前强制使用公网地址，点击切换为自动感知',
)
</script>

<template>
  <button class="lan-badge" :title="tip" type="button" @click="panel.cycleLanMode()">
    <component :is="icon" :size="14" :class="{ pulse: panel.lanProbing }" />
    <span>{{ label }}</span>
  </button>
</template>

<style scoped>
.lan-badge {
  position: fixed;
  right: 18px;
  bottom: 18px;
  z-index: 30;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 34px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1px solid var(--glass-border);
  background: rgba(var(--glass-rgb), 0.62);
  backdrop-filter: blur(18px) saturate(160%);
  color: var(--text-soft);
  font-size: 11.5px;
  letter-spacing: 0.04em;
  cursor: pointer;
  box-shadow: 0 10px 26px -18px rgba(var(--shadow-color), 0.55);
  transition: all 0.2s ease;
}

.lan-badge:hover {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 40%, transparent);
  transform: translateY(-1px);
}

.pulse {
  animation: pulse 1s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.35;
  }
}

@media (max-width: 560px) {
  .lan-badge {
    right: 12px;
    bottom: 12px;
  }
}
</style>
