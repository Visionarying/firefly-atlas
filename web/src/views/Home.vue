<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NDropdown, useDialog, useMessage } from 'naive-ui'
import {
  Check,
  LogIn,
  LogOut,
  Moon,
  Pencil,
  Plus,
  Settings,
  Sun,
  UserRound,
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
import { usePanelStore } from '@/store/panel'
import { useAuthStore } from '@/store/auth'
import { panelApi } from '@/api'
import { openExternal } from '@/utils/url'
import type { ItemGroup, ItemIcon } from '@/typings'

const panel = usePanelStore()
const auth = useAuthStore()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const config = panel.config

const settingsShow = ref(false)
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

onMounted(async () => {
  try {
    await panel.load()
    if (panel.publicBlocked) {
      router.replace('/login')
    }
  } catch {
    if (!auth.isLoggedIn) router.replace('/login')
  }
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
      const res = await panelApi.saveGroup({ title: value })
      panel.groups.push({ id: res.id, title: value, sort: panel.groups.length })
      message.success('分组已创建')
    } else if (promptGroup.value) {
      await panelApi.saveGroup({ id: promptGroup.value.id, title: value })
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
      <header v-if="config.showHeader" class="hero" :class="heroClass">
        <h1 class="logo" :style="logoStyle">{{ config.logoText }}</h1>
        <Clock v-if="config.showClock" />
      </header>

      <SearchBar v-if="config.showSearch" />

      <div class="groups">
        <GroupSection
          v-for="group in panel.groups"
          :key="group.id"
          :group="group"
          :items="panel.itemsByGroup.get(group.id) || []"
          :edit-mode="panel.editMode"
          @open="onOpenItem"
          @edit="openEditItem"
          @delete="deleteItem"
          @add="openAddItem"
          @rename="openRenameGroup"
          @remove="removeGroup"
          @sort="onSortItems"
        />

        <div v-if="panel.loaded && panel.groups.length === 0" class="empty-state">
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

    <div v-if="panel.loading && !panel.loaded" class="loading">
      <span class="spinner" />
    </div>

    <SettingsModal v-if="auth.isLoggedIn" v-model:show="settingsShow" />

    <EditItemModal
      v-if="auth.isLoggedIn"
      v-model:show="itemModalShow"
      :item="editingItem"
      :groups="panel.groups"
      :default-group-id="defaultGroupId"
      @saved="reload"
    />

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
