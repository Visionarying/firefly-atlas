<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Activity, Clock3, Cpu, HardDrive, MemoryStick, Network, Server } from 'lucide-vue-next'
import { systemApi } from '@/api'
import { useAuthStore } from '@/store/auth'
import { usePanelStore } from '@/store/panel'
import type { MonitorData } from '@/typings'

const panel = usePanelStore()
const auth = useAuthStore()

const data = ref<MonitorData | null>(null)
const loading = ref(false)
const failed = ref(false)
let timer: number | undefined

async function refresh() {
  loading.value = true
  try {
    data.value = auth.isLoggedIn ? await systemApi.monitor() : await systemApi.publicMonitor()
    failed.value = !data.value
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void refresh()
  timer = window.setInterval(() => void refresh(), 5000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})

const showTitle = computed(() => panel.config.systemMonitorShowTitle)

function formatBytes(value: number): string {
  if (!value || value < 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let size = value
  let index = 0
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024
    index += 1
  }
  return `${size.toFixed(size >= 100 || index === 0 ? 0 : 1)} ${units[index]}`
}

function formatUptime(seconds: number): string {
  if (!seconds) return '—'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days} 天 ${hours} 小时`
  if (hours > 0) return `${hours} 小时 ${minutes} 分`
  return `${minutes} 分钟`
}

function percent(value?: number): number {
  return Math.min(100, Math.max(0, Number(value) || 0))
}

function tone(value?: number): string {
  const p = percent(value)
  if (p >= 88) return 'danger'
  if (p >= 68) return 'warn'
  return 'ok'
}

const cpu = computed(() => percent(data.value?.cpuPercent))
const memory = computed(() => percent(data.value?.memPercent))
const disk = computed(() => percent(data.value?.diskPercent))
</script>

<template>
  <section v-if="data" class="monitor">
    <header v-if="showTitle" class="monitor-head">
      <div class="monitor-title">
        <Server :size="15" />
        <span>系统状态</span>
        <span v-if="data.hostname" class="host">{{ data.hostname }}</span>
      </div>
      <div class="monitor-meta">
        <span v-if="data.platform" class="platform">{{ data.platform }}</span>
        <span class="uptime"><Clock3 :size="12" />已运行 {{ formatUptime(data.uptime) }}</span>
        <button class="refresh" type="button" title="刷新" :class="{ spin: loading }" @click="refresh">
          <Activity :size="13" />
        </button>
      </div>
    </header>

    <div class="metrics">
      <div class="metric">
        <div class="metric-top">
          <span class="metric-name"><Cpu :size="12" />CPU</span>
          <strong :class="tone(cpu)">{{ cpu.toFixed(0) }}%</strong>
        </div>
        <span class="metric-sub">{{ data.cpuCores }} 核 · {{ data.cpuModel || '未知型号' }}</span>
      </div>

      <div class="metric">
        <div class="metric-top">
          <span class="metric-name"><MemoryStick :size="12" />内存</span>
          <strong :class="tone(memory)">{{ memory.toFixed(0) }}%</strong>
        </div>
        <span class="metric-sub">{{ formatBytes(data.memUsed) }} / {{ formatBytes(data.memTotal) }}</span>
      </div>

      <div class="metric">
        <div class="metric-top">
          <span class="metric-name"><HardDrive :size="12" />磁盘</span>
          <strong :class="tone(disk)">{{ disk.toFixed(0) }}%</strong>
        </div>
        <span class="metric-sub">{{ formatBytes(data.diskUsed) }} / {{ formatBytes(data.diskTotal) }}</span>
      </div>

      <div class="metric">
        <div class="metric-top">
          <span class="metric-name"><Network :size="12" />网络</span>
          <strong class="net">↑{{ formatBytes(data.netSent) }} ↓{{ formatBytes(data.netRecv) }}</strong>
        </div>
        <span class="metric-sub">累计流量 · 每 5 秒刷新</span>
      </div>
    </div>

    <p v-if="failed" class="monitor-err">暂时无法读取系统状态</p>
  </section>
</template>

<style scoped>
.monitor {
  width: 100%;
  margin: 14px auto 0;
  padding: 8px 14px 9px;
  border-radius: 13px;
  border: 1px solid var(--glass-border);
  background: rgba(var(--glass-rgb), 0.4);
  backdrop-filter: blur(22px) saturate(160%);
  -webkit-backdrop-filter: blur(22px) saturate(160%);
  box-shadow: 0 12px 30px -28px rgba(var(--shadow-color), 0.5);
  animation: fadeUp 0.55s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.monitor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 5px;
  padding-bottom: 5px;
  border-bottom: 1px dashed var(--line);
}

.monitor-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11.5px;
  font-weight: 600;
  color: var(--text-strong);
}
.monitor-title svg {
  color: var(--accent);
}
.host {
  font-size: 10px;
  font-weight: 400;
  color: var(--text-faint);
  padding: 0 6px;
  border-radius: 999px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.4);
}

.monitor-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: 10px;
  color: var(--text-faint);
}
.platform {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 180px;
}
.uptime {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

.refresh {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: 1px solid var(--line);
  border-radius: 7px;
  background: transparent;
  color: var(--text-faint);
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}
.refresh:hover {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 35%, transparent);
}
.refresh.spin svg {
  animation: mon-spin 0.9s linear infinite;
}
@keyframes mon-spin {
  to {
    transform: rotate(360deg);
  }
}

.metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0;
}

.metric {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 0 12px;
  border-left: 1px solid var(--line);
}
.metric:first-child {
  padding-left: 0;
  border-left: 0;
}

.metric-top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}
.metric-name {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10.5px;
  color: var(--text-soft);
}
.metric-top strong {
  font-size: 13px;
  color: var(--text-strong);
  font-variant-numeric: tabular-nums;
}
.metric-top strong.ok {
  color: var(--accent-strong, #059669);
}
.metric-top strong.warn {
  color: #d97706;
}
.metric-top strong.danger {
  color: #e11d48;
}
.metric-top strong.net {
  font-size: 10.5px;
  color: var(--text-soft);
  font-weight: 500;
}

.metric-sub {
  font-size: 9.5px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.monitor-err {
  margin: 8px 0 0;
  font-size: 10px;
  color: var(--text-faint);
}

@media (max-width: 860px) {
  .metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    row-gap: 7px;
  }
  .metric {
    padding: 0 10px;
  }
  .metric:nth-child(odd) {
    padding-left: 0;
    border-left: 0;
  }
  .metric:nth-child(n + 3) {
    padding-top: 7px;
    border-top: 1px solid var(--line);
  }
  .platform {
    display: none;
  }
}

@media (max-width: 560px) {
  .monitor {
    padding: 9px 12px 10px;
    margin-top: 12px;
  }
  .uptime {
    display: none;
  }
  .metric {
    padding: 0 9px;
  }
  .metric:nth-child(odd) {
    padding-left: 0;
  }
}
</style>
