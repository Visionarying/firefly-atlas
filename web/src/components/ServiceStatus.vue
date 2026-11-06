<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { Activity, Bell, BellOff, RefreshCw, Wrench } from 'lucide-vue-next'
import { healthApi, panelApi, settingApi } from '@/api'
import { useAuthStore } from '@/store/auth'
import { usePanelStore } from '@/store/panel'
import type { HealthStatusItem, HealthStatusOverview } from '@/typings'
import {
  baselineHealthAlerts,
  notificationPermission,
  notificationsEnabled,
  notificationsSupported,
  notifyHealthAlerts,
  requestNotificationPermission,
  setNotificationsEnabled,
} from '@/utils/healthNotify'

const message = useMessage()
const panel = usePanelStore()
const auth = useAuthStore()

const overview = ref<HealthStatusOverview | null>(null)
const settings = ref<Record<string, string>>({})
const loading = ref(false)
const runPending = ref(false)
const togglingItem = ref(0)
const filter = ref<'all' | 'monitored'>('all')
const permission = ref(notificationPermission())
const notifyEnabled = ref(notificationsEnabled())
const timers: number[] = []

const isAdmin = computed(() => auth.user?.role === 1)
const items = computed(() => overview.value?.items || [])
const monitored = computed(() => items.value.filter((item) => item.monitored))
const visibleItems = computed(() =>
  filter.value === 'monitored' ? items.value.filter((item) => item.monitored) : items.value,
)
const onlineCount = computed(() => monitored.value.filter((item) => item.checkedAt && item.online).length)
const avgUptime = computed(() => {
  const values = monitored.value
    .map((item) => item.uptime24h)
    .filter((value): value is number => value !== null && value !== undefined)
  if (!values.length) return '—'
  const average = values.reduce((sum, value) => sum + value, 0) / values.length
  return `${average.toFixed(1)}%`
})
const lastSampleAt = computed(() => {
  const stamps = monitored.value.map((item) => item.checkedAt).filter((value) => value > 0)
  return stamps.length ? Math.max(...stamps) : 0
})
const lastSampleText = computed(() => (lastSampleAt.value ? timeAgo(lastSampleAt.value) : '等待首次采样'))
const sampling = computed(() => overview.value?.sampling || runPending.value)
const monitorEnabled = computed(() => overview.value?.enabled ?? true)
const maintenanceEnabled = computed(() => settings.value.maintenance_enabled === '1')
const notifyLabel = computed(() => {
  if (permission.value === 'unsupported') return '浏览器不支持通知'
  if (permission.value === 'denied') return '通知被浏览器禁止'
  if (!notifyEnabled.value) return '开启浏览器通知'
  if (permission.value !== 'granted') return '授权浏览器通知'
  return '浏览器通知已开启'
})

function timeAgo(timestamp: number): string {
  const diff = Date.now() - timestamp
  if (diff < 60_000) return '刚刚'
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`
  return `${Math.floor(diff / 86_400_000)} 天前`
}

function uptimeText(item: HealthStatusItem): string {
  if (item.uptime24h === null || item.uptime24h === undefined) return '—'
  return `${item.uptime24h.toFixed(1)}%`
}

function uptimeWidth(item: HealthStatusItem): string {
  if (item.uptime24h === null || item.uptime24h === undefined) return '0%'
  return `${Math.min(100, Math.max(0, item.uptime24h))}%`
}

function dotClass(item: HealthStatusItem): string {
  if (!item.monitored || !item.checkedAt) return 'idle'
  return item.online ? 'online' : 'offline'
}

function statusText(item: HealthStatusItem): string {
  if (!item.monitored) return '未监测'
  if (!item.checkedAt) return '等待采样'
  return item.online ? '在线' : '离线'
}

async function refresh(quiet = false) {
  if (!quiet) loading.value = true
  try {
    const data = await healthApi.overview()
    overview.value = data
    notifyHealthAlerts(data.items)
  } catch (e: any) {
    if (!quiet) message.error(e.message || '获取服务状态失败')
  } finally {
    if (!quiet) loading.value = false
  }
}

async function loadSettings() {
  try {
    const list = await settingApi.get()
    const map: Record<string, string> = {}
    for (const setting of list) map[setting.configName] = setting.content
    settings.value = map
  } catch {
    /* settings are optional for the status view */
  }
}

function scheduleRefresh(delay: number) {
  const timer = window.setTimeout(() => void refresh(true), delay)
  timers.push(timer)
}

async function runNow() {
  if (!isAdmin.value) return
  runPending.value = true
  try {
    const result = await healthApi.run()
    message.success(result.started ? '已开始采样，稍后刷新结果' : '采样正在进行中')
    scheduleRefresh(4000)
    timers.push(
      window.setTimeout(() => {
        runPending.value = false
        void refresh(true)
      }, 12000),
    )
  } catch (e: any) {
    runPending.value = false
    message.error(e.message || '触发采样失败')
  }
}

async function saveSettings(next: { configName: string; content: string }[]) {
  if (!isAdmin.value) return
  try {
    await settingApi.save(next)
    for (const item of next) settings.value[item.configName] = item.content
  } catch (e: any) {
    message.error(e.message || '保存失败')
  }
}

async function toggleMonitor() {
  if (!isAdmin.value || !overview.value) return
  const next = !monitorEnabled.value
  overview.value.enabled = next
  settings.value.health_monitor_enabled = next ? '1' : '0'
  await saveSettings([{ configName: 'health_monitor_enabled', content: next ? '1' : '0' }])
  message.success(next ? '后台采样已开启' : '后台采样已关闭')
}

async function toggleMaintenance() {
  if (!isAdmin.value) return
  const next = !maintenanceEnabled.value
  settings.value.maintenance_enabled = next ? '1' : '0'
  if (overview.value) overview.value.maintenance.enabled = next
  await saveSettings([{ configName: 'maintenance_enabled', content: next ? '1' : '0' }])
  message.success(next ? '维护窗口已开启' : '维护窗口已关闭')
}

async function saveMaintenanceTime(key: 'maintenance_start' | 'maintenance_end') {
  if (!isAdmin.value) return
  const value = settings.value[key] || ''
  if (!/^\d{2}:\d{2}$/.test(value)) {
    message.warning('时间格式应为 HH:MM')
    return
  }
  if (overview.value) {
    if (key === 'maintenance_start') overview.value.maintenance.start = value
    else overview.value.maintenance.end = value
  }
  await saveSettings([{ configName: key, content: value }])
}

async function toggleItem(item: HealthStatusItem) {
  if (!isAdmin.value || togglingItem.value) return
  const full = panel.items.find((entry) => entry.id === item.itemId)
  if (!full) {
    message.warning('找不到这张卡片，请刷新页面后重试')
    return
  }
  const next = !item.monitored
  togglingItem.value = item.itemId
  try {
    await panelApi.saveItem({
      id: full.id,
      itemIconGroupId: full.itemIconGroupId,
      title: full.title,
      url: full.url,
      lanUrl: full.lanUrl,
      description: full.description,
      openMethod: full.openMethod,
      healthCheck: next,
      icon: full.icon,
      sort: full.sort,
    })
    full.healthCheck = next
    item.monitored = next
    message.success(next ? `已开启「${item.title}」监测` : `已停止「${item.title}」监测`)
    if (next) scheduleRefresh(3000)
  } catch (e: any) {
    message.error(e.message || '保存失败')
  } finally {
    togglingItem.value = 0
  }
}

async function toggleNotifications() {
  if (!notificationsSupported()) {
    message.warning('当前浏览器不支持桌面通知')
    return
  }
  if (permission.value === 'denied') {
    message.warning('通知已被浏览器禁止，请在站点设置中允许后重试')
    return
  }
  if (notifyEnabled.value && permission.value === 'granted') {
    setNotificationsEnabled(false)
    notifyEnabled.value = false
    message.success('浏览器通知已关闭')
    return
  }
  const result = await requestNotificationPermission()
  permission.value = result
  if (result !== 'granted') {
    message.warning('未获得通知权限')
    return
  }
  setNotificationsEnabled(true)
  notifyEnabled.value = true
  baselineHealthAlerts(items.value)
  message.success('浏览器通知已开启，离线时会提醒一次')
}

let pollTimer: number | undefined

onMounted(async () => {
  await Promise.all([refresh(), loadSettings()])
  pollTimer = window.setInterval(() => void refresh(true), 60_000)
})

onUnmounted(() => {
  if (pollTimer) window.clearInterval(pollTimer)
  for (const timer of timers) window.clearTimeout(timer)
})
</script>

<template>
  <section class="svc-pane">
    <div class="svc-head">
      <h2><Activity :size="17" class="svc-ico" /> 服务状态</h2>
      <p>
        后台每 {{ overview?.intervalMinutes || 5 }} 分钟采样一次，历史保留
        {{ overview?.retentionDays || 7 }} 天；在线率由数据库聚合，不会把全量历史读进内存。
      </p>
    </div>

    <div class="svc-block">
      <div class="stat-grid">
        <div>
          <span>监测卡片</span>
          <strong>{{ monitored.length }}<small> / {{ items.length }}</small></strong>
        </div>
        <div>
          <span>当前在线</span>
          <strong class="ok">{{ onlineCount }}</strong>
        </div>
        <div>
          <span>24h 平均在线率</span>
          <strong>{{ avgUptime }}</strong>
        </div>
        <div>
          <span>最近采样</span>
          <strong class="small-value">{{ lastSampleText }}</strong>
        </div>
      </div>
      <div class="svc-actions">
        <button class="svc-btn primary" type="button" :disabled="!isAdmin || sampling" @click="runNow">
          <RefreshCw :size="13" :class="{ spin: sampling }" />
          {{ sampling ? '采样中…' : '立即检查' }}
        </button>
        <button class="svc-btn" type="button" @click="toggleNotifications">
          <Bell v-if="notifyEnabled && permission === 'granted'" :size="13" />
          <BellOff v-else :size="13" />
          {{ notifyLabel }}
        </button>
      </div>
      <p class="svc-hint">
        浏览器通知只在面板页面打开时可用；在线 → 离线只提醒一次，维护窗口内不会提醒。
      </p>
    </div>

    <div class="svc-block">
      <h3><Wrench :size="13" class="svc-ico" /> 采样与维护窗口</h3>
      <div class="svc-row">
        <div>
          <div class="row-label">启用后台采样</div>
          <div class="row-hint">关闭后不再写入新记录，已有历史仍会保留到清理日期。</div>
        </div>
        <button
          class="toggle"
          type="button"
          :class="{ active: monitorEnabled }"
          :disabled="!isAdmin"
          @click="toggleMonitor"
        >
          <span />
        </button>
      </div>
      <div class="svc-row">
        <div>
          <div class="row-label">维护窗口不告警</div>
          <div class="row-hint">支持跨午夜，例如 23:30 - 01:30；开始与结束相同表示关闭。</div>
        </div>
        <button
          class="toggle"
          type="button"
          :class="{ active: maintenanceEnabled }"
          :disabled="!isAdmin"
          @click="toggleMaintenance"
        >
          <span />
        </button>
      </div>
      <div v-if="maintenanceEnabled" class="svc-row">
        <div>
          <div class="row-label">每日维护时段</div>
          <div class="row-hint">按面板服务器本地时间计算，维护期间不产生离线提醒。</div>
        </div>
        <div class="time-range">
          <input
            v-model="settings.maintenance_start"
            class="svc-field"
            type="time"
            :disabled="!isAdmin"
            @change="saveMaintenanceTime('maintenance_start')"
          />
          <span>至</span>
          <input
            v-model="settings.maintenance_end"
            class="svc-field"
            type="time"
            :disabled="!isAdmin"
            @change="saveMaintenanceTime('maintenance_end')"
          />
        </div>
      </div>
      <p v-if="!isAdmin" class="svc-hint">普通用户可以查看状态，但只有管理员能修改采样配置。</p>
    </div>

    <div class="svc-block">
      <div class="svc-block-head">
        <h3>卡片监测</h3>
        <div class="svc-filter">
          <button type="button" :class="{ active: filter === 'all' }" @click="filter = 'all'">
            全部 {{ items.length }}
          </button>
          <button type="button" :class="{ active: filter === 'monitored' }" @click="filter = 'monitored'">
            仅监测 {{ monitored.length }}
          </button>
        </div>
      </div>
      <div v-if="loading && !items.length" class="svc-empty">正在读取状态…</div>
      <div v-else-if="!items.length" class="svc-empty">还没有卡片，先到「分组管理」添加一个吧。</div>
      <div v-else-if="!visibleItems.length" class="svc-empty">还没有卡片开启监测，点下面的「开启监测」即可。</div>
      <div
        v-for="item in visibleItems"
        :key="item.itemId"
        class="status-row"
        :class="{ muted: !item.monitored, offline: item.monitored && item.checkedAt && !item.online }"
      >
        <span class="status-dot" :class="dotClass(item)" :title="statusText(item)" />
        <div class="status-info">
          <strong>{{ item.title }}</strong>
          <span :title="item.url">{{ item.url || '未配置地址' }}</span>
        </div>
        <div class="status-metric">
          <b>{{ item.monitored ? uptimeText(item) : '—' }}</b>
          <small>24h 在线率</small>
          <span class="uptime-bar"><i :style="{ width: item.monitored ? uptimeWidth(item) : '0%' }" /></span>
        </div>
        <div class="status-metric">
          <b>{{ item.monitored && item.checkedAt ? (item.online ? `${item.latencyMs} ms` : '离线') : '—' }}</b>
          <small>最近延迟</small>
        </div>
        <div class="status-metric time">
          <b>{{ item.checkedAt ? timeAgo(item.checkedAt) : '—' }}</b>
          <small>最后检查</small>
        </div>
        <button
          class="mini"
          type="button"
          :disabled="!isAdmin || togglingItem === item.itemId"
          @click="toggleItem(item)"
        >
          {{ item.monitored ? '停止监测' : '开启监测' }}
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.svc-pane {
  max-width: 720px;
  margin: 0 auto;
}
.svc-head {
  margin-bottom: 18px;
}
.svc-head h2 {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 18px;
  font-weight: 650;
  color: var(--text-strong);
}
.svc-head p {
  margin: 5px 0 0;
  font-size: 12px;
  line-height: 1.65;
  color: var(--text-soft);
}
.svc-ico {
  color: var(--accent);
  flex-shrink: 0;
}
.svc-block {
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 14px 16px;
  background: rgba(var(--glass-rgb), 0.28);
  margin-bottom: 14px;
}
.svc-block h3 {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: 0 0 6px;
  font-size: 13px;
  font-weight: 650;
  color: var(--text-strong);
}
.svc-block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 4px;
}
.svc-block-head h3 {
  margin: 0;
}
.svc-block-head > span {
  font-size: 11px;
  color: var(--text-faint);
}
.svc-filter {
  display: inline-flex;
  flex-shrink: 0;
  padding: 2px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(var(--glass-rgb), 0.4);
}
.svc-filter button {
  padding: 3px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-soft);
  font-size: 10.5px;
  cursor: pointer;
}
.svc-filter button.active {
  background: rgba(var(--glass-rgb), 0.95);
  color: var(--text-strong);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}
.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}
.stat-grid > div {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 9px 11px;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: rgba(var(--glass-rgb), 0.36);
}
.stat-grid span {
  font-size: 10.5px;
  color: var(--text-faint);
}
.stat-grid strong {
  font-size: 18px;
  line-height: 1.25;
  color: var(--text-strong);
  font-variant-numeric: tabular-nums;
}
.stat-grid strong small {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-faint);
}
.stat-grid strong.ok {
  color: #16a34a;
}
.stat-grid strong.small-value {
  font-size: 13px;
  padding-top: 3px;
}
.svc-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}
.svc-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 13px;
  border: 1px solid var(--line);
  border-radius: 9px;
  background: rgba(var(--glass-rgb), 0.6);
  color: var(--text-strong);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.16s;
}
.svc-btn:hover {
  border-color: color-mix(in srgb, var(--accent) 35%, transparent);
}
.svc-btn.primary {
  border-color: transparent;
  background: var(--accent);
  color: #fff;
}
.svc-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.svc-hint {
  margin: 9px 0 0;
  font-size: 11px;
  line-height: 1.55;
  color: var(--text-faint);
}
.svc-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 9px 0;
  border-top: 1px dashed var(--line);
}
.svc-block h3 + .svc-row {
  border-top: none;
}
.row-label {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--text-strong);
}
.row-hint {
  margin-top: 2px;
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-faint);
}
.time-range {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-faint);
}
.svc-field {
  height: 30px;
  padding: 0 8px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(var(--glass-rgb), 0.55);
  color: var(--text-strong);
  font-size: 12px;
  color-scheme: light dark;
}
.svc-empty {
  padding: 18px 6px;
  text-align: center;
  font-size: 11.5px;
  color: var(--text-faint);
}
.status-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-top: 1px dashed var(--line);
}
.status-row.muted {
  opacity: 0.62;
}
.status-dot {
  width: 8px;
  height: 8px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--text-faint);
}
.status-dot.online {
  background: #22c55e;
  box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.14);
}
.status-dot.offline {
  background: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.15);
}
.status-dot.idle {
  opacity: 0.45;
}
.status-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.status-info strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12.5px;
  color: var(--text-strong);
}
.status-info span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 10.5px;
  color: var(--text-faint);
}
.status-metric {
  width: 84px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
  text-align: right;
}
.status-metric.time {
  width: 76px;
}
.status-metric b {
  font-size: 12px;
  color: var(--text-strong);
  font-variant-numeric: tabular-nums;
}
.status-metric small {
  font-size: 10px;
  color: var(--text-faint);
}
.uptime-bar {
  display: block;
  height: 3px;
  margin-top: 3px;
  border-radius: 999px;
  background: rgba(var(--glass-rgb), 0.7);
  overflow: hidden;
}
.uptime-bar i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #22c55e;
  transition: width 0.3s ease;
}
.status-row.offline .uptime-bar i {
  background: #ef4444;
}
.mini {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 26px;
  padding: 0 9px;
  flex-shrink: 0;
  border: 1px solid var(--line);
  border-radius: 7px;
  background: rgba(var(--glass-rgb), 0.55);
  color: var(--text-soft);
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s;
}
.mini:hover {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 35%, transparent);
}
.mini:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.toggle {
  position: relative;
  width: 40px;
  height: 22px;
  flex-shrink: 0;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: rgba(var(--glass-rgb), 0.6);
  cursor: pointer;
  transition: all 0.18s;
}
.toggle span {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--text-faint);
  transition: all 0.18s;
}
.toggle.active {
  border-color: transparent;
  background: var(--accent);
}
.toggle.active span {
  left: 20px;
  background: #fff;
}
.toggle:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.spin {
  animation: svc-spin 1s linear infinite;
}
@keyframes svc-spin {
  to {
    transform: rotate(360deg);
  }
}
@media (max-width: 640px) {
  .stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .status-row {
    flex-wrap: wrap;
    gap: 8px;
  }
  .status-info {
    flex-basis: calc(100% - 26px);
  }
  .status-metric {
    width: auto;
    flex: 1;
    text-align: left;
  }
}
</style>
