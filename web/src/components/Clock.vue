<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { usePanelStore } from '@/store/panel'

const panel = usePanelStore()
const config = panel.config
const now = ref(new Date())
let timer = 0

onMounted(() => {
  timer = window.setInterval(() => (now.value = new Date()), 1000)
})
onUnmounted(() => window.clearInterval(timer))

const pad = (n: number) => String(n).padStart(2, '0')

const hours = computed(() => pad(now.value.getHours()))
const minutes = computed(() => pad(now.value.getMinutes()))
const seconds = computed(() => pad(now.value.getSeconds()))

const dateText = computed(() => {
  const d = now.value
  const weekday = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'][d.getDay()]
  return `${d.getMonth() + 1} 月 ${d.getDate()} 日 · ${weekday}`
})

const color = computed(() => config.logoColor || 'var(--text-strong)')
</script>

<template>
  <div class="clock" :style="{ color, opacity: Math.max(0, config.headerOpacity) / 100 }">
    <div class="time">
      <span>{{ hours }}</span>
      <span class="colon">:</span>
      <span>{{ minutes }}</span>
      <span v-if="config.clockShowSeconds" class="seconds">{{ seconds }}</span>
    </div>
    <div v-if="config.clockShowDate" class="date">{{ dateText }}</div>
  </div>
</template>

<style scoped>
.clock {
  text-align: center;
  user-select: none;
}

.time {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 1px;
  font-family: 'Inter', 'SF Pro Display', system-ui, sans-serif;
  font-size: clamp(42px, 6.4vw, 76px);
  font-weight: 200;
  line-height: 1;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
  text-shadow: 0 12px 40px rgba(23, 32, 51, 0.18);
}

.colon {
  transform: translateY(-0.06em);
  opacity: 0.55;
  margin: 0 0.02em;
}

.seconds {
  font-size: 0.42em;
  font-weight: 300;
  opacity: 0.6;
  margin-left: 0.18em;
}

.date {
  margin-top: 10px;
  font-size: 12px;
  letter-spacing: 0.22em;
  color: var(--text-soft);
  opacity: 0.9;
}
</style>
