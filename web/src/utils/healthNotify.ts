import type { HealthStatusItem } from '@/typings'

const NOTIFY_KEY = 'firefly_health_notify'
const SEEN_KEY = 'firefly_health_alert_seen'
const BASELINE_KEY = 'firefly_health_alert_baseline'

export function notificationsSupported(): boolean {
  return typeof window !== 'undefined' && 'Notification' in window
}

export function notificationPermission(): NotificationPermission | 'unsupported' {
  if (!notificationsSupported()) return 'unsupported'
  return Notification.permission
}

export function notificationsEnabled(): boolean {
  return localStorage.getItem(NOTIFY_KEY) !== '0'
}

export function setNotificationsEnabled(enabled: boolean) {
  localStorage.setItem(NOTIFY_KEY, enabled ? '1' : '0')
}

export async function requestNotificationPermission(): Promise<NotificationPermission | 'unsupported'> {
  if (!notificationsSupported()) return 'unsupported'
  try {
    return await Notification.requestPermission()
  } catch {
    return Notification.permission
  }
}

function loadSeen(): Record<string, number> {
  try {
    const raw = localStorage.getItem(SEEN_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function saveSeen(seen: Record<string, number>) {
  // Keep the map small even after many cards were renamed or deleted.
  const entries = Object.entries(seen)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 200)
  localStorage.setItem(SEEN_KEY, JSON.stringify(Object.fromEntries(entries)))
}

/**
 * Record the current sample time of every card without notifying. Used the
 * first time notifications are enabled so historical offline samples do not
 * produce a burst of alerts.
 */
export function baselineHealthAlerts(items: HealthStatusItem[]) {
  const seen = loadSeen()
  for (const item of items) {
    if (item.checkedAt) seen[item.itemId] = item.checkedAt
  }
  saveSeen(seen)
  localStorage.setItem(BASELINE_KEY, '1')
}

/**
 * Show one browser notification per new online -> offline sample. The backend
 * marks only real transitions outside maintenance windows with alert = true.
 */
export function notifyHealthAlerts(items: HealthStatusItem[]) {
  if (!notificationsEnabled() || !notificationsSupported()) return
  if (Notification.permission !== 'granted') return
  if (localStorage.getItem(BASELINE_KEY) !== '1') {
    baselineHealthAlerts(items)
    return
  }

  const seen = loadSeen()
  let changed = false
  for (const item of items) {
    if (!item.checkedAt) continue
    const last = seen[item.itemId] || 0
    if (item.checkedAt <= last) continue
    seen[item.itemId] = item.checkedAt
    changed = true
    if (!item.alert) continue
    const when = new Date(item.checkedAt).toLocaleTimeString('zh-CN', { hour12: false })
    try {
      new Notification(`${item.title} 已离线`, {
        body: `${item.url || '服务地址'} 在 ${when} 无法访问`,
        tag: `firefly-health-${item.itemId}`,
      })
    } catch {
      // Some browsers require a service worker; ignoring keeps polling alive.
    }
  }
  if (changed) saveSeen(seen)
}
