<script setup lang="ts">
import { computed, h, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NDropdown, useDialog, useMessage, type DropdownOption } from 'naive-ui'
import {
  Activity,
  Check,
  ExternalLink,
  Globe2,
  House,
  LayoutGrid,
  LogIn,
  LogOut,
  Moon,
  MoveRight,
  PanelTopOpen,
  Pencil,
  Plus,
  Settings,
  Sun,
  Trash2,
  UserRound,
  X,
} from 'lucide-vue-next'
import Background from '@/components/Background.vue'
import Clock from '@/components/Clock.vue'
import SearchBar from '@/components/SearchBar.vue'
import GroupSection from '@/components/GroupSection.vue'
import FooterBar from '@/components/FooterBar.vue'
import LanBadge from '@/components/LanBadge.vue'
import SettingsModal from '@/components/SettingsModal.vue'
import EditItemModal from '@/components/EditItemModal.vue'
import PromptModal from '@/components/PromptModal.vue'
import NoticeBanner from '@/components/NoticeBanner.vue'
import WebViewModal from '@/components/WebViewModal.vue'
import SystemMonitor from '@/components/SystemMonitor.vue'
import { usePanelStore } from '@/store/panel'
import { useAuthStore } from '@/store/auth'
import { panelApi, systemApi, healthApi } from '@/api'
import { normalizeUrl, openExternal } from '@/utils/url'
import { notifyHealthAlerts } from '@/utils/healthNotify'
import { checkForUpdate, type UpdateCheck } from '@/utils/update'
import type { ItemGroup, ItemIcon } from '@/typings'

const panel = usePanelStore()
const auth = useAuthStore()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const config = panel.config

const settingsShow = ref(false)
const settingsRef = ref<InstanceType<typeof SettingsModal> | null>(null)
const itemModalShow = ref(false)
const editingItem = ref<ItemIcon | null>(null)
const defaultGroupId = ref<number | undefined>(undefined)

const promptShow = ref(false)
const promptMode = ref<'add' | 'rename'>('add')
const promptTitle = ref('')
const promptValue = ref('')
const promptGroup = ref<ItemGroup | null>(null)

const pageStyle = computed(() => ({
  maxWidth: `${config.maxWidth}${config.maxWidthUnit}`,
  padding: `${config.marginTop}vh ${config.marginX}px ${config.marginBottom}vh`,
  '--item-gap': `${config.itemGap}px`,
}))

const pageColumns = computed(() => panel.currentPage?.columns || 1)

async function setPageColumns(columns: number) {
  const page = panel.currentPage
  if (!page || page.columns === columns) return
  const previous = page.columns
  page.columns = columns
  try {
    await panelApi.savePage({ id: page.id, title: page.title, columns })
  } catch (e: any) {
    page.columns = previous
    message.error(e.message || '切换分栏失败')
  }
}

async function openPageSettings() {
  settingsShow.value = true
  await nextTick()
  settingsRef.value?.openSection('groups')
}

async function openServiceStatus() {
  settingsShow.value = true
  await nextTick()
  settingsRef.value?.openSection('status')
}

const flatCurrentItems = computed(() =>
  panel.currentGroups.flatMap((group) => panel.itemsByGroup.get(group.id) || []),
)

const updateInfo = ref<UpdateCheck | null>(null)
const UPDATE_CHECK_INTERVAL = 24 * 60 * 60 * 1000

async function checkUpdateNotice() {
  if (!auth.isLoggedIn) return
  const dismissed = localStorage.getItem('firefly_update_dismissed') || ''
  try {
    const cached = JSON.parse(localStorage.getItem('firefly_update_cache') || 'null') as
      | (UpdateCheck & { at: number })
      | null
    if (cached && Date.now() - cached.at < UPDATE_CHECK_INTERVAL) {
      if (cached.hasUpdate && cached.latest !== dismissed) updateInfo.value = cached
      return
    }
    const info = await systemApi.version()
    const result = await checkForUpdate(info.version)
    if (!result) return
    localStorage.setItem('firefly_update_cache', JSON.stringify({ ...result, at: Date.now() }))
    if (result.hasUpdate && result.latest !== dismissed) updateInfo.value = result
  } catch {
    /* update prompts are best-effort */
  }
}

function dismissUpdate() {
  if (updateInfo.value) localStorage.setItem('firefly_update_dismissed', updateInfo.value.latest)
  updateInfo.value = null
}

function onNumericShortcut(event: KeyboardEvent) {
  if (event.ctrlKey || event.metaKey || event.altKey || event.shiftKey) return
  if (!/^[0-9]$/.test(event.key)) return
  if (settingsShow.value || itemModalShow.value || promptShow.value || panel.viewer) return
  const target = event.target as HTMLElement | null
  if (target && (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))) {
    return
  }
  const index = event.key === '0' ? 9 : Number(event.key) - 1
  const item = flatCurrentItems.value[index]
  if (!item) return
  event.preventDefault()
  onOpenItem(item)
}

const logoStyle = computed(() => ({
  fontFamily: config.logoFontFamily,
  color: config.logoColor || 'var(--text-strong)',
  opacity: Math.max(0, config.headerOpacity) / 100,
}))

const heroClass = computed(() => ({
  left: config.headerAlign === 'left',
  glass: config.logoGlassEffect,
}))

const userMenuOptions = [
  { label: '系统设置', key: 'settings' },
  { label: '退出登录', key: 'logout' },
]

/* ---------- right-click menu ---------- */
const ctx = reactive<{ show: boolean; x: number; y: number; item: ItemIcon | null }>({
  show: false,
  x: 0,
  y: 0,
  item: null,
})

const ctxOptions = computed<DropdownOption[]>(() => {
  const item = ctx.item
  if (!item) return []
  const options: DropdownOption[] = []
  if (item.lanUrl) {
    options.push({ label: '打开内网地址', key: 'lan', icon: () => h(House, { size: 15 }) })
  }
  if (item.url) {
    options.push({ label: '打开公网地址', key: 'wan', icon: () => h(Globe2, { size: 15 }) })
  }
  options.push({ label: '在新窗口打开', key: 'newtab', icon: () => h(ExternalLink, { size: 15 }) })
  options.push({ label: '页内弹窗打开', key: 'viewer', icon: () => h(PanelTopOpen, { size: 15 }) })
  if (auth.isLoggedIn) {
    options.push({ type: 'divider', key: 'd1' })
    const targets = panel.groups.filter((group) => group.id !== item.itemIconGroupId)
    if (targets.length > 0) {
      options.push({
        label: '移动到分组',
        key: 'move',
        icon: () => h(MoveRight, { size: 15 }),
        children: targets.map((group) => {
          const page = panel.pages.find((entry) => entry.id === group.pageId)
          return {
            label: page && panel.pages.length > 1 ? `${page.title} / ${group.title}` : group.title,
            key: `move:${group.id}`,
          }
        }),
      })
    }
    options.push({ label: '编辑卡片', key: 'edit', icon: () => h(Pencil, { size: 15 }) })
    options.push({ label: '删除卡片', key: 'delete', icon: () => h(Trash2, { size: 15 }) })
  }
  return options
})

function openContext(item: ItemIcon, event: MouseEvent) {
  event.preventDefault()
  ctx.item = item
  ctx.x = event.clientX
  ctx.y = event.clientY
  ctx.show = true
}

function openContextUrl(url: string, item: ItemIcon, forceNewTab = false) {
  const target = normalizeUrl(url)
  if (!target) return
  if (forceNewTab) {
    openExternal(target, true)
    return
  }
  if (item.openMethod === 3) {
    panel.openViewer(target, item.title)
    return
  }
  openExternal(target, item.openMethod !== 2)
}

function onCtxSelect(key: string) {
  const item = ctx.item
  ctx.show = false
  if (!item) return
  if (key.startsWith('move:')) {
    void moveItemToGroup(item, Number(key.slice(5)))
    return
  }
  if (key === 'lan') openContextUrl(item.lanUrl, item)
  else if (key === 'wan') openContextUrl(item.url, item)
  else if (key === 'newtab') openContextUrl(panel.resolveUrl(item), item, true)
  else if (key === 'viewer') panel.openViewer(panel.resolveUrl(item), item.title)
  else if (key === 'edit') openEditItem(item)
  else if (key === 'delete') deleteItem(item)
}

async function moveItemToGroup(item: ItemIcon, groupId: number) {
  if (!groupId) return
  try {
    await panelApi.moveItem(item.id, groupId)
    const group = panel.groups.find((entry) => entry.id === groupId)
    item.itemIconGroupId = groupId
    item.sort = Number.MAX_SAFE_INTEGER
    if (group && group.pageId !== panel.currentPageId) {
      message.success(`已移动到「${group.title}」`)
    } else {
      message.success('已移动卡片')
    }
  } catch (e: any) {
    message.error(e.message || '移动失败')
  }
}

const notice = computed(() => {
  const settings = panel.settings
  if (settings.notice_enabled !== '1' || !settings.notice_content) return null
  const scope = settings.notice_scope || 'home'
  if (scope !== 'home' && scope !== 'both') return null
  return { title: settings.notice_title || '公告', content: settings.notice_content }
})

/* ---------- service health ---------- */
const healthMap = ref<Record<number, { online: boolean; status?: number; latencyMs?: number }>>({})
let healthTimer: number | undefined

async function refreshHealth() {
  if (!auth.isLoggedIn) return
  const items = panel.items.filter((item) => item.healthCheck)
  const urls: string[] = []
  const byItem: Record<number, string> = {}
  for (const item of items) {
    const url = panel.resolveUrl(item)
    if (!url) continue
    urls.push(url)
    byItem[item.id] = url
  }
  if (urls.length === 0) {
    healthMap.value = {}
    return
  }
  try {
    const results = await panelApi.health(Array.from(new Set(urls)))
    const byUrl = new Map(results.map((result) => [result.url, result]))
    const next: typeof healthMap.value = {}
    for (const [id, url] of Object.entries(byItem)) {
      const result = byUrl.get(url)
      if (result) {
        next[Number(id)] = {
          online: result.online,
          status: result.status,
          latencyMs: result.latencyMs,
        }
      }
    }
    healthMap.value = next
  } catch {
    /* health checks are best-effort */
  }
}

/* ---------- v1.3 service status notifications ---------- */
async function refreshHealthAlerts() {
  if (!auth.isLoggedIn) return
  try {
    const overview = await healthApi.overview()
    notifyHealthAlerts(overview.items)
  } catch {
    /* notification polling is best-effort */
  }
}

onMounted(async () => {
  window.addEventListener('keydown', onNumericShortcut)
  try {
    await panel.load()
    if (panel.publicBlocked) {
      router.replace('/login')
      return
    }
    await refreshHealth()
    void refreshHealthAlerts()
    healthTimer = window.setInterval(() => {
      void refreshHealth()
      void refreshHealthAlerts()
    }, 60000)
    void checkUpdateNotice()
  } catch {
    if (!auth.isLoggedIn) router.replace('/login')
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', onNumericShortcut)
  if (healthTimer) window.clearInterval(healthTimer)
})

watch(
  () => panel.config,
  () => {
    if (panel.loaded && auth.isLoggedIn) panel.scheduleSave()
  },
  { deep: true },
)

function toggleTheme() {
  if (!auth.isLoggedIn) return
  panel.setTheme(panel.isDark ? 'light' : 'dark')
}

function openAddItem(group: ItemGroup) {
  if (!auth.isLoggedIn) return
  editingItem.value = null
  defaultGroupId.value = group.id
  itemModalShow.value = true
}

function openEditItem(item: ItemIcon) {
  editingItem.value = item
  defaultGroupId.value = item.itemIconGroupId
  itemModalShow.value = true
}

function onOpenItem(item: ItemIcon, forceNewTab?: boolean) {
  if (forceNewTab) {
    openExternal(panel.resolveUrl(item), true)
    return
  }
  panel.openItem(item)
}

function deleteItem(item: ItemIcon) {
  dialog.warning({
    title: '删除卡片',
    content: `确定删除「${item.title}」吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await panelApi.deleteItem(item.id)
        panel.items = panel.items.filter((it) => it.id !== item.id)
        message.success('卡片已删除')
      } catch (e: any) {
        message.error(e.message || '删除失败')
      }
    },
  })
}

function openAddGroup() {
  promptMode.value = 'add'
  promptTitle.value = '新建分组'
  promptValue.value = ''
  promptGroup.value = null
  promptShow.value = true
}

function openRenameGroup(group: ItemGroup) {
  promptMode.value = 'rename'
  promptTitle.value = '重命名分组'
  promptValue.value = group.title
  promptGroup.value = group
  promptShow.value = true
}

async function onPromptConfirm(value: string) {
  try {
    if (promptMode.value === 'add') {
      const res = await panelApi.saveGroup({ title: value, pageId: panel.currentPageId })
      panel.groups.push({ id: res.id, pageId: panel.currentPageId, title: value, sort: panel.groups.length })
      message.success('分组已创建')
    } else if (promptGroup.value) {
      await panelApi.saveGroup({ id: promptGroup.value.id, pageId: promptGroup.value.pageId, title: value })
      promptGroup.value.title = value
      message.success('分组已重命名')
    }
  } catch (e: any) {
    message.error(e.message || '操作失败')
  }
}

function removeGroup(group: ItemGroup) {
  dialog.warning({
    title: '删除分组',
    content: `确定删除「${group.title}」及组内所有卡片吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await panelApi.deleteGroup(group.id)
        panel.groups = panel.groups.filter((g) => g.id !== group.id)
        panel.items = panel.items.filter((it) => it.itemIconGroupId !== group.id)
        message.success('分组已删除')
      } catch (e: any) {
        message.error(e.message || '删除失败')
      }
    },
  })
}

async function onSortItems(ids: number[]) {
  try {
    await panelApi.sortItems(ids)
    ids.forEach((id, index) => {
      const item = panel.items.find((it) => it.id === id)
      if (item) item.sort = index
    })
  } catch {
    /* silent */
  }
}

async function reload() {
  try {
    await panel.load()
    await refreshHealth()
  } catch {
    /* ignore */
  }
}

function onUserMenu(key: string) {
  if (key === 'settings') settingsShow.value = true
  if (key === 'logout') {
    void auth.logout().then(() => router.push('/login'))
  }
}
</script>

<template>
  <div class="home">
    <Background />

    <div class="topbar">
      <span v-if="!auth.isLoggedIn" class="guest-badge">访客浏览</span>

      <button v-if="auth.isLoggedIn" class="icon-button" :title="panel.isDark ? '切换浅色' : '切换深色'" @click="toggleTheme">
        <Sun v-if="panel.isDark" :size="16" />
        <Moon v-else :size="16" />
      </button>

      <button
        v-if="auth.isLoggedIn"
        class="icon-button"
        :class="{ active: panel.editMode }"
        :title="panel.editMode ? '完成编辑' : '编辑布局'"
        @click="panel.editMode = !panel.editMode"
      >
        <Check v-if="panel.editMode" :size="16" />
        <Pencil v-else :size="16" />
      </button>

      <button v-if="auth.isLoggedIn" class="icon-button" title="服务状态" @click="openServiceStatus">
        <Activity :size="16" />
      </button>

      <button v-if="auth.isLoggedIn" class="icon-button" title="系统设置" @click="settingsShow = true">
        <Settings :size="16" />
      </button>

      <n-dropdown v-if="auth.isLoggedIn" trigger="click" :options="userMenuOptions" @select="onUserMenu">
        <button class="icon-button" title="账号">
          <UserRound :size="16" />
        </button>
      </n-dropdown>

      <button v-else class="login-btn" @click="router.push('/login')">
        <LogIn :size="15" />
        <span>登录</span>
      </button>
    </div>

    <main class="page" :style="pageStyle">
      <nav v-if="panel.pages.length > 1 || auth.isLoggedIn" class="page-tabs">
        <button
          v-for="page in panel.pages"
          :key="page.id"
          type="button"
          class="page-tab"
          :class="{ active: page.id === panel.currentPageId }"
          @click="panel.setCurrentPage(page.id)"
        >
          <LayoutGrid :size="13" />
          {{ page.title }}
        </button>
        <span v-if="auth.isLoggedIn" class="page-divider" />
        <button
          v-for="columns in [1, 2, 3]"
          :key="columns"
          type="button"
          class="page-tab col-tab"
          :class="{ active: pageColumns === columns }"
          :title="`当前页面显示为 ${columns} 列`"
          @click="setPageColumns(columns)"
        >
          {{ columns }} 列
        </button>
        <button v-if="auth.isLoggedIn" type="button" class="page-tab page-add" @click="openPageSettings">
          <Plus :size="13" />
          新建页面
        </button>
      </nav>

      <div v-if="auth.isLoggedIn && updateInfo?.hasUpdate" class="update-banner">
        <span class="update-dot" />
        <span>
          新版本 <strong>v{{ updateInfo.latest }}</strong> 可用
          <em v-if="updateInfo.name && updateInfo.name !== `v${updateInfo.latest}`">{{ updateInfo.name }}</em>
        </span>
        <button class="update-link" type="button" @click="openExternal(updateInfo.url, true)">查看发布页</button>
        <button class="update-close" type="button" title="本次不再提示" @click="dismissUpdate">
          <X :size="13" />
        </button>
      </div>

      <NoticeBanner v-if="notice" :title="notice.title" :content="notice.content" />

      <header v-if="config.showHeader" class="hero" :class="heroClass">
        <h1 class="logo" :style="logoStyle">{{ config.logoText }}</h1>
        <Clock v-if="config.showClock" />
      </header>

      <SearchBar v-if="config.showSearch" />

      <SystemMonitor v-if="config.showSystemMonitor" />

      <div class="groups">
        <div class="groups-grid" :class="`cols-${pageColumns}`">
          <GroupSection
            v-for="group in panel.currentGroups"
            :key="group.id"
            :group="group"
            :items="panel.itemsByGroup.get(group.id) || []"
            :edit-mode="panel.editMode"
            :health-map="healthMap"
            @open="onOpenItem"
            @edit="openEditItem"
            @delete="deleteItem"
            @context="openContext"
            @add="openAddItem"
            @rename="openRenameGroup"
            @remove="removeGroup"
            @sort="onSortItems"
          />
        </div>

        <div v-if="panel.loaded && panel.currentGroups.length === 0" class="empty-state">
          <div class="empty-icon">F</div>
          <h3>还没有任何分组</h3>
          <p>{{ auth.isLoggedIn ? '点击右上角编辑按钮，开始搭建你的导航面板。' : '管理员还没有公开任何内容。' }}</p>
          <button v-if="auth.isLoggedIn" class="primary-btn" @click="openAddGroup">
            <Plus :size="15" /> 新建分组
          </button>
        </div>

        <button v-if="auth.isLoggedIn && panel.editMode" class="add-group" @click="openAddGroup">
          <Plus :size="15" />
          新建分组
        </button>
      </div>

      <FooterBar />
    </main>

    <LanBadge v-if="!panel.publicBlocked" />

    <n-dropdown
      trigger="manual"
      placement="bottom-start"
      :show="ctx.show"
      :x="ctx.x"
      :y="ctx.y"
      :options="ctxOptions"
      @select="onCtxSelect"
      @clickoutside="ctx.show = false"
    />

    <div v-if="panel.loading && !panel.loaded" class="loading">
      <span class="spinner" />
    </div>

    <SettingsModal v-if="auth.isLoggedIn" ref="settingsRef" v-model:show="settingsShow" />

    <EditItemModal
      v-if="auth.isLoggedIn"
      v-model:show="itemModalShow"
      :item="editingItem"
      :groups="panel.groups"
      :default-group-id="defaultGroupId"
      @saved="reload"
    />

    <WebViewModal />

    <PromptModal
      v-model:show="promptShow"
      :title="promptTitle"
      :value="promptValue"
      :confirm-text="promptMode === 'add' ? '创建' : '保存'"
      @confirm="onPromptConfirm"
    />
  </div>
</template>

<style scoped>
.home {
  position: relative;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
}

.topbar {
  position: fixed;
  top: 16px;
  right: 18px;
  z-index: 40;
  display: flex;
  align-items: center;
  gap: 8px;
}

.icon-button.active {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 45%, transparent);
  background: color-mix(in srgb, var(--accent) 12%, transparent);
}

.guest-badge {
  font-size: 10.5px;
  letter-spacing: 0.08em;
  padding: 5px 10px;
  border-radius: 999px;
  color: var(--text-soft);
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.45);
  backdrop-filter: blur(12px);
}

.login-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 38px;
  padding: 0 14px;
  border-radius: 12px;
  border: 1px solid var(--glass-border);
  background: rgba(var(--glass-rgb), 0.55);
  backdrop-filter: blur(16px);
  color: var(--text-strong);
  font-size: 12.5px;
  cursor: pointer;
  transition: all 0.2s;
}
.login-btn:hover {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 40%, transparent);
  transform: translateY(-1px);
}

.page {
  width: 100%;
  min-height: 100dvh;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

.hero {
  text-align: center;
  padding: 4vh 0 0;
  animation: fadeUp 0.7s cubic-bezier(0.22, 1, 0.36, 1) both;
}
.hero.left {
  text-align: left;
}

.logo {
  display: inline-block;
  margin: 0;
  font-size: clamp(32px, 5.2vw, 62px);
  font-weight: 600;
  line-height: 1.15;
  letter-spacing: 0.03em;
  text-shadow: 0 12px 40px rgba(var(--shadow-color), 0.16);
}

.hero.glass .logo {
  padding: 8px 26px;
  border-radius: 999px;
  background: rgba(var(--glass-rgb), 0.35);
  border: 1px solid var(--glass-border);
  backdrop-filter: blur(20px) saturate(160%);
  -webkit-backdrop-filter: blur(20px) saturate(160%);
  box-shadow: 0 18px 50px -28px rgba(var(--shadow-color), 0.5);
}

.page-tabs {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 20px auto 0;
  padding: 4px;
  width: fit-content;
  max-width: 100%;
  border-radius: 999px;
  border: 1px solid var(--glass-border);
  background: rgba(var(--glass-rgb), 0.42);
  backdrop-filter: blur(18px);
}
.page-tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 32px;
  padding: 0 13px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-soft);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.16s;
}
.page-tab:hover {
  color: var(--text-strong);
}
.page-tab.active {
  color: var(--accent-strong, #059669);
  background: color-mix(in srgb, var(--accent) 12%, transparent);
}
.page-divider {
  width: 1px;
  height: 18px;
  margin: 0 2px;
  background: var(--line);
}
.col-tab {
  font-size: 11px;
  padding: 0 9px;
  opacity: 0.72;
}
.col-tab.active {
  opacity: 1;
}
.page-add {
  border: 1px dashed color-mix(in srgb, var(--accent) 42%, transparent);
  color: var(--accent-strong, #059669);
}
.page-add:hover {
  background: color-mix(in srgb, var(--accent) 9%, transparent);
}

.update-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  width: fit-content;
  max-width: 100%;
  margin: 14px auto 0;
  padding: 7px 10px 7px 12px;
  border-radius: 999px;
  border: 1px solid color-mix(in srgb, var(--accent) 32%, transparent);
  background: color-mix(in srgb, var(--accent) 9%, rgba(var(--glass-rgb), 0.5));
  backdrop-filter: blur(16px);
  color: var(--text-soft);
  font-size: 12px;
}
.update-banner strong {
  color: var(--accent-strong, #059669);
}
.update-banner em {
  margin-left: 5px;
  font-style: normal;
  color: var(--text-faint);
}
.update-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 16%, transparent);
}
.update-link {
  border: none;
  background: transparent;
  color: var(--accent-strong, #059669);
  font-size: 12px;
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
}
.update-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--text-faint);
  cursor: pointer;
}
.update-close:hover {
  background: rgba(var(--glass-rgb), 0.7);
  color: var(--text-strong);
}

.groups-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 0 22px;
  align-items: start;
}
.groups-grid.cols-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.groups-grid.cols-3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
@media (max-width: 760px) {
  .groups-grid.cols-2,
  .groups-grid.cols-3 {
    grid-template-columns: minmax(0, 1fr);
  }
}

.groups {
  width: 100%;
  flex: 1;
}

.empty-state {
  margin-top: 60px;
  padding: 44px 24px;
  text-align: center;
  border: 1px dashed var(--line);
  border-radius: 20px;
  background: rgba(var(--glass-rgb), 0.28);
  backdrop-filter: blur(16px);
  animation: fadeUp 0.5s ease both;
}
.empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: 16px;
  background: linear-gradient(135deg, var(--accent), color-mix(in srgb, var(--accent) 50%, #3b82f6));
  color: #fff;
  font-size: 22px;
  font-weight: 700;
  margin-bottom: 14px;
}
.empty-state h3 {
  margin: 0 0 6px;
  font-size: 15px;
  color: var(--text-strong);
}
.empty-state p {
  margin: 0 0 18px;
  font-size: 12px;
  color: var(--text-soft);
}

.primary-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 10px;
  background: var(--accent);
  color: #fff;
  font-size: 12.5px;
  cursor: pointer;
  box-shadow: 0 12px 26px -14px color-mix(in srgb, var(--accent) 80%, transparent);
  transition: all 0.18s;
}
.primary-btn:hover {
  transform: translateY(-1px);
  filter: brightness(1.05);
}

.add-group {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  width: 100%;
  margin-top: 26px;
  padding: 13px;
  border: 1px dashed var(--line);
  border-radius: 14px;
  background: rgba(var(--glass-rgb), 0.22);
  color: var(--text-soft);
  font-size: 12.5px;
  cursor: pointer;
  transition: all 0.2s;
  backdrop-filter: blur(10px);
}
.add-group:hover {
  border-color: color-mix(in srgb, var(--accent) 40%, transparent);
  color: var(--accent-strong, #059669);
  background: rgba(var(--glass-rgb), 0.38);
}

.loading {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--panel-bg) 55%, transparent);
  backdrop-filter: blur(4px);
}
.spinner {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 2px solid color-mix(in srgb, var(--accent) 25%, transparent);
  border-top-color: var(--accent);
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 560px) {
  .topbar {
    top: 10px;
    right: 10px;
    gap: 6px;
  }
  .icon-button {
    width: 34px;
    height: 34px;
    border-radius: 10px;
  }
  .login-btn {
    height: 34px;
    padding: 0 11px;
  }
  .guest-badge {
    display: none;
  }
  .hero {
    padding-top: 3vh;
  }
}
</style>
