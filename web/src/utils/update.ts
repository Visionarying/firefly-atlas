export interface UpdateCheck {
  current: string
  latest: string
  url: string
  name: string
  hasUpdate: boolean
}

const RELEASE_API = 'https://api.github.com/repos/Visionarying/firefly-atlas/releases/latest'

export function compareVersions(a: string, b: string): number {
  const clean = (value: string) => value.replace(/^v/i, '').split(/[.+-]/).map((part) => Number(part) || 0)
  const left = clean(a)
  const right = clean(b)
  const length = Math.max(left.length, right.length)
  for (let index = 0; index < length; index += 1) {
    const diff = (left[index] || 0) - (right[index] || 0)
    if (diff !== 0) return diff > 0 ? 1 : -1
  }
  return 0
}

/**
 * Best-effort GitHub release check. Network failures return null so the panel
 * never blocks or shows an error just because GitHub is unreachable.
 */
export async function checkForUpdate(current: string, timeoutMs = 6000): Promise<UpdateCheck | null> {
  if (!current || /dev/i.test(current)) return null
  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), timeoutMs)
  try {
    const response = await fetch(RELEASE_API, {
      headers: { Accept: 'application/vnd.github+json' },
      signal: controller.signal,
    })
    if (!response.ok) return null
    const release = (await response.json()) as {
      tag_name?: string
      html_url?: string
      name?: string
    }
    const latest = (release.tag_name || '').replace(/^v/i, '')
    if (!latest) return null
    return {
      current,
      latest,
      url: release.html_url || 'https://github.com/Visionarying/firefly-atlas/releases',
      name: release.name || `v${latest}`,
      hasUpdate: compareVersions(latest, current) > 0,
    }
  } catch {
    return null
  } finally {
    window.clearTimeout(timer)
  }
}
