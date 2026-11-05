export function normalizeUrl(rawUrl?: string): string {
  if (!rawUrl) return ''
  const trimmed = rawUrl.trim()
  if (!trimmed) return ''
  if (/^https?:\/\//i.test(trimmed) || /^\/\//.test(trimmed) || /^[a-zA-Z0-9_-]+:\/\//.test(trimmed)) {
    return trimmed
  }
  return `http://${trimmed}`
}

export function isUrlLike(input: string): boolean {
  const value = input.trim()
  if (!value || /\s/.test(value)) return false
  return (
    /^https?:\/\//i.test(value) ||
    /^\/\//.test(value) ||
    /^[\w-]+(\.[\w-]+)+(\/.*)?$/.test(value) ||
    /^localhost(:\d+)?(\/.*)?$/i.test(value) ||
    /^\d{1,3}(\.\d{1,3}){3}(:\d+)?(\/.*)?$/.test(value)
  )
}

export function isPrivateHost(hostname: string): boolean {
  const host = hostname.toLowerCase()
  if (!host) return false
  if (host === 'localhost' || host.endsWith('.local') || host.endsWith('.lan')) return true
  if (/^\d{1,3}(\.\d{1,3}){3}$/.test(host)) {
    const parts = host.split('.').map(Number)
    if (parts[0] === 10) return true
    if (parts[0] === 127) return true
    if (parts[0] === 172 && parts[1] >= 16 && parts[1] <= 31) return true
    if (parts[0] === 192 && parts[1] === 168) return true
    if (parts[0] === 169 && parts[1] === 254) return true
  }
  return host === '::1'
}

export function isCurrentHostPrivate(): boolean {
  return isPrivateHost(window.location.hostname)
}

export async function probeLanReachable(url: string, timeout = 450): Promise<boolean> {
  const target = normalizeUrl(url)
  if (!target) return false
  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), timeout)
  try {
    await fetch(target, { mode: 'no-cors', signal: controller.signal, cache: 'no-store' })
    return true
  } catch {
    return false
  } finally {
    window.clearTimeout(timer)
  }
}

export function openExternal(url: string, newTab = true) {
  const target = normalizeUrl(url)
  if (!target) return
  if (target.startsWith('//')) {
    window.open(`http:${target}`, '_blank', 'noopener,noreferrer')
    return
  }
  if (newTab) {
    window.open(target, '_blank', 'noopener,noreferrer')
  } else {
    window.location.href = target
  }
}
