import { computed, reactive, ref, toRaw } from 'vue'
import { defineStore } from 'pinia'
import { panelApi } from '@/api'
import { isCurrentHostPrivate, normalizeUrl, openExternal, probeLanReachable } from '@/utils/url'
import { defaultConfig, defaultSearchEngines } from './defaults'
import { useAuthStore } from './auth'
import type { ItemIcon, PanelConfig, PanelData, PanelSettings, SearchEngine, ThemeMode } from '@/typings'

export type LanMode = 'auto' | 'lan' | 'wan'

const mergeConfig = (incoming?: Partial<PanelConfig> | null): PanelConfig => ({
  ...defaultConfig,
  ...(incoming || {}),
})

export const usePanelStore = defineStore('panel', () => {
  const auth = useAuthStore()

  const config = reactive<PanelConfig>(mergeConfig())
  const searchEngines = ref<SearchEngine[]>([...defaultSearchEngines])
  const groups = ref<{ id: number; title: string; sort: number }[]>([])
  const items = ref<ItemIcon[]>([])
  const settings = ref<PanelSettings>({})
  const loaded = ref(false)
  const loading = ref(false)
  const publicBlocked = ref(false)
  const editMode = ref(false)
  const viewer = ref<{ url: string; title: string } | null>(null)

  const lanMode = ref<LanMode>((localStorage.getItem('firefly_lan_mode') as LanMode) || 'auto')
  const lanReachable = ref(false)
  const lanProbing = ref(false)
  const systemDark = ref(window.matchMedia('(prefers-color-scheme: dark)').matches)
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    systemDark.value = e.matches
  })

  const isDark = computed(
    () =>
      config.themeMode === 'dark' ||
      (config.themeMode === 'auto' && systemDark.value),
  )

  const isPublic = computed(() => !auth.isLoggedIn)
  const canEdit = computed(() => auth.isLoggedIn)

  const itemsByGroup = computed(() => {
    const map = new Map<number, ItemIcon[]>()
    for (const group of groups.value) map.set(group.id, [])
    for (const item of items.value) {
      if (!map.has(item.itemIconGroupId)) map.set(item.itemIconGroupId, [])
      map.get(item.itemIconGroupId)!.push(item)
    }
    for (const list of map.values()) list.sort((a, b) => a.sort - b.sort || a.id - b.id)
    return map
  })

  function applyData(data: PanelData) {
    Object.assign(config, mergeConfig(data.panelConfig))
    if (Array.isArray(data.searchEngines) && data.searchEngines.length) {
      searchEngines.value = data.searchEngines
    }
    groups.value = (data.groups || []).slice().sort((a, b) => a.sort - b.sort || a.id - b.id)
    items.value = (data.items || []).slice().sort((a, b) => a.sort - b.sort || a.id - b.id)
    settings.value = data.settings || {}
    loaded.value = true
  }

  async function load() {
    loading.value = true
    publicBlocked.value = false
    try {
      const data = auth.isLoggedIn ? await panelApi.index() : await panelApi.public()
      if (!auth.isLoggedIn && !data.publicVisitMode) {
        publicBlocked.value = true
        loaded.value = true
        return
      }
      applyData(data)
      if (auth.isLoggedIn && auth.isAdmin) {
        // Admin sees everything; standard users keep their own space.
      }
      if (lanMode.value === 'auto') await detectLan()
    } finally {
      loading.value = false
    }
  }

  async function detectLan() {
    if (isCurrentHostPrivate()) {
      lanReachable.value = true
      return
    }
    const candidate = items.value.find((it) => it.lanUrl && it.lanUrl.trim())
    if (!candidate) {
      lanReachable.value = false
      return
    }
    lanProbing.value = true
    try {
      lanReachable.value = await probeLanReachable(candidate.lanUrl)
    } finally {
      lanProbing.value = false
    }
  }

  function resolveUrl(item: ItemIcon): string {
    const lanUrl = normalizeUrl(item.lanUrl)
    const wanUrl = normalizeUrl(item.url)
    if (lanMode.value === 'lan') return lanUrl || wanUrl
    if (lanMode.value === 'wan') return wanUrl || lanUrl
    if (lanReachable.value) return lanUrl || wanUrl
    return wanUrl || lanUrl
  }

  function openItem(item: ItemIcon) {
    const url = resolveUrl(item)
    if (!url) return
    if (item.openMethod === 3) {
      viewer.value = { url, title: item.title }
      return
    }
    openExternal(url, item.openMethod !== 2)
  }

  function openViewer(url: string, title = '') {
    if (!url) return
    viewer.value = { url, title }
  }

  function closeViewer() {
    viewer.value = null
  }

  function setLanMode(mode: LanMode) {
    lanMode.value = mode
    localStorage.setItem('firefly_lan_mode', mode)
    if (mode === 'auto') void detectLan()
  }

  function cycleLanMode() {
    const order: LanMode[] = ['auto', 'lan', 'wan']
    const next = order[(order.indexOf(lanMode.value) + 1) % order.length]
    setLanMode(next)
  }

  let saveTimer: number | undefined
  let savingPromise: Promise<void> | null = null

  async function saveNow() {
    if (!auth.isLoggedIn) return
    if (savingPromise) await savingPromise
    savingPromise = panelApi
      .saveConfig(toRaw(config), searchEngines.value)
      .then(() => undefined)
      .catch(() => undefined)
      .finally(() => {
        savingPromise = null
      })
    await savingPromise
  }

  function scheduleSave() {
    if (!auth.isLoggedIn) return
    window.clearTimeout(saveTimer)
    saveTimer = window.setTimeout(() => {
      void saveNow()
    }, 500)
  }

  function setTheme(mode: ThemeMode) {
    config.themeMode = mode
    scheduleSave()
  }

  return {
    config,
    searchEngines,
    groups,
    items,
    settings,
    loaded,
    loading,
    publicBlocked,
    editMode,
    viewer,
    lanMode,
    lanReachable,
    lanProbing,
    isDark,
    isPublic,
    canEdit,
    itemsByGroup,
    load,
    applyData,
    detectLan,
    resolveUrl,
    openItem,
    openViewer,
    closeViewer,
    setLanMode,
    cycleLanMode,
    saveNow,
    scheduleSave,
    setTheme,
  }
})
