<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NSwitch, useDialog, useMessage } from 'naive-ui'
import {
  Activity,
  ArrowDown,
  ArrowUp,
  Check,
  Cloud,
  CloudDownload,
  CloudUpload,
  Code2,
  Copy,
  DatabaseBackup,
  Download,
  Eye,
  EyeOff,
  FileArchive,
  FolderSync,
  Github,
  HardDriveDownload,
  Images,
  Info,
  KeyRound,
  LayoutGrid,
  Link2,
  LogOut,
  Megaphone,
  Minimize2,
  MonitorCog,
  MoveRight,
  Palette,
  Plus,
  RefreshCw,
  Server,
  ShieldCheck,
  Square,
  SquareCheck,
  Trash2,
  TriangleAlert,
  Upload,
  User,
  X,
} from 'lucide-vue-next'
import RoundCardModal from './RoundCardModal.vue'
import PromptModal from './PromptModal.vue'
import ServiceStatus from './ServiceStatus.vue'
import { authApi, backupApi, cloudApi, fileApi, panelApi, settingApi, siteBackupApi, systemApi, userApi } from '@/api'
import { useAuthStore } from '@/store/auth'
import { usePanelStore } from '@/store/panel'
import { fontOptions } from '@/store/defaults'
import { checkForUpdate, type UpdateCheck } from '@/utils/update'
import type {
  AuditFile,
  CloudBackup,
  CloudConfig,
  CloudImage,
  FileRecord,
  User as PanelUser,
} from '@/typings'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', value: boolean): void }>()
defineExpose({
  openSection: (key: string) => {
    active.value = key
  },
})

const panel = usePanelStore()
const auth = useAuthStore()
const message = useMessage()
const dialog = useDialog()
const router = useRouter()

const config = panel.config
const active = ref('profile')

const sections = [
  { key: 'profile', label: '我的信息', icon: User },
  { key: 'style', label: '风格设置', icon: Palette },
  { key: 'groups', label: '分组管理', icon: LayoutGrid },
  { key: 'status', label: '服务状态', icon: Activity },
  { key: 'library', label: '图库管理', icon: Images },
  { key: 'data', label: '导出导入', icon: DatabaseBackup },
  { key: 'account', label: '账号管理', icon: ShieldCheck },
  { key: 'advanced', label: '高级设置', icon: Code2 },
  { key: 'cloud', label: '云存储与漫游', icon: Cloud },
  { key: 'about', label: '关于', icon: Info },
]

const isAdmin = computed(() => auth.user?.role === 1)
const roleLabel = computed(() => (auth.user?.role === 1 ? '管理员' : '普通用户'))
const joinedAt = computed(() => {
  if (!auth.user?.createdAt) return ''
  const d = new Date(auth.user.createdAt)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
})

function touch() {
  panel.scheduleSave()
}

/* ---------- password ---------- */
const pwd = reactive({ old: '', next: '', confirm: '' })
const changing = ref(false)

async function changePassword() {
  if (!pwd.old || !pwd.next) {
    message.warning('请填写完整')
    return
  }
  if (pwd.next.length < 6) {
    message.warning('新密码至少 6 位')
    return
  }
  if (pwd.next !== pwd.confirm) {
    message.warning('两次输入的新密码不一致')
    return
  }
  changing.value = true
  try {
    await authApi.changePassword(pwd.old, pwd.next)
    message.success('密码已更新')
    pwd.old = pwd.next = pwd.confirm = ''
  } catch (e: any) {
    message.error(e.message || '修改失败')
  } finally {
    changing.value = false
  }
}

/* ---------- pages & groups ---------- */
const newGroup = ref('')
const activePageId = ref(0)
const newPageTitle = ref('')

const pageGroups = computed(() =>
  panel.groups
    .filter((group) => group.pageId === activePageId.value)
    .slice()
    .sort((a, b) => a.sort - b.sort || a.id - b.id),
)

const activePage = computed(
  () => panel.pages.find((page) => page.id === activePageId.value) || null,
)

function ensureActivePage() {
  if (panel.pages.some((page) => page.id === activePageId.value)) return
  activePageId.value = panel.currentPageId || panel.pages[0]?.id || 0
}

async function addPage() {
  const title = newPageTitle.value.trim()
  if (!title) return
  try {
    const res = await panelApi.savePage({ title, columns: 1 })
    panel.pages.push({ id: res.id, title, sort: panel.pages.length, columns: 1 })
    activePageId.value = res.id
    panel.setCurrentPage(res.id)
    newPageTitle.value = ''
    message.success('页面已创建')
  } catch (e: any) {
    message.error(e.message || '创建页面失败')
  }
}

async function movePage(id: number, delta: number) {
  const list = panel.pages.slice().sort((a, b) => a.sort - b.sort || a.id - b.id)
  const index = list.findIndex((page) => page.id === id)
  const target = index + delta
  if (index < 0 || target < 0 || target >= list.length) return
  const [page] = list.splice(index, 1)
  list.splice(target, 0, page)
  list.forEach((entry, order) => (entry.sort = order))
  panel.pages = list
  try {
    await panelApi.sortPages(list.map((entry) => entry.id))
  } catch {
    /* order is optimistic */
  }
}

async function savePageMeta() {
  if (!activePage.value) return
  try {
    await panelApi.savePage({
      id: activePage.value.id,
      title: activePage.value.title.trim() || '未命名页面',
      columns: activePage.value.columns,
    })
    message.success('页面设置已保存')
  } catch (e: any) {
    message.error(e.message || '保存页面失败')
  }
}

function removePage() {
  if (!activePage.value) return
  dialog.warning({
    title: '删除页面',
    content: `确定删除「${activePage.value.title}」及页面内所有分组和卡片吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      const pageID = activePage.value?.id
      if (!pageID) return
      try {
        await panelApi.deletePage(pageID)
        panel.pages = panel.pages.filter((page) => page.id !== pageID)
        const removedGroups = panel.groups.filter((group) => group.pageId === pageID)
        const removedGroupIDs = new Set(removedGroups.map((group) => group.id))
        panel.groups = panel.groups.filter((group) => group.pageId !== pageID)
        panel.items = panel.items.filter((item) => !removedGroupIDs.has(item.itemIconGroupId))
        activePageId.value = panel.pages[0]?.id || 0
        panel.setCurrentPage(activePageId.value)
        message.success('页面已删除')
      } catch (e: any) {
        message.error(e.message || '删除页面失败')
      }
    },
  })
}

async function addGroup() {
  const title = newGroup.value.trim()
  if (!title) return
  if (!activePageId.value) {
    message.warning('请先创建或选择一个页面')
    return
  }
  try {
    const res = await panelApi.saveGroup({ title, pageId: activePageId.value })
    const maxSort = Math.max(-1, ...pageGroups.value.map((group) => group.sort)) + 1
    panel.groups.push({ id: res.id, pageId: activePageId.value, title, sort: maxSort })
    newGroup.value = ''
    message.success('分组已创建')
  } catch (e: any) {
    message.error(e.message || '创建失败')
  }
}

async function renameGroup(group: { id: number; pageId: number; title: string }) {
  if (!group.title.trim()) return
  try {
    await panelApi.saveGroup({ id: group.id, pageId: group.pageId, title: group.title.trim() })
  } catch (e: any) {
    message.error(e.message || '重命名失败')
  }
}

function removeGroup(group: { id: number; title: string }) {
  dialog.warning({
    title: '删除分组',
    content: `确定删除「${group.title}」及其中的所有卡片吗？`,
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

async function moveGroup(index: number, delta: number) {
  const list = pageGroups.value.slice()
  const target = index + delta
  if (target < 0 || target >= list.length) return
  const [item] = list.splice(index, 1)
  list.splice(target, 0, item)
  list.forEach((group, i) => (group.sort = i))
  try {
    await panelApi.sortGroups(list.map((group) => group.id))
  } catch {
    /* ignore */
  }
}

/* ---------- library ---------- */
const libraryType = ref<1 | 2>(1)
const libraryScope = ref<'mine' | 'all'>('mine')
const files = ref<FileRecord[]>([])
const libraryLoading = ref(false)
const libraryInput = ref<HTMLInputElement | null>(null)
const selectedFileIds = ref<number[]>([])
const renameShow = ref(false)
const renameTarget = ref<FileRecord | null>(null)
const renameValue = ref('')

const libraryTotalSize = computed(() => files.value.reduce((sum, file) => sum + (file.size || 0), 0))
const isAllSelected = computed(
  () => files.value.length > 0 && files.value.every((file) => selectedFileIds.value.includes(file.id)),
)

function formatSize(value: number): string {
  if (!value) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let index = 0
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024
    index++
  }
  return `${size.toFixed(size >= 100 || index === 0 ? 0 : 1)} ${units[index]}`
}

function syncLabel(status: number): string {
  if (status === 1) return '待审核'
  if (status === 2) return '已上云'
  if (status === 3) return '已驳回'
  return '仅本地'
}

async function loadFiles() {
  if (!auth.isLoggedIn) return
  libraryLoading.value = true
  try {
    files.value = await fileApi.list(
      libraryType.value,
      isAdmin.value && libraryScope.value === 'all' ? 'all' : undefined,
    )
    selectedFileIds.value = selectedFileIds.value.filter((id) => files.value.some((f) => f.id === id))
  } catch {
    files.value = []
  } finally {
    libraryLoading.value = false
  }
}

function toggleFile(id: number) {
  const index = selectedFileIds.value.indexOf(id)
  if (index >= 0) selectedFileIds.value.splice(index, 1)
  else selectedFileIds.value.push(id)
}

function toggleSelectAll() {
  if (isAllSelected.value) {
    selectedFileIds.value = []
  } else {
    selectedFileIds.value = files.value.map((file) => file.id)
  }
}

async function uploadLibrary(e: Event) {
  const input = e.target as HTMLInputElement
  const list = Array.from(input.files || [])
  if (list.length === 0) return
  try {
    for (const file of list) {
      await fileApi.upload(file, libraryType.value)
    }
    message.success(`已上传 ${list.length} 个文件`)
    await loadFiles()
  } catch (err: any) {
    message.error(err.message || '上传失败')
  } finally {
    input.value = ''
  }
}

function applyWallpaper(file: FileRecord) {
  config.backgroundImage = file.src
  touch()
  message.success('已设为背景')
}

async function deleteFileIds(ids: number[]) {
  if (ids.length === 0) return
  try {
    await fileApi.batchRemove(ids)
    const removed = files.value.filter((file) => ids.includes(file.id))
    if (removed.some((file) => config.backgroundImage === file.src)) {
      config.backgroundImage = ''
      touch()
    }
    files.value = files.value.filter((file) => !ids.includes(file.id))
    selectedFileIds.value = selectedFileIds.value.filter((id) => !ids.includes(id))
    message.success(`已删除 ${removed.length} 个文件`)
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

function removeFile(file: FileRecord) {
  dialog.warning({
    title: '删除文件',
    content: `确定删除「${file.fileName}」吗？此操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: () => deleteFileIds([file.id]),
  })
}

function removeSelected() {
  const count = selectedFileIds.value.length
  if (count === 0) return
  dialog.warning({
    title: '批量删除',
    content: `确定删除选中的 ${count} 个文件吗？此操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: () => deleteFileIds([...selectedFileIds.value]),
  })
}

async function moveSelected(target: 1 | 2) {
  if (selectedFileIds.value.length === 0) return
  if (target === libraryType.value) {
    message.info('已经在这个分类里了')
    return
  }
  try {
    await fileApi.move([...selectedFileIds.value], target)
    message.success(`已移动 ${selectedFileIds.value.length} 个文件`)
    selectedFileIds.value = []
    await loadFiles()
  } catch (e: any) {
    message.error(e.message || '移动失败')
  }
}

const compressing = ref(false)

function compressSelected() {
  const ids = [...selectedFileIds.value]
  if (ids.length === 0 || compressing.value) return
  dialog.warning({
    title: '压缩选中图片',
    content: 'JPEG 按质量 82 重新编码，PNG 无损重写；原图会保留在 uploads\\originals\\ 中，确认后开始。',
    positiveText: '开始压缩',
    negativeText: '取消',
    onPositiveClick: async () => {
      compressing.value = true
      try {
        const result = await fileApi.compress(ids)
        message.success(result.message || `已压缩 ${result.compressed} 个文件`)
        selectedFileIds.value = []
        await loadFiles()
      } catch (e: any) {
        message.error(e.message || '压缩失败')
      } finally {
        compressing.value = false
      }
    },
  })
}

function openRename(file: FileRecord) {
  renameTarget.value = file
  renameValue.value = file.fileName
  renameShow.value = true
}

async function confirmRename(value: string) {
  if (!renameTarget.value) return
  try {
    await fileApi.rename(renameTarget.value.id, value)
    renameTarget.value.fileName = value
    message.success('文件名已更新')
  } catch (e: any) {
    message.error(e.message || '重命名失败')
  }
}

async function pushSelectedToCloud() {
  const ids = [...selectedFileIds.value]
  if (ids.length === 0) return
  try {
    if (isAdmin.value) {
      const result = await fileApi.uploadCloud(ids)
      message.success(`已推送 ${result.success} 个文件到云端${result.failed.length ? `，${result.failed.length} 个失败` : ''}`)
    } else {
      const result = await fileApi.applySync(ids)
      message.success(`已提交 ${result.pending} 个文件，等待管理员审核`)
    }
    await loadFiles()
  } catch (e: any) {
    message.error(e.message || '同步失败')
  }
}

/* ---------- backup ---------- */
const importing = ref(false)
const importInput = ref<HTMLInputElement | null>(null)

async function exportBackup() {
  try {
    const data = await backupApi.export()
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    const now = new Date()
    a.href = url
    a.download = `firefly-backup-${now.getFullYear()}${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}.json`
    a.click()
    URL.revokeObjectURL(url)
    message.success('备份已导出')
  } catch (e: any) {
    message.error(e.message || '导出失败')
  }
}

async function onImportFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    importing.value = true
    const text = await file.text()
    const data = JSON.parse(text)
    dialog.warning({
      title: '还原备份',
      content: '当前账号的分组、卡片与风格配置将被备份内容覆盖，确定继续吗？',
      positiveText: '覆盖还原',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          await backupApi.import(data)
          await panel.load()
          message.success('还原完成')
        } catch (err: any) {
          message.error(err.message || '还原失败')
        } finally {
          importing.value = false
        }
      },
    })
  } catch {
    message.error('文件不是有效的 JSON 备份')
    importing.value = false
  } finally {
    input.value = ''
  }
}

/* ---------- full-site backup / bookmarks ---------- */
const siteBackupPassword = ref('')
const siteBackupConfirm = ref('')
const siteBackupExporting = ref(false)
const siteBackupFile = ref<File | null>(null)
const siteBackupImportPassword = ref('')
const siteBackupImporting = ref(false)
const siteBackupInput = ref<HTMLInputElement | null>(null)

const bookmarkFile = ref<File | null>(null)
const bookmarkInput = ref<HTMLInputElement | null>(null)
const bookmarkMode = ref<'folders' | 'single'>('folders')
const bookmarkGroupId = ref<number>(0)
const bookmarkReuseGroups = ref(true)
const bookmarkUpdateExisting = ref(true)
const bookmarkImporting = ref(false)

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

async function exportSiteBackup() {
  if (siteBackupPassword.value && siteBackupPassword.value.length < 6) {
    message.warning('加密密码至少 6 位，或留空导出未加密 ZIP')
    return
  }
  if (siteBackupPassword.value !== siteBackupConfirm.value) {
    message.warning('两次输入的密码不一致')
    return
  }
  siteBackupExporting.value = true
  try {
    const blob = await siteBackupApi.export(siteBackupPassword.value)
    if (blob.type.includes('application/json')) {
      const text = await blob.text()
      let msg = '导出失败'
      try {
        msg = JSON.parse(text).msg || msg
      } catch {
        /* ignore */
      }
      throw new Error(msg)
    }
    const stamp = new Date().toISOString().slice(0, 19).replace(/\D/g, '')
    downloadBlob(blob, `firefly-atlas-full-${stamp}${siteBackupPassword.value ? '.ffbak' : '.zip'}`)
    message.success(siteBackupPassword.value ? '加密整站备份已导出' : '整站 ZIP 备份已导出')
  } catch (e: any) {
    message.error(e.message || '导出失败')
  } finally {
    siteBackupExporting.value = false
  }
}

function onSiteBackupFile(e: Event) {
  const input = e.target as HTMLInputElement
  siteBackupFile.value = input.files?.[0] || null
  input.value = ''
}

function importSiteBackup() {
  if (!siteBackupFile.value) {
    message.warning('请先选择备份文件')
    return
  }
  dialog.warning({
    title: '整站还原',
    content: '将覆盖现有用户、分组、卡片、图库记录与上传文件，且会替换系统设置。确定继续吗？',
    positiveText: '确认还原',
    negativeText: '取消',
    onPositiveClick: async () => {
      siteBackupImporting.value = true
      try {
        const stats = await siteBackupApi.import(siteBackupFile.value as File, siteBackupImportPassword.value)
        message.success(
          `还原完成：${stats.users} 个用户、${stats.groups} 个分组、${stats.items} 张卡片、${stats.files} 个文件记录`,
        )
        window.setTimeout(() => window.location.reload(), 900)
      } catch (e: any) {
        message.error(e.message || '还原失败')
      } finally {
        siteBackupImporting.value = false
      }
    },
  })
}

function onBookmarkFile(e: Event) {
  const input = e.target as HTMLInputElement
  bookmarkFile.value = input.files?.[0] || null
  input.value = ''
}

async function importBookmarks() {
  if (!bookmarkFile.value) {
    message.warning('请先选择浏览器导出的书签 HTML')
    return
  }
  bookmarkImporting.value = true
  try {
    const form = new FormData()
    form.append('file', bookmarkFile.value)
    form.append('mode', bookmarkMode.value)
    form.append('groupId', String(bookmarkGroupId.value))
    form.append('pageId', String(panel.currentPageId))
    form.append('reuseGroups', bookmarkReuseGroups.value ? '1' : '0')
    form.append('updateExisting', bookmarkUpdateExisting.value ? '1' : '0')
    const result = await panelApi.importBookmarks(form)
    const parts = [`新增 ${result.groups} 个分组、${result.items} 张卡片`]
    if (result.reusedGroups) parts.push(`复用 ${result.reusedGroups} 个已有分组`)
    if (result.updated) parts.push(`更新 ${result.updated} 个标题`)
    if (result.skipped) parts.push(`跳过 ${result.skipped} 个重复项`)
    message.success(`导入完成：${parts.join('，')}`)
    bookmarkFile.value = null
    await panel.load()
  } catch (e: any) {
    message.error(e.message || '书签导入失败')
  } finally {
    bookmarkImporting.value = false
  }
}


const publicMode = ref(panel.settings.public_visit_mode === '1')
const siteTitle = ref(panel.settings.site_title || 'Firefly Atlas')

/* ---------- advanced ---------- */
const customCss = ref('')
const customJs = ref('')
const savingCode = ref(false)
const noticeEnabled = ref(false)
const noticeTitle = ref('')
const noticeContent = ref('')
const noticeScope = ref('home')
const savingNotice = ref(false)

function syncSiteSettings() {
  customCss.value = panel.settings.custom_css || ''
  customJs.value = panel.settings.custom_js || ''
  noticeEnabled.value = panel.settings.notice_enabled === '1'
  noticeTitle.value = panel.settings.notice_title || ''
  noticeContent.value = panel.settings.notice_content || ''
  noticeScope.value = panel.settings.notice_scope || 'home'
}

async function saveCustomCode() {
  savingCode.value = true
  try {
    const items = [
      { configName: 'custom_css', content: customCss.value },
      { configName: 'custom_js', content: customJs.value },
    ]
    await settingApi.save(items)
    panel.settings.custom_css = customCss.value
    panel.settings.custom_js = customJs.value
    message.success('自定义 CSS 与 JS 已保存')
  } catch (e: any) {
    message.error(e.message || '保存失败')
  } finally {
    savingCode.value = false
  }
}

async function saveNotice() {
  savingNotice.value = true
  try {
    const items = [
      { configName: 'notice_enabled', content: noticeEnabled.value ? '1' : '0' },
      { configName: 'notice_title', content: noticeTitle.value },
      { configName: 'notice_content', content: noticeContent.value },
      { configName: 'notice_scope', content: noticeScope.value },
    ]
    await settingApi.save(items)
    panel.settings.notice_enabled = noticeEnabled.value ? '1' : '0'
    panel.settings.notice_title = noticeTitle.value
    panel.settings.notice_content = noticeContent.value
    panel.settings.notice_scope = noticeScope.value
    message.success(noticeEnabled.value ? '公告已保存并启用' : '公告已保存并关闭')
  } catch (e: any) {
    message.error(e.message || '保存失败')
  } finally {
    savingNotice.value = false
  }
}

async function copyFileLink(file: FileRecord) {
  const link = `${window.location.origin}${file.src}`
  try {
    await navigator.clipboard.writeText(link)
    message.success('图片链接已复制')
  } catch {
    const input = document.createElement('textarea')
    input.value = link
    document.body.appendChild(input)
    input.select()
    try {
      document.execCommand('copy')
      message.success('图片链接已复制')
    } catch {
      message.error('复制失败，请手动复制')
    }
    input.remove()
  }
}

/* ---------- search engines ---------- */
const newEngine = reactive({ name: '', url: '' })

function addEngine() {
  const name = newEngine.name.trim()
  const url = newEngine.url.trim()
  if (!name || !url) {
    message.warning('请填写引擎名称和搜索地址')
    return
  }
  panel.searchEngines.push({ name, url })
  newEngine.name = ''
  newEngine.url = ''
  panel.scheduleSave()
  message.success('搜索引擎已添加')
}

function removeEngine(index: number) {
  const [removed] = panel.searchEngines.splice(index, 1)
  if (removed) panel.scheduleSave()
}

watch(
  () => props.show,
  (visible) => {
    if (!visible) return
    publicMode.value = panel.settings.public_visit_mode === '1'
    siteTitle.value = panel.settings.site_title || 'Firefly Atlas'
    syncSiteSettings()
    if (active.value === 'library') void loadFiles()
    if (active.value === 'account') void loadUsers()
    if (active.value === 'cloud') void refreshCloud()
    if (active.value === 'groups') ensureActivePage()
    if (active.value === 'about') void loadVersion()
  },
)

watch(active, (value) => {
  if (value === 'library') void loadFiles()
  if (value === 'account') void loadUsers()
  if (value === 'cloud') void refreshCloud()
  if (value === 'groups') ensureActivePage()
  if (value === 'about') void loadVersion()
})

watch(
  () => panel.pages,
  () => {
    if (props.show && active.value === 'groups') ensureActivePage()
  },
  { deep: true },
)

watch(activePageId, (value) => {
  if (value) panel.setCurrentPage(value)
})

watch(libraryType, () => {
  if (props.show) void loadFiles()
})

watch(libraryScope, () => {
  if (props.show) void loadFiles()
})

async function savePublicMode(value: boolean) {
  try {
    await settingApi.save([{ configName: 'public_visit_mode', content: value ? '1' : '0' }])
    panel.settings.public_visit_mode = value ? '1' : '0'
    message.success(value ? '公开访客模式已开启' : '公开访客模式已关闭')
  } catch (e: any) {
    publicMode.value = !value
    message.error(e.message || '保存失败')
  }
}

async function saveSiteTitle() {
  try {
    await settingApi.save([{ configName: 'site_title', content: siteTitle.value }])
    panel.settings.site_title = siteTitle.value
    message.success('站点标题已保存')
  } catch (e: any) {
    message.error(e.message || '保存失败')
  }
}

/* ---------- users ---------- */
const users = ref<PanelUser[]>([])
const usersLoading = ref(false)
const creatingUser = ref(false)
const newUser = reactive({ username: '', nickname: '', password: '', role: 2 })

async function loadUsers() {
  if (!isAdmin.value) return
  usersLoading.value = true
  try {
    users.value = await userApi.list()
  } catch {
    users.value = []
  } finally {
    usersLoading.value = false
  }
}

async function createUser() {
  if (!newUser.username.trim() || newUser.password.length < 6) {
    message.warning('请填写用户名，密码至少 6 位')
    return
  }
  creatingUser.value = true
  try {
    await userApi.save({
      username: newUser.username.trim(),
      nickname: newUser.nickname.trim(),
      password: newUser.password,
      role: newUser.role,
      status: 1,
    })
    message.success('用户已创建')
    newUser.username = ''
    newUser.nickname = ''
    newUser.password = ''
    newUser.role = 2
    await loadUsers()
  } catch (e: any) {
    message.error(e.message || '创建失败')
  } finally {
    creatingUser.value = false
  }
}

async function toggleUser(user: PanelUser) {
  const prev = user.status
  try {
    await userApi.save({
      id: user.id,
      username: user.username,
      nickname: user.nickname,
      role: user.role,
      status: prev === 1 ? 0 : 1,
    })
    user.status = prev === 1 ? 0 : 1
    message.success(user.status === 1 ? '账号已启用' : '账号已禁用')
  } catch (e: any) {
    message.error(e.message || '操作失败')
  }
}

async function changeUserRole(user: PanelUser, role: number) {
  const prev = user.role
  try {
    await userApi.save({
      id: user.id,
      username: user.username,
      nickname: user.nickname,
      role,
      status: user.status,
    })
    user.role = role
    message.success('角色已更新')
  } catch (e: any) {
    user.role = prev
    message.error(e.message || '操作失败')
  }
}

function removeUser(user: PanelUser) {
  dialog.warning({
    title: '删除用户',
    content: `确定删除「${user.nickname || user.username}」及其全部面板数据吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await userApi.remove(user.id)
        users.value = users.value.filter((u) => u.id !== user.id)
        message.success('用户已删除')
      } catch (e: any) {
        message.error(e.message || '删除失败')
      }
    },
  })
}

/* ---------- cloud storage ---------- */
type CloudTab = 'images' | 'audit' | 'config' | 'backups'

function emptyCloudConfig(): CloudConfig {
  return {
    enabled: false,
    provider: 'github',
    dualWrite: false,
    github: { owner: '', repo: '', branch: 'main', token: '', basePath: 'firefly-atlas', useCdn: false },
    s3: {
      endpoint: '',
      accountId: '',
      region: 'auto',
      accessKeyId: '',
      secretAccessKey: '',
      bucket: '',
      publicBaseUrl: '',
      insecure: false,
    },
  }
}

const cloudActiveTab = ref<CloudTab>('images')
const cloudConfig = reactive<CloudConfig>(emptyCloudConfig())
const cloudConfigLoading = ref(false)
const cloudSaving = ref(false)
const cloudTesting = ref(false)
const cloudTestResult = ref('')
const cloudImages = ref<CloudImage[]>([])
const cloudImagesLoading = ref(false)
const cloudImageCategory = ref<'all' | 'wallpapers' | 'icons'>('all')
const cloudSelectedKeys = ref<string[]>([])
const cloudAuditList = ref<AuditFile[]>([])
const cloudAuditLoading = ref(false)
const cloudSelectedAuditIds = ref<number[]>([])
const cloudBackups = ref<CloudBackup[]>([])
const cloudBackupsLoading = ref(false)
const cloudBackupPassword = ref('')
const cloudBackupConfirm = ref('')
const cloudUploadingBackup = ref(false)
const cloudRestoreShow = ref(false)
const cloudRestoreKey = ref('')
const cloudRestorePassword = ref('')
const cloudRestoring = ref(false)
const cloudBusy = ref(false)

const filteredCloudImages = computed(() =>
  cloudImageCategory.value === 'all'
    ? cloudImages.value
    : cloudImages.value.filter((img) =>
        cloudImageCategory.value === 'wallpapers' ? img.category === 1 : img.category === 2,
      ),
)
const cloudTotalSize = computed(() => cloudImages.value.reduce((sum, img) => sum + (img.size || 0), 0))
const isAllCloudSelected = computed(
  () =>
    filteredCloudImages.value.length > 0 &&
    filteredCloudImages.value.every((img) => cloudSelectedKeys.value.includes(img.key)),
)

function applyCloudConfig(cfg: CloudConfig) {
  const base = emptyCloudConfig()
  Object.assign(cloudConfig, base, cfg)
  cloudConfig.github = { ...base.github, ...(cfg.github || {}) }
  cloudConfig.s3 = { ...base.s3, ...(cfg.s3 || {}) }
}

async function loadCloudConfig() {
  if (!isAdmin.value) return
  cloudConfigLoading.value = true
  try {
    applyCloudConfig(await cloudApi.getConfig())
  } catch (e: any) {
    message.error(e.message || '读取云存储配置失败')
  } finally {
    cloudConfigLoading.value = false
  }
}

async function saveCloudConfig() {
  cloudSaving.value = true
  try {
    await cloudApi.saveConfig(JSON.parse(JSON.stringify(cloudConfig)))
    message.success('云存储配置已保存')
  } catch (e: any) {
    message.error(e.message || '保存失败')
  } finally {
    cloudSaving.value = false
  }
}

async function testCloudConnection() {
  cloudTesting.value = true
  cloudTestResult.value = ''
  try {
    const result = await cloudApi.test(JSON.parse(JSON.stringify(cloudConfig)))
    cloudTestResult.value = `${result.description}`
    message.success('连接测试通过')
  } catch (e: any) {
    cloudTestResult.value = e.message || '连接失败'
    message.error(e.message || '连接失败')
  } finally {
    cloudTesting.value = false
  }
}

async function loadCloudImages() {
  if (!isAdmin.value) return
  if (!cloudConfig.enabled) {
    cloudImages.value = []
    cloudSelectedKeys.value = []
    return
  }
  cloudImagesLoading.value = true
  try {
    cloudImages.value = await cloudApi.listImages()
    cloudSelectedKeys.value = cloudSelectedKeys.value.filter((key) =>
      cloudImages.value.some((img) => img.key === key),
    )
  } catch (e: any) {
    message.error(e.message || '读取云端图库失败')
  } finally {
    cloudImagesLoading.value = false
  }
}

async function loadCloudAudit() {
  if (!isAdmin.value) return
  cloudAuditLoading.value = true
  try {
    cloudAuditList.value = await fileApi.auditList()
    cloudSelectedAuditIds.value = cloudSelectedAuditIds.value.filter((id) =>
      cloudAuditList.value.some((file) => file.id === id),
    )
  } catch (e: any) {
    message.error(e.message || '读取待审列表失败')
  } finally {
    cloudAuditLoading.value = false
  }
}

async function loadCloudBackups() {
  if (!isAdmin.value) return
  if (!cloudConfig.enabled) {
    cloudBackups.value = []
    return
  }
  cloudBackupsLoading.value = true
  try {
    cloudBackups.value = await cloudApi.listBackups()
  } catch (e: any) {
    message.error(e.message || '读取云端备份失败')
  } finally {
    cloudBackupsLoading.value = false
  }
}

async function refreshCloud() {
  if (!isAdmin.value) return
  await loadCloudConfig()
  await loadCloudAudit()
  if (cloudActiveTab.value === 'images') await loadCloudImages()
  else if (cloudActiveTab.value === 'backups') await loadCloudBackups()
}

function toggleCloudImage(key: string) {
  const index = cloudSelectedKeys.value.indexOf(key)
  if (index >= 0) cloudSelectedKeys.value.splice(index, 1)
  else cloudSelectedKeys.value.push(key)
}

function toggleAllCloudImages() {
  if (isAllCloudSelected.value) {
    cloudSelectedKeys.value = []
  } else {
    cloudSelectedKeys.value = filteredCloudImages.value.map((img) => img.key)
  }
}

async function syncSelectedCloud() {
  if (cloudSelectedKeys.value.length === 0) return
  cloudBusy.value = true
  try {
    const result = await cloudApi.sync([...cloudSelectedKeys.value])
    message.success(`已拉取 ${result.success} 张到本地${result.failed.length ? `，${result.failed.length} 张失败` : ''}`)
    await loadCloudImages()
    await loadFiles()
  } catch (e: any) {
    message.error(e.message || '拉取失败')
  } finally {
    cloudBusy.value = false
  }
}

async function syncAllCloud() {
  cloudBusy.value = true
  try {
    const result = await cloudApi.syncAll()
    message.success(`已拉取 ${result.success}/${result.total || result.success} 张到本地`)
    await loadCloudImages()
    await loadFiles()
  } catch (e: any) {
    message.error(e.message || '拉取失败')
  } finally {
    cloudBusy.value = false
  }
}

function deleteSelectedCloud() {
  if (cloudSelectedKeys.value.length === 0) return
  const count = cloudSelectedKeys.value.length
  dialog.warning({
    title: '删除云端图片',
    content: `确定从云端删除选中的 ${count} 张图片吗？本地文件不会被删除。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      cloudBusy.value = true
      try {
        const result = await cloudApi.deleteImages([...cloudSelectedKeys.value])
        message.success(`已删除 ${result.success} 张${result.failed.length ? `，${result.failed.length} 张失败` : ''}`)
        cloudSelectedKeys.value = []
        await loadCloudImages()
      } catch (e: any) {
        message.error(e.message || '删除失败')
      } finally {
        cloudBusy.value = false
      }
    },
  })
}

function clearCloudImages() {
  dialog.warning({
    title: '清空云端图床',
    content: `确定删除云端全部 ${cloudImages.value.length} 张图片吗？此操作不可恢复。`,
    positiveText: '清空',
    negativeText: '取消',
    onPositiveClick: async () => {
      cloudBusy.value = true
      try {
        const result = await cloudApi.clearImages()
        message.success(`已清空 ${result.success} 张云端图片`)
        cloudSelectedKeys.value = []
        await loadCloudImages()
      } catch (e: any) {
        message.error(e.message || '清空失败')
      } finally {
        cloudBusy.value = false
      }
    },
  })
}

function toggleAudit(id: number) {
  const index = cloudSelectedAuditIds.value.indexOf(id)
  if (index >= 0) cloudSelectedAuditIds.value.splice(index, 1)
  else cloudSelectedAuditIds.value.push(id)
}

function toggleAllAudit() {
  if (cloudSelectedAuditIds.value.length === cloudAuditList.value.length) {
    cloudSelectedAuditIds.value = []
  } else {
    cloudSelectedAuditIds.value = cloudAuditList.value.map((file) => file.id)
  }
}

async function auditSelected(action: 'approve' | 'reject') {
  const ids = [...cloudSelectedAuditIds.value]
  if (ids.length === 0) return
  cloudBusy.value = true
  try {
    const result = await fileApi.audit(ids, action)
    if (action === 'approve') {
      message.success(`已批准 ${result.success || 0} 个文件${result.failed?.length ? `，${result.failed.length} 个失败` : ''}`)
    } else {
      message.success(`已驳回 ${result.rejected || 0} 个文件`)
    }
    cloudSelectedAuditIds.value = []
    await loadCloudAudit()
    await loadFiles()
  } catch (e: any) {
    message.error(e.message || '审核失败')
  } finally {
    cloudBusy.value = false
  }
}

async function uploadCloudBackup() {
  if (cloudBackupPassword.value.length < 6) {
    message.warning('备份密码至少 6 位')
    return
  }
  if (cloudBackupPassword.value !== cloudBackupConfirm.value) {
    message.warning('两次输入的密码不一致')
    return
  }
  cloudUploadingBackup.value = true
  try {
    const result = await cloudApi.uploadBackup(cloudBackupPassword.value)
    message.success(`云端加密备份已上传：${result.name}`)
    cloudBackupPassword.value = ''
    cloudBackupConfirm.value = ''
    await loadCloudBackups()
  } catch (e: any) {
    message.error(e.message || '上传备份失败')
  } finally {
    cloudUploadingBackup.value = false
  }
}

function openCloudRestore(backup: CloudBackup) {
  cloudRestoreKey.value = backup.key
  cloudRestorePassword.value = ''
  cloudRestoreShow.value = true
}

async function restoreCloudBackup() {
  if (!cloudRestoreKey.value || !cloudRestorePassword.value) {
    message.warning('请填写解密密码')
    return
  }
  cloudRestoring.value = true
  try {
    const stats = await cloudApi.restoreBackup(cloudRestoreKey.value, cloudRestorePassword.value)
    message.success(
      `云端还原完成：${stats.users} 个用户、${stats.groups} 个分组、${stats.items} 张卡片`,
    )
    window.setTimeout(() => window.location.reload(), 900)
  } catch (e: any) {
    message.error(e.message || '还原失败')
  } finally {
    cloudRestoring.value = false
  }
}

async function downloadCloudBackup(backup: CloudBackup) {
  try {
    const blob = await cloudApi.downloadBackup(backup.key)
    if (blob.type.includes('application/json')) {
      const text = await blob.text()
      let msg = '下载失败'
      try {
        msg = JSON.parse(text).msg || msg
      } catch {
        /* ignore */
      }
      throw new Error(msg)
    }
    downloadBlob(blob, backup.name || 'firefly-backup.ffbak')
  } catch (e: any) {
    message.error(e.message || '下载失败')
  }
}

function deleteCloudBackup(backup: CloudBackup) {
  dialog.warning({
    title: '删除云端备份',
    content: `确定删除「${backup.name}」吗？删除后无法通过该备份恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await cloudApi.deleteBackups([backup.key])
        message.success('备份已删除')
        await loadCloudBackups()
      } catch (e: any) {
        message.error(e.message || '删除失败')
      }
    },
  })
}

function formatCloudDate(value: string): string {
  if (!value) return '—'
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? '—' : d.toLocaleString()
}

watch(cloudActiveTab, (tab) => {
  if (!props.show || !isAdmin.value) return
  if (tab === 'images') void loadCloudImages()
  else if (tab === 'audit') void loadCloudAudit()
  else if (tab === 'config') void loadCloudConfig()
  else void loadCloudBackups()
})


async function logout() {
  await auth.logout()
  emit('update:show', false)
  router.push('/login')
}

const version = ref('1.3.2')
const updateChecking = ref(false)
const updateState = ref<UpdateCheck | null>(null)
const updateError = ref('')

async function loadVersion() {
  try {
    const info = await systemApi.version()
    if (info?.version) version.value = info.version
  } catch {
    /* keep the bundled fallback */
  }
}

async function checkUpdates() {
  updateChecking.value = true
  updateError.value = ''
  try {
    const result = await checkForUpdate(version.value)
    if (!result) {
      updateError.value = '暂时无法连接 GitHub，请稍后再试'
      return
    }
    updateState.value = result
    if (!result.hasUpdate) message.success('当前已是最新版本')
  } finally {
    updateChecking.value = false
  }
}
</script>

<template>
  <RoundCardModal
    :show="show"
    :width="940"
    :height="650"
    :min-width="360"
    :min-height="420"
    title="系统应用"
    @update:show="emit('update:show', $event)"
  >
    <div class="settings-shell">
      <aside class="settings-nav">
        <div class="nav-brand">
          <span class="brand-dot"><span /></span>
          <div>
            <strong>Firefly Atlas</strong>
            <small>系统应用</small>
          </div>
        </div>
        <button
          v-for="section in sections"
          :key="section.key"
          class="nav-item"
          :class="{ active: active === section.key }"
          type="button"
          @click="active = section.key"
        >
          <component :is="section.icon" :size="15" />
          <span>{{ section.label }}</span>
        </button>
      </aside>

      <div class="settings-content">
        <!-- 我的信息 -->
        <section v-if="active === 'profile'" class="pane">
          <header class="pane-head">
            <h2>我的信息</h2>
            <p>账号概况与登录密码。</p>
          </header>

          <div class="profile-card">
            <div class="avatar">{{ (auth.user?.nickname || auth.user?.username || 'F').slice(0, 1) }}</div>
            <div class="profile-meta">
              <strong>{{ auth.user?.nickname || '未登录' }}</strong>
              <span>@{{ auth.user?.username }}</span>
              <em>{{ roleLabel }}</em>
            </div>
            <div v-if="joinedAt" class="joined">加入于 {{ joinedAt }}</div>
          </div>

          <div class="block">
            <h3>修改密码</h3>
            <div class="pwd-grid">
              <div>
                <label class="label">原密码</label>
                <input v-model="pwd.old" class="field" type="password" autocomplete="current-password" />
              </div>
              <div>
                <label class="label">新密码</label>
                <input v-model="pwd.next" class="field" type="password" autocomplete="new-password" />
              </div>
              <div>
                <label class="label">确认新密码</label>
                <input v-model="pwd.confirm" class="field" type="password" autocomplete="new-password" />
              </div>
            </div>
            <div class="block-foot">
              <button class="btn primary" type="button" :disabled="changing" @click="changePassword">
                {{ changing ? '提交中...' : '更新密码' }}
              </button>
            </div>
          </div>
        </section>

        <!-- 风格设置 -->
        <section v-else-if="active === 'style'" class="pane">
          <header class="pane-head">
            <h2>风格设置</h2>
            <p>所有调整实时生效并自动保存。</p>
          </header>

          <div class="block">
            <h3>主题与主色</h3>
            <div class="row">
              <div><div class="row-label">主题模式</div><div class="row-hint">浅色、深色或跟随系统</div></div>
              <div class="seg">
                <button :class="{ active: config.themeMode === 'light' }" @click="config.themeMode = 'light'; touch()">浅色</button>
                <button :class="{ active: config.themeMode === 'dark' }" @click="config.themeMode = 'dark'; touch()">深色</button>
                <button :class="{ active: config.themeMode === 'auto' }" @click="config.themeMode = 'auto'; touch()">跟随</button>
              </div>
            </div>
            <div class="row">
              <div><div class="row-label">主色</div><div class="row-hint">按钮、高亮与状态点</div></div>
              <div class="color-set">
                <input v-model="config.accentColor" type="color" class="color-input" @input="touch" />
                <span class="mono">{{ config.accentColor }}</span>
              </div>
            </div>
          </div>

          <div class="block">
            <h3>标题与时钟</h3>
            <div class="row">
              <div><div class="row-label">标题文字</div></div>
              <input v-model="config.logoText" class="field w-260" maxlength="40" @input="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">标题字体</div></div>
              <select v-model="config.logoFontFamily" class="field w-260" @change="touch">
                <option v-for="font in fontOptions" :key="font.label" :value="font.value">{{ font.label }}</option>
              </select>
            </div>
            <div class="row">
              <div><div class="row-label">标题颜色</div><div class="row-hint">留空时自动跟随主题</div></div>
              <div class="color-set">
                <input
                  :value="config.logoColor || '#172033'"
                  type="color"
                  class="color-input"
                  @input="config.logoColor = ($event.target as HTMLInputElement).value; touch()"
                />
                <button class="mini" type="button" @click="config.logoColor = ''; touch()">自动</button>
              </div>
            </div>
            <div class="row">
              <div><div class="row-label">标题毛玻璃底</div></div>
              <n-switch v-model:value="config.logoGlassEffect" size="small" @update:value="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">头部透明度</div><div class="row-hint">{{ config.headerOpacity }}%</div></div>
              <input v-model.number="config.headerOpacity" class="range" type="range" min="20" max="100" @input="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">显示时钟</div></div>
              <n-switch v-model:value="config.showClock" size="small" @update:value="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">显示日期</div></div>
              <n-switch v-model:value="config.clockShowDate" size="small" @update:value="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">显示秒数</div></div>
              <n-switch v-model:value="config.clockShowSeconds" size="small" @update:value="touch" />
            </div>
          </div>

          <div class="block">
            <h3>卡片与布局</h3>
            <div class="row">
              <div><div class="row-label">卡片毛玻璃透明度</div><div class="row-hint">0% 为完全通透，仅保留图标与文字</div></div>
              <div class="range-wrap">
                <input v-model.number="config.cardGlassOpacity" class="range" type="range" min="0" max="100" @input="touch" />
                <span class="range-val">{{ config.cardGlassOpacity }}%</span>
              </div>
            </div>
            <div class="row">
              <div><div class="row-label">图标风格</div></div>
              <div class="seg">
                <button :class="{ active: Number(config.iconStyle) === 0 }" @click="config.iconStyle = 0; touch()">详情卡片</button>
                <button :class="{ active: Number(config.iconStyle) === 1 }" @click="config.iconStyle = 1; touch()">大图标</button>
                <button :class="{ active: Number(config.iconStyle) === 2 }" @click="config.iconStyle = 2; touch()">小图标</button>
              </div>
            </div>
            <div class="row">
              <div><div class="row-label">页面最大宽度</div></div>
              <div class="inline-set">
                <input v-model="config.maxWidth" class="field w-90" @input="touch" />
                <select v-model="config.maxWidthUnit" class="field w-80" @change="touch">
                  <option value="px">px</option>
                  <option value="%">%</option>
                </select>
              </div>
            </div>
            <div class="row">
              <div><div class="row-label">左右边距</div><div class="row-hint">{{ config.marginX }} px</div></div>
              <input v-model.number="config.marginX" class="range" type="range" min="0" max="120" @input="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">上边距</div><div class="row-hint">{{ config.marginTop }} vh</div></div>
              <input v-model.number="config.marginTop" class="range" type="range" min="0" max="30" @input="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">下边距</div><div class="row-hint">{{ config.marginBottom }} vh</div></div>
              <input v-model.number="config.marginBottom" class="range" type="range" min="0" max="20" @input="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">卡片间隙</div><div class="row-hint">{{ config.itemGap }} px</div></div>
              <input v-model.number="config.itemGap" class="range" type="range" min="4" max="40" @input="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">卡片圆角</div><div class="row-hint">{{ config.cardRadius }} px</div></div>
              <input v-model.number="config.cardRadius" class="range" type="range" min="0" max="32" @input="touch" />
            </div>
          </div>

          <div class="block">
            <h3>卡片信息与搜索</h3>
            <div class="row">
              <div>
                <div class="row-label">隐藏卡片标题</div>
                <div class="row-hint">适合大图标 / 小图标风格，鼠标悬浮时显示名称</div>
              </div>
              <n-switch v-model:value="config.hideTitle" size="small" @update:value="touch" />
            </div>
            <div class="row">
              <div>
                <div class="row-label">隐藏卡片描述</div>
                <div class="row-hint">仅对详情卡片风格生效</div>
              </div>
              <n-switch v-model:value="config.hideDescription" size="small" @update:value="touch" />
            </div>
            <div class="row">
              <div>
                <div class="row-label">搜索时过滤面板项目</div>
                <div class="row-hint">输入关键词时，在搜索栏下方列出匹配的卡片</div>
              </div>
              <n-switch v-model:value="config.searchFilterItems" size="small" @update:value="touch" />
            </div>
          </div>

          <div class="block">
            <h3><MonitorCog :size="13" class="inline-ico" /> 系统状态</h3>
            <p class="block-desc">展示宿主机的 CPU、内存、磁盘与网络概况，每 5 秒自动刷新。</p>
            <div class="row">
              <div><div class="row-label">显示系统状态</div></div>
              <n-switch v-model:value="config.showSystemMonitor" size="small" @update:value="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">显示状态标题</div></div>
              <n-switch v-model:value="config.systemMonitorShowTitle" size="small" @update:value="touch" />
            </div>
            <div v-if="isAdmin" class="row">
              <div>
                <div class="row-label">允许访客查看</div>
                <div class="row-hint">需同时开启公开访客模式，并使用管理员的面板配置</div>
              </div>
              <n-switch v-model:value="config.systemMonitorPublic" size="small" @update:value="touch" />
            </div>
          </div>

          <div class="block">
            <h3>搜索</h3>
            <div class="row">
              <div><div class="row-label">显示搜索栏</div></div>
              <n-switch v-model:value="config.showSearch" size="small" @update:value="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">占位文字</div></div>
              <input v-model="config.searchPlaceholder" class="field w-260" @input="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">在新标签页打开</div></div>
              <n-switch v-model:value="config.searchOpenInNewTab" size="small" @update:value="touch" />
            </div>
          </div>

          <div class="block">
            <h3>搜索引擎</h3>
            <p class="block-desc">搜索地址以查询参数结尾，关键词会自动追加到末尾。</p>
            <div class="engine-list">
              <div v-for="(engine, index) in panel.searchEngines" :key="index" class="engine-row">
                <input v-model="engine.name" class="field engine-name" maxlength="20" @input="touch" />
                <input v-model="engine.url" class="field engine-url" @input="touch" />
                <button class="mini icon danger" type="button" title="删除" @click="removeEngine(index)">
                  <Trash2 :size="12" />
                </button>
              </div>
            </div>
            <div class="engine-add">
              <input v-model="newEngine.name" class="field engine-name" placeholder="名称" maxlength="20" />
              <input
                v-model="newEngine.url"
                class="field engine-url"
                placeholder="https://example.com/search?q="
                @keyup.enter="addEngine"
              />
              <button class="mini" type="button" @click="addEngine">
                <Plus :size="12" /> 添加
              </button>
            </div>
          </div>

          <div class="block">
            <h3>页脚</h3>
            <div class="row">
              <div><div class="row-label">显示页脚</div></div>
              <n-switch v-model:value="config.showFooter" size="small" @update:value="touch" />
            </div>
            <div class="row vertical">
              <div><div class="row-label">页脚 HTML</div><div class="row-hint">支持链接与简单标签</div></div>
              <textarea v-model="config.footerHtml" class="field mono" rows="3" @input="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">独立垂直偏移</div><div class="row-hint">{{ config.footerOffsetY }} px</div></div>
              <input v-model.number="config.footerOffsetY" class="range" type="range" min="-120" max="150" @input="touch" />
            </div>
          </div>

          <div class="block">
            <h3>背景</h3>
            <div class="row">
              <div><div class="row-label">背景图片</div><div class="row-hint">留空使用内置极光渐变</div></div>
              <div class="inline-set">
                <input v-model="config.backgroundImage" class="field w-260" placeholder="/uploads/wallpapers/..." @input="touch" />
                <button class="mini" type="button" @click="config.backgroundImage = ''; touch()">清除</button>
              </div>
            </div>
            <div class="row">
              <div><div class="row-label">背景模糊</div><div class="row-hint">{{ config.backgroundBlur }} px</div></div>
              <input v-model.number="config.backgroundBlur" class="range" type="range" min="0" max="24" @input="touch" />
            </div>
            <div class="row">
              <div><div class="row-label">遮罩浓度</div><div class="row-hint">{{ config.backgroundMask }}%</div></div>
              <input v-model.number="config.backgroundMask" class="range" type="range" min="0" max="90" @input="touch" />
            </div>
          </div>
        </section>

        <!-- 分组管理 -->
        <section v-else-if="active === 'groups'" class="pane">
          <header class="pane-head">
            <h2>页面与分组</h2>
            <p>每个页面拥有自己的分组和卡片；分页只改变展示位置，不会移动数据到别的账号。</p>
          </header>

          <div class="block">
            <h3>页面</h3>
            <div class="row">
              <div><div class="row-label">当前页面</div></div>
              <div class="inline-set">
                <select v-model.number="activePageId" class="field w-220">
                  <option v-for="page in panel.pages" :key="page.id" :value="page.id">{{ page.title }}</option>
                </select>
                <button
                  class="mini icon"
                  type="button"
                  title="页面前移"
                  :disabled="!activePage || panel.pages[0]?.id === activePageId"
                  @click="movePage(activePageId, -1)"
                >
                  <ArrowUp :size="13" />
                </button>
                <button
                  class="mini icon"
                  type="button"
                  title="页面后移"
                  :disabled="!activePage || panel.pages[panel.pages.length - 1]?.id === activePageId"
                  @click="movePage(activePageId, 1)"
                >
                  <ArrowDown :size="13" />
                </button>
              </div>
            </div>
            <template v-if="activePage">
              <div class="row">
                <div><div class="row-label">页面名称</div></div>
                <div class="inline-set">
                  <input v-model="activePage.title" class="field w-220" maxlength="64" @change="savePageMeta" />
                </div>
              </div>
              <div class="row">
                <div><div class="row-label">分栏</div><div class="row-hint">桌面端显示为 1 / 2 / 3 列，手机自动单列。</div></div>
                <select v-model.number="activePage.columns" class="field w-120" @change="savePageMeta">
                  <option :value="1">1 列</option>
                  <option :value="2">2 列</option>
                  <option :value="3">3 列</option>
                </select>
              </div>
              <div class="block-foot">
                <button class="btn danger" type="button" :disabled="panel.pages.length <= 1" @click="removePage">
                  <Trash2 :size="14" /> 删除当前页面
                </button>
              </div>
            </template>
            <div class="add-row">
              <input
                v-model="newPageTitle"
                class="field"
                placeholder="新页面名称"
                maxlength="64"
                @keyup.enter="addPage"
              />
              <button class="btn primary" type="button" @click="addPage">
                <Plus :size="14" /> 新建页面
              </button>
            </div>
          </div>

          <div class="block">
            <h3>{{ activePage?.title || '当前页面' }} · 分组</h3>
            <div class="group-row" v-for="(group, index) in pageGroups" :key="group.id">
              <span class="order">{{ index + 1 }}</span>
              <input v-model="group.title" class="field" @change="renameGroup(group)" />
              <div class="group-tools">
                <button class="mini icon" title="上移" :disabled="index === 0" @click="moveGroup(index, -1)">
                  <ArrowUp :size="13" />
                </button>
                <button
                  class="mini icon"
                  title="下移"
                  :disabled="index === pageGroups.length - 1"
                  @click="moveGroup(index, 1)"
                >
                  <ArrowDown :size="13" />
                </button>
                <button class="mini icon danger" title="删除" @click="removeGroup(group)">
                  <Trash2 :size="13" />
                </button>
              </div>
            </div>

            <div class="add-row">
              <input
                v-model="newGroup"
                class="field"
                placeholder="新分组名称"
                maxlength="32"
                @keyup.enter="addGroup"
              />
              <button class="btn primary" type="button" @click="addGroup">
                <Plus :size="14" /> 新建分组
              </button>
            </div>
          </div>
        </section>

        <!-- 服务状态 -->
        <section v-else-if="active === 'status'" class="pane status-pane">
          <ServiceStatus />
        </section>

        <!-- 图库管理 -->
        <section v-else-if="active === 'library'" class="pane">
          <header class="pane-head">
            <h2>图库管理</h2>
            <p>壁纸与应用图标保存在本地数据目录，支持分类移动、重命名、批量压缩、批量删除与云端同步；列表优先加载缩略图。</p>
          </header>

          <div class="library-bar">
            <div class="seg">
              <button :class="{ active: libraryType === 1 }" @click="libraryType = 1">壁纸图库</button>
              <button :class="{ active: libraryType === 2 }" @click="libraryType = 2">应用图标</button>
            </div>
            <select v-if="isAdmin" v-model="libraryScope" class="field w-120">
              <option value="mine">我的文件</option>
              <option value="all">全部用户</option>
            </select>
            <span class="library-stat">{{ files.length }} 个 · {{ formatSize(libraryTotalSize) }}</span>
            <div class="spacer" />
            <button class="btn" type="button" @click="toggleSelectAll">
              <SquareCheck v-if="isAllSelected" :size="14" />
              <Square v-else :size="14" />
              {{ isAllSelected ? '取消全选' : '全选' }}
            </button>
            <button class="btn" type="button" @click="libraryInput?.click()">
              <Upload :size="14" /> 上传
            </button>
            <input ref="libraryInput" type="file" accept="image/*" multiple hidden @change="uploadLibrary" />
          </div>

          <div v-if="selectedFileIds.length" class="library-actions">
            <span>已选 {{ selectedFileIds.length }} 个</span>
            <button class="mini" type="button" @click="moveSelected(libraryType === 1 ? 2 : 1)">
              <MoveRight :size="13" /> 移到{{ libraryType === 1 ? '应用图标' : '壁纸图库' }}
            </button>
            <button class="mini" type="button" :disabled="compressing" @click="compressSelected">
              <Minimize2 :size="13" /> {{ compressing ? '压缩中…' : '压缩' }}
            </button>
            <button class="mini" type="button" @click="pushSelectedToCloud">
              <CloudUpload :size="13" /> {{ isAdmin ? '推送到云端' : '申请同步' }}
            </button>
            <button class="mini danger" type="button" @click="removeSelected">
              <Trash2 :size="13" /> 删除
            </button>
          </div>

          <div v-if="libraryLoading" class="empty-tip">加载中...</div>
          <div v-else-if="files.length === 0" class="empty-tip">还没有文件，先上传一张吧。</div>
          <div v-else class="file-grid">
            <div
              v-for="file in files"
              :key="file.id"
              class="file-item"
              :class="{ selected: selectedFileIds.includes(file.id) }"
            >
              <button class="file-select" type="button" @click.stop="toggleFile(file.id)">
                <SquareCheck v-if="selectedFileIds.includes(file.id)" :size="15" />
                <Square v-else :size="15" />
              </button>
              <a class="thumb" :href="file.src" target="_blank" rel="noopener noreferrer" :title="file.fileName">
                <img :src="file.thumb || file.src" :alt="file.fileName" loading="lazy" />
              </a>
              <div class="file-meta">
                <button class="file-name" type="button" :title="`${file.fileName}（点击重命名）`" @click="openRename(file)">
                  {{ file.fileName }}
                </button>
                <div class="file-sub">
                  <span>{{ isAdmin && libraryScope === 'all' ? `UID ${file.userId} · ` : '' }}{{ formatSize(file.size) }}</span>
                  <span class="sync-badge" :class="`s-${file.syncStatus}`">{{ syncLabel(file.syncStatus) }}</span>
                </div>
                <div class="file-actions">
                  <button
                    v-if="libraryType === 1"
                    class="mini"
                    type="button"
                    @click="applyWallpaper(file)"
                  >
                    设为背景
                  </button>
                  <button class="mini" type="button" title="复制图片链接" @click="copyFileLink(file)">
                    复制链接
                  </button>
                  <button class="mini icon danger" type="button" @click="removeFile(file)">
                    <Trash2 :size="12" />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <PromptModal
            v-model:show="renameShow"
            title="重命名文件"
            :value="renameValue"
            confirm-text="保存"
            @confirm="confirmRename"
          />
        </section>

        <!-- 导出导入 -->
        <section v-else-if="active === 'data'" class="pane">
          <header class="pane-head">
            <h2>导出导入</h2>
            <p>普通账号可导出自己的卡片 JSON；管理员还可导出包含上传图片的整站 ZIP，并使用 AES-256-GCM 加密。</p>
          </header>

          <div class="block">
            <h3>我的卡片数据（JSON）</h3>
            <p class="block-desc">包含当前账号的分组、卡片、风格与搜索引擎配置，不含图片二进制。</p>
            <div class="inline-set">
              <button class="btn primary" type="button" @click="exportBackup">
                <Download :size="14" /> 导出 JSON
              </button>
              <button class="btn" type="button" :disabled="importing" @click="importInput?.click()">
                <Upload :size="14" /> {{ importing ? '导入中...' : '导入 JSON' }}
              </button>
              <input ref="importInput" type="file" accept="application/json,.json" hidden @change="onImportFile" />
            </div>
          </div>

          <template v-if="isAdmin">
            <div class="block">
              <h3>整站备份（ZIP + 可选 AES-256 加密）</h3>
              <p class="block-desc">
                包含全部用户、分组、卡片、系统设置、图库记录与 uploads 图片。设置密码后导出为
                <code>.ffbak</code> 加密包，适合上传到私有仓库或对象存储。
              </p>
              <div class="row">
                <div><div class="row-label">加密密码</div><div class="row-hint">留空则导出未加密 ZIP</div></div>
                <input v-model="siteBackupPassword" class="field w-220" type="password" autocomplete="new-password" placeholder="至少 6 位，可留空" />
              </div>
              <div class="row">
                <div><div class="row-label">确认密码</div></div>
                <input v-model="siteBackupConfirm" class="field w-220" type="password" autocomplete="new-password" placeholder="再次输入" />
              </div>
              <div class="block-foot">
                <button class="btn primary" type="button" :disabled="siteBackupExporting" @click="exportSiteBackup">
                  <FileArchive :size="14" /> {{ siteBackupExporting ? '打包中…' : '导出整站备份' }}
                </button>
              </div>
            </div>

            <div class="block">
              <h3>整站还原</h3>
              <p class="block-desc">
                支持本工具导出的 ZIP / .ffbak。还原会覆盖现有用户、分组、卡片、图库记录与上传目录，请先导出当前备份。
              </p>
              <div class="inline-set">
                <button class="btn" type="button" @click="siteBackupInput?.click()">
                  <Upload :size="14" /> 选择备份文件
                </button>
                <input ref="siteBackupInput" type="file" accept=".zip,.ffbak,.enc,.bin" hidden @change="onSiteBackupFile" />
                <span class="file-picked">{{ siteBackupFile ? `${siteBackupFile.name} · ${formatSize(siteBackupFile.size)}` : '未选择文件' }}</span>
              </div>
              <div class="row">
                <div><div class="row-label">解密密码</div><div class="row-hint">未加密 ZIP 可留空</div></div>
                <input v-model="siteBackupImportPassword" class="field w-220" type="password" autocomplete="new-password" placeholder="加密备份密码" />
              </div>
              <div class="block-foot">
                <button class="btn danger" type="button" :disabled="siteBackupImporting" @click="importSiteBackup">
                  <RefreshCw :size="14" /> {{ siteBackupImporting ? '还原中…' : '开始还原' }}
                </button>
              </div>
            </div>
          </template>

          <div class="block">
            <h3><Link2 :size="13" class="inline-ico" /> 导入浏览器书签</h3>
            <p class="block-desc">
              支持 Chrome / Edge / Firefox 导出的“书签 HTML”。按文件夹分组或全部放入一个分组，自动跳过重复网址；再次导入时可复用同名分组并更新已有标题。
            </p>
            <div class="inline-set">
              <button class="btn" type="button" @click="bookmarkInput?.click()">
                <Upload :size="14" /> 选择书签 HTML
              </button>
              <input ref="bookmarkInput" type="file" accept=".html,.htm,text/html" hidden @change="onBookmarkFile" />
              <span class="file-picked">{{ bookmarkFile ? bookmarkFile.name : '未选择文件' }}</span>
            </div>
            <div class="row">
              <div><div class="row-label">分组方式</div></div>
              <select v-model="bookmarkMode" class="field w-220">
                <option value="folders">按书签文件夹分组</option>
                <option value="single">全部放入一个分组</option>
              </select>
            </div>
            <div class="row">
              <div><div class="row-label">目标分组</div><div class="row-hint">选择后会追加到已有分组</div></div>
              <select v-model.number="bookmarkGroupId" class="field w-220">
                <option :value="0">自动创建新分组</option>
                <option v-for="group in panel.groups" :key="group.id" :value="group.id">{{ group.title }}</option>
              </select>
            </div>
            <div class="row">
              <div><div class="row-label">复用同名分组</div><div class="row-hint">重复导入时不再创建同名分组</div></div>
              <n-switch v-model:value="bookmarkReuseGroups" size="small" />
            </div>
            <div class="row">
              <div><div class="row-label">更新已有书签标题</div><div class="row-hint">网址已存在时，用新标题覆盖旧标题</div></div>
              <n-switch v-model:value="bookmarkUpdateExisting" size="small" />
            </div>
            <div class="block-foot">
              <button class="btn primary" type="button" :disabled="bookmarkImporting" @click="importBookmarks">
                <Link2 :size="14" /> {{ bookmarkImporting ? '导入中…' : '开始导入' }}
              </button>
            </div>
          </div>

          <div class="block muted-block">
            <h3>云端加密备份</h3>
            <p class="block-desc">配置 GitHub / R2 后，可在「云存储与漫游 → 云端加密备份」中一键上传与还原。</p>
            <button class="mini" type="button" @click="active = 'cloud'">前往云存储设置</button>
          </div>
        </section>

        <!-- 账号管理 -->
        <section v-else-if="active === 'account'" class="pane">
          <header class="pane-head">
            <h2>账号管理</h2>
            <p>访客模式、站点信息与用户账号。</p>
          </header>

          <div v-if="isAdmin" class="block">
            <h3>公开访客模式</h3>
            <div class="row">
              <div>
                <div class="row-label">允许未登录访客浏览主页</div>
                <div class="row-hint">访客只能查看，不能新增、修改或删除任何内容。</div>
              </div>
              <n-switch v-model:value="publicMode" size="small" @update:value="savePublicMode" />
            </div>
            <div class="row">
              <div><div class="row-label">站点标题</div></div>
              <div class="inline-set">
                <input v-model="siteTitle" class="field w-200" maxlength="40" />
                <button class="mini" type="button" @click="saveSiteTitle">保存</button>
              </div>
            </div>
          </div>
          <div v-else class="block muted-block">
            <h3>需要管理员权限</h3>
            <p class="block-desc">公开访客模式与站点设置仅管理员可修改。</p>
          </div>

          <div class="block">
            <h3>当前会话</h3>
            <div class="row">
              <div>
                <div class="row-label">{{ auth.user?.nickname || auth.user?.username }}</div>
                <div class="row-hint">{{ roleLabel }} · UID {{ auth.user?.id }}</div>
              </div>
              <button class="btn danger" type="button" @click="logout">
                <LogOut :size="14" /> 退出登录
              </button>
            </div>
          </div>

          <div v-if="isAdmin" class="block">
            <h3>用户管理</h3>
            <p class="block-desc">普通用户拥有独立的分组与卡片空间，互相不可见。</p>

            <div v-if="usersLoading" class="empty-tip small">加载中...</div>
            <div v-else class="user-list">
              <div v-for="user in users" :key="user.id" class="user-row">
                <span class="user-avatar">{{ (user.nickname || user.username).slice(0, 1) }}</span>
                <div class="user-main">
                  <strong>
                    {{ user.nickname || user.username }}
                    <em v-if="user.id === auth.user?.id">当前</em>
                  </strong>
                  <span>@{{ user.username }} · UID {{ user.id }}</span>
                </div>
                <select
                  class="field user-role"
                  :value="user.role"
                  :disabled="user.id === 1 || user.id === auth.user?.id"
                  @change="changeUserRole(user, Number(($event.target as HTMLSelectElement).value))"
                >
                  <option :value="1">管理员</option>
                  <option :value="2">普通用户</option>
                </select>
                <button
                  class="mini"
                  type="button"
                  :disabled="user.id === 1 || user.id === auth.user?.id"
                  @click="toggleUser(user)"
                >
                  {{ user.status === 1 ? '禁用' : '启用' }}
                </button>
                <button
                  class="mini icon danger"
                  type="button"
                  :disabled="user.id === 1 || user.id === auth.user?.id"
                  @click="removeUser(user)"
                >
                  <Trash2 :size="12" />
                </button>
              </div>
            </div>

            <div class="add-user">
              <input v-model="newUser.username" class="field" placeholder="用户名" maxlength="32" />
              <input v-model="newUser.nickname" class="field" placeholder="昵称（可选）" maxlength="32" />
              <input
                v-model="newUser.password"
                class="field"
                type="password"
                placeholder="密码（至少 6 位）"
                autocomplete="new-password"
              />
              <select v-model.number="newUser.role" class="field user-role">
                <option :value="2">普通用户</option>
                <option :value="1">管理员</option>
              </select>
              <button class="btn primary" type="button" :disabled="creatingUser" @click="createUser">
                <Plus :size="14" /> {{ creatingUser ? '创建中...' : '添加' }}
              </button>
            </div>
          </div>
          <div v-else class="block muted-block">
            <h3>用户管理</h3>
            <p class="block-desc">仅管理员可以创建、禁用或删除账号。</p>
          </div>
        </section>

        <!-- 高级设置 -->
        <section v-else-if="active === 'advanced'" class="pane">
          <header class="pane-head">
            <h2>高级设置</h2>
            <p>自定义代码与公告通知，修改后对全站生效。</p>
          </header>

          <template v-if="isAdmin">
            <div class="block">
              <h3>自定义 CSS</h3>
              <p class="block-desc">会注入到首页与登录页，可用于微调间距、圆角、字体等细节。</p>
              <textarea
                v-model="customCss"
                class="field mono code-area"
                rows="7"
                spellcheck="false"
                placeholder=".nav-card { border-radius: 20px; }"
              />
            </div>

            <div class="block">
              <h3>自定义 JavaScript</h3>
              <p class="block-desc">
                会作为普通脚本在页面加载后执行，适合接入统计或自定义交互。请只粘贴可信代码。
              </p>
              <textarea
                v-model="customJs"
                class="field mono code-area"
                rows="6"
                spellcheck="false"
                placeholder="console.log('Firefly Atlas ready')"
              />
              <div class="block-foot">
                <button class="btn primary" type="button" :disabled="savingCode" @click="saveCustomCode">
                  {{ savingCode ? '保存中…' : '保存自定义代码' }}
                </button>
              </div>
            </div>

            <div class="block">
              <h3><Megaphone :size="13" class="inline-ico" /> 公告通知</h3>
              <p class="block-desc">向访客或成员展示一条可关闭的公告，内容更新后会自动重新出现。</p>
              <div class="row">
                <div><div class="row-label">启用公告</div></div>
                <n-switch v-model:value="noticeEnabled" size="small" />
              </div>
              <div class="row">
                <div><div class="row-label">公告标题</div></div>
                <input v-model="noticeTitle" class="field w-260" maxlength="40" placeholder="例如：服务维护通知" />
              </div>
              <div class="row">
                <div><div class="row-label">显示位置</div></div>
                <select v-model="noticeScope" class="field w-200">
                  <option value="home">仅首页</option>
                  <option value="login">仅登录页</option>
                  <option value="both">首页与登录页</option>
                </select>
              </div>
              <div class="row vertical">
                <div><div class="row-label">公告内容</div><div class="row-hint">支持换行，最多 500 字</div></div>
                <textarea v-model="noticeContent" class="field" rows="4" maxlength="500" />
              </div>
              <div class="block-foot">
                <button class="btn primary" type="button" :disabled="savingNotice" @click="saveNotice">
                  {{ savingNotice ? '保存中…' : '保存公告' }}
                </button>
              </div>
            </div>
          </template>
          <div v-else class="block muted-block">
            <h3>需要管理员权限</h3>
            <p class="block-desc">自定义 CSS、JavaScript 与公告通知仅管理员可修改。</p>
          </div>
        </section>

        <!-- 云存储 -->
        <section v-else-if="active === 'cloud'" class="pane">
          <header class="pane-head">
            <h2>云存储与漫游</h2>
            <p>把图库与加密备份放到自己的 GitHub 私有仓库或 S3 / R2 对象存储。</p>
          </header>

          <div v-if="!isAdmin" class="block muted-block">
            <h3>需要管理员权限</h3>
            <p class="block-desc">
              云存储配置、云端图库与备份还原仅管理员可操作。普通用户可以在「图库管理」中选中图片并提交同步申请。
            </p>
          </div>

          <template v-else>
            <div class="cloud-tabs">
              <button :class="{ active: cloudActiveTab === 'images' }" type="button" @click="cloudActiveTab = 'images'">
                <Cloud :size="14" /> 云端图库
              </button>
              <button :class="{ active: cloudActiveTab === 'audit' }" type="button" @click="cloudActiveTab = 'audit'">
                <ShieldCheck :size="14" /> 待审图片
                <span v-if="cloudAuditList.length" class="cloud-count">{{ cloudAuditList.length }}</span>
              </button>
              <button :class="{ active: cloudActiveTab === 'config' }" type="button" @click="cloudActiveTab = 'config'">
                <Server :size="14" /> 存储配置
              </button>
              <button :class="{ active: cloudActiveTab === 'backups' }" type="button" @click="cloudActiveTab = 'backups'">
                <FileArchive :size="14" /> 加密备份
              </button>
            </div>

            <!-- 云端图库 -->
            <div v-if="cloudActiveTab === 'images'" class="cloud-pane">
              <div class="cloud-toolbar">
                <div class="seg">
                  <button :class="{ active: cloudImageCategory === 'all' }" type="button" @click="cloudImageCategory = 'all'">全部</button>
                  <button :class="{ active: cloudImageCategory === 'wallpapers' }" type="button" @click="cloudImageCategory = 'wallpapers'">壁纸</button>
                  <button :class="{ active: cloudImageCategory === 'icons' }" type="button" @click="cloudImageCategory = 'icons'">图标</button>
                </div>
                <span class="library-stat">{{ filteredCloudImages.length }} 张 · {{ formatSize(cloudTotalSize) }}</span>
                <div class="spacer" />
                <button class="mini" type="button" :disabled="cloudImagesLoading || !cloudConfig.enabled" @click="loadCloudImages">
                  <RefreshCw :size="13" /> 刷新
                </button>
                <button class="mini" type="button" @click="toggleAllCloudImages">
                  <SquareCheck v-if="isAllCloudSelected" :size="13" />
                  <Square v-else :size="13" />
                  全选当前
                </button>
                <button class="mini" type="button" :disabled="cloudSelectedKeys.length === 0 || cloudBusy || !cloudConfig.enabled" @click="syncSelectedCloud">
                  <CloudDownload :size="13" /> 拉取到本地 ({{ cloudSelectedKeys.length }})
                </button>
                <button class="mini" type="button" :disabled="cloudBusy || !cloudConfig.enabled" @click="syncAllCloud">
                  <HardDriveDownload :size="13" /> 一键全部拉取
                </button>
                <button class="mini danger" type="button" :disabled="cloudSelectedKeys.length === 0 || cloudBusy || !cloudConfig.enabled" @click="deleteSelectedCloud">
                  <Trash2 :size="13" /> 删除
                </button>
                <button class="mini danger" type="button" :disabled="cloudBusy || cloudImages.length === 0 || !cloudConfig.enabled" @click="clearCloudImages">
                  清空云端
                </button>
              </div>

              <div v-if="cloudImagesLoading" class="empty-tip">加载云端图库...</div>
              <div v-else-if="filteredCloudImages.length === 0" class="empty-tip">
                云端还没有图片。先在「图库管理」上传，再推送到云端。
              </div>
              <div v-else class="cloud-grid">
                <div
                  v-for="img in filteredCloudImages"
                  :key="img.key"
                  class="cloud-item"
                  :class="{ selected: cloudSelectedKeys.includes(img.key) }"
                >
                  <button class="file-select" type="button" @click="toggleCloudImage(img.key)">
                    <SquareCheck v-if="cloudSelectedKeys.includes(img.key)" :size="15" />
                    <Square v-else :size="15" />
                  </button>
                  <a class="cloud-thumb" :href="img.url || '#'" target="_blank" rel="noopener noreferrer">
                    <img :src="img.url" :alt="img.name" loading="lazy" />
                  </a>
                  <div class="cloud-meta">
                    <span :title="img.name">{{ img.name }}</span>
                    <small>{{ formatSize(img.size) }} · {{ img.category === 2 ? '图标' : '壁纸' }}</small>
                  </div>
                </div>
              </div>
            </div>

            <!-- 待审图片 -->
            <div v-else-if="cloudActiveTab === 'audit'" class="cloud-pane">
              <div class="cloud-toolbar">
                <span class="library-stat">待审核 {{ cloudAuditList.length }} 个</span>
                <div class="spacer" />
                <button class="mini" type="button" :disabled="cloudAuditLoading" @click="loadCloudAudit">
                  <RefreshCw :size="13" /> 刷新
                </button>
                <button class="mini" type="button" @click="toggleAllAudit">
                  <SquareCheck v-if="cloudSelectedAuditIds.length === cloudAuditList.length && cloudAuditList.length > 0" :size="13" />
                  <Square v-else :size="13" />
                  全选
                </button>
                <button class="mini" type="button" :disabled="cloudSelectedAuditIds.length === 0 || cloudBusy || !cloudConfig.enabled" @click="auditSelected('approve')">
                  <Check :size="13" /> 批准并上云 ({{ cloudSelectedAuditIds.length }})
                </button>
                <button class="mini danger" type="button" :disabled="cloudSelectedAuditIds.length === 0 || cloudBusy" @click="auditSelected('reject')">
                  <X :size="13" /> 驳回
                </button>
              </div>
              <div v-if="cloudAuditLoading" class="empty-tip">加载待审列表...</div>
              <div v-else-if="cloudAuditList.length === 0" class="empty-tip">当前没有待审核的同步申请。</div>
              <div v-else class="cloud-grid">
                <div
                  v-for="file in cloudAuditList"
                  :key="file.id"
                  class="cloud-item"
                  :class="{ selected: cloudSelectedAuditIds.includes(file.id) }"
                >
                  <button class="file-select" type="button" @click="toggleAudit(file.id)">
                    <SquareCheck v-if="cloudSelectedAuditIds.includes(file.id)" :size="15" />
                    <Square v-else :size="15" />
                  </button>
                  <a class="cloud-thumb" :href="file.src" target="_blank" rel="noopener noreferrer">
                    <img :src="file.thumb || file.src" :alt="file.fileName" loading="lazy" />
                  </a>
                  <div class="cloud-meta">
                    <span :title="file.fileName">{{ file.fileName }}</span>
                    <small>{{ file.userName }} · {{ file.fileType === 2 ? '图标' : '壁纸' }}</small>
                  </div>
                </div>
              </div>
            </div>

            <!-- 存储配置 -->
            <div v-else-if="cloudActiveTab === 'config'" class="cloud-pane">
              <div v-if="cloudConfigLoading" class="empty-tip">加载云存储配置...</div>
              <template v-else>
                <div class="block">
                  <h3>基础设置</h3>
                  <div class="row">
                    <div><div class="row-label">启用云存储</div><div class="row-hint">关闭后不会上传，也不影响本地图库。</div></div>
                    <n-switch v-model:value="cloudConfig.enabled" size="small" />
                  </div>
                  <div class="row">
                    <div><div class="row-label">本地 + 云端双写</div><div class="row-hint">新上传的图片自动镜像到云端，默认关闭。</div></div>
                    <n-switch v-model:value="cloudConfig.dualWrite" size="small" />
                  </div>
                  <div class="row">
                    <div><div class="row-label">存储引擎</div></div>
                    <select v-model="cloudConfig.provider" class="field w-220">
                      <option value="github">GitHub 仓库</option>
                      <option value="s3">Cloudflare R2 / S3 兼容</option>
                    </select>
                  </div>
                </div>

                <div v-if="cloudConfig.provider === 'github'" class="block">
                  <h3><Github :size="13" class="inline-ico" /> GitHub 仓库</h3>
                  <p class="block-desc">建议使用私有仓库；系统会自动探测仓库是 Private 还是 Public。</p>
                  <div class="grid-2">
                    <div class="row compact"><div class="row-label">Owner</div><input v-model="cloudConfig.github.owner" class="field" placeholder="your-name" /></div>
                    <div class="row compact"><div class="row-label">Repo</div><input v-model="cloudConfig.github.repo" class="field" placeholder="firefly-data" /></div>
                    <div class="row compact"><div class="row-label">Branch</div><input v-model="cloudConfig.github.branch" class="field" placeholder="main" /></div>
                    <div class="row compact"><div class="row-label">Base Path</div><input v-model="cloudConfig.github.basePath" class="field" placeholder="firefly-atlas" /></div>
                  </div>
                  <div class="row">
                    <div><div class="row-label">Personal Access Token</div><div class="row-hint">需要 repo 读写权限，仅保存在本机数据库。</div></div>
                    <input v-model="cloudConfig.github.token" class="field w-320" type="password" autocomplete="new-password" placeholder="ghp_..." />
                  </div>
                  <div class="row">
                    <div><div class="row-label">使用 jsDelivr CDN</div><div class="row-hint">仅公开仓库有效。</div></div>
                    <n-switch v-model:value="cloudConfig.github.useCdn" size="small" />
                  </div>
                </div>

                <div v-else class="block">
                  <h3><Server :size="13" class="inline-ico" /> Cloudflare R2 / S3</h3>
                  <p class="block-desc">R2 可只填 Account ID，Endpoint 会自动拼成 &lt;account&gt;.r2.cloudflarestorage.com。</p>
                  <div class="grid-2">
                    <div class="row compact"><div class="row-label">Account ID / Endpoint</div><input v-model="cloudConfig.s3.accountId" class="field" placeholder="R2 Account ID" /></div>
                    <div class="row compact"><div class="row-label">自定义 Endpoint</div><input v-model="cloudConfig.s3.endpoint" class="field" placeholder="可留空，如 s3.example.com" /></div>
                    <div class="row compact"><div class="row-label">Bucket</div><input v-model="cloudConfig.s3.bucket" class="field" placeholder="firefly" /></div>
                    <div class="row compact"><div class="row-label">Region</div><input v-model="cloudConfig.s3.region" class="field" placeholder="auto" /></div>
                    <div class="row compact"><div class="row-label">Access Key ID</div><input v-model="cloudConfig.s3.accessKeyId" class="field" placeholder="Access Key" /></div>
                    <div class="row compact"><div class="row-label">Secret Access Key</div><input v-model="cloudConfig.s3.secretAccessKey" class="field" type="password" autocomplete="new-password" placeholder="Secret Key" /></div>
                  </div>
                  <div class="row">
                    <div><div class="row-label">公开访问域名</div><div class="row-hint">可选，例如 https://img.example.com；留空视为私有存储。</div></div>
                    <input v-model="cloudConfig.s3.publicBaseUrl" class="field w-320" placeholder="https://img.example.com" />
                  </div>
                  <div class="row">
                    <div><div class="row-label">使用 HTTP</div><div class="row-hint">仅局域网 MinIO 等自建服务需要开启。</div></div>
                    <n-switch v-model:value="cloudConfig.s3.insecure" size="small" />
                  </div>
                </div>

                <div class="block">
                  <div class="inline-set">
                    <button class="btn" type="button" :disabled="cloudTesting" @click="testCloudConnection">
                      <RefreshCw :size="14" /> {{ cloudTesting ? '测试中…' : '测试连接' }}
                    </button>
                    <button class="btn primary" type="button" :disabled="cloudSaving" @click="saveCloudConfig">
                      <Check :size="14" /> {{ cloudSaving ? '保存中…' : '保存配置' }}
                    </button>
                    <span v-if="cloudTestResult" class="test-result">{{ cloudTestResult }}</span>
                  </div>
                </div>
              </template>
            </div>

            <!-- 云端加密备份 -->
            <div v-else class="cloud-pane">
              <div class="block">
                <h3><FileArchive :size="13" class="inline-ico" /> 创建云端加密备份</h3>
                <p class="block-desc">
                  打包整站数据库与 uploads 图片后，用 AES-256-GCM（scrypt 派生密钥）加密，再上传到云端 backups/ 目录。
                </p>
                <div class="row">
                  <div><div class="row-label">加密密码</div><div class="row-hint">至少 6 位，不会保存在服务器。</div></div>
                  <input v-model="cloudBackupPassword" class="field w-220" type="password" autocomplete="new-password" placeholder="备份密码" />
                </div>
                <div class="row">
                  <div><div class="row-label">确认密码</div></div>
                  <input v-model="cloudBackupConfirm" class="field w-220" type="password" autocomplete="new-password" placeholder="再次输入" />
                </div>
                <div class="block-foot">
                  <button class="btn primary" type="button" :disabled="cloudUploadingBackup || !cloudConfig.enabled" @click="uploadCloudBackup">
                    <CloudUpload :size="14" /> {{ cloudUploadingBackup ? '加密上传中…' : '加密并上传云端' }}
                  </button>
                </div>
              </div>

              <div class="block">
                <div class="block-head">
                  <h3>云端备份历史</h3>
                  <button class="mini" type="button" :disabled="cloudBackupsLoading || !cloudConfig.enabled" @click="loadCloudBackups">
                    <RefreshCw :size="13" /> 刷新
                  </button>
                </div>
                <div v-if="cloudBackupsLoading" class="empty-tip">加载备份列表...</div>
                <div v-else-if="cloudBackups.length === 0" class="empty-tip">云端还没有加密备份。</div>
                <div v-else class="backup-list">
                  <div v-for="backup in cloudBackups" :key="backup.key" class="backup-row">
                    <FileArchive :size="16" />
                    <div class="backup-main">
                      <strong>{{ backup.name }}</strong>
                      <span>{{ formatSize(backup.size) }} · {{ formatCloudDate(backup.updatedAt) }}</span>
                    </div>
                    <button class="mini" type="button" title="下载到本地" :disabled="!cloudConfig.enabled" @click="downloadCloudBackup(backup)">
                      <Download :size="13" />
                    </button>
                    <button class="mini" type="button" :disabled="!cloudConfig.enabled" @click="openCloudRestore(backup)">还原</button>
                    <button class="mini icon danger" type="button" :disabled="!cloudConfig.enabled" @click="deleteCloudBackup(backup)">
                      <Trash2 :size="13" />
                    </button>
                  </div>
                </div>

                <div v-if="cloudRestoreShow" class="restore-panel">
                  <div class="row-label">还原 {{ cloudRestoreKey }}</div>
                  <div class="inline-set">
                    <input v-model="cloudRestorePassword" class="field w-220" type="password" autocomplete="new-password" placeholder="输入备份密码" />
                    <button class="btn danger" type="button" :disabled="cloudRestoring" @click="restoreCloudBackup">
                      {{ cloudRestoring ? '还原中…' : '确认还原' }}
                    </button>
                    <button class="btn" type="button" @click="cloudRestoreShow = false">取消</button>
                  </div>
                </div>
              </div>
            </div>
          </template>
        </section>

        <!-- 关于 -->
        <section v-else class="pane">
          <header class="pane-head">
            <h2>关于</h2>
            <p>一款为私有云、NAS 与家庭工作站准备的导航面板。</p>
          </header>

          <div class="about-hero">
            <div class="about-logo">F</div>
            <div>
              <strong>Firefly Atlas</strong>
              <div class="row-hint">Version {{ version }} · V1.3.2「Sentinel」</div>
            </div>
          </div>

          <div class="block">
            <h3>版本更新</h3>
            <p class="block-desc">从 GitHub Releases 检查最新版本；网络不可达时不会影响面板使用。</p>
            <div class="inline-set">
              <button class="btn" type="button" :disabled="updateChecking" @click="checkUpdates">
                <RefreshCw :size="14" /> {{ updateChecking ? '检查中…' : '检查更新' }}
              </button>
              <template v-if="updateState">
                <span v-if="updateState.hasUpdate" class="update-badge new">
                  新版本 v{{ updateState.latest }}
                </span>
                <span v-else class="update-badge">已是最新</span>
                <a v-if="updateState.hasUpdate" :href="updateState.url" target="_blank" rel="noopener noreferrer">
                  查看发布页
                </a>
              </template>
              <span v-else-if="updateError" class="update-error">{{ updateError }}</span>
            </div>
          </div>

          <div class="block">
            <h3>V1.2.0「Lumen」新增</h3>
            <div class="roadmap">
              <div class="roadmap-item"><Check :size="14" /><span>多页面标签与 1 / 2 / 3 列分栏，卡片可跨页移动</span></div>
              <div class="roadmap-item"><Check :size="14" /><span>图库缩略图持久化，JPEG / PNG 批量压缩并保留原图</span></div>
              <div class="roadmap-item"><Check :size="14" /><span>书签增量导入：复用同名分组、更新已存在网址标题</span></div>
              <div class="roadmap-item"><Check :size="14" /><span>数字快捷键、搜索前缀跳转与版本更新提示</span></div>
              <div class="roadmap-item"><Check :size="14" /><span>带图标的原生 GUI 启动器，不再依赖 cmd / VBScript</span></div>
            </div>
          </div>

          <div class="block">
            <h3>技术栈</h3>
            <div class="chips">
              <span>Vue 3</span>
              <span>TypeScript</span>
              <span>Vite</span>
              <span>Pinia</span>
              <span>Naive UI</span>
              <span>Tailwind CSS</span>
              <span>Go</span>
              <span>Gin</span>
              <span>GORM</span>
              <span>SQLite</span>
            </div>
          </div>

          <div class="block">
            <h3>开源与致谢</h3>
            <p class="block-desc">
              本项目的界面布局与功能思路参考了
              <a href="https://github.com/hslr-s/sun-panel" target="_blank" rel="noopener noreferrer">hslr-s/sun-panel</a>
              （作者：红烧猎人）及其二次开发版（作者：Roland），代码为独立实现；同时参考了 Glance、Dashy、Homarr、Homepage 与 FlatNas 的公开思路。感谢原作者与社区的开源工作。
            </p>
          </div>

          <div class="block">
            <h3>v1.3「Sentinel」方向</h3>
            <div class="roadmap">
              <div class="roadmap-item"><Check :size="14" /><span>服务状态历史、24 小时在线率与状态变化告警</span></div>
              <div class="roadmap-item"><Check :size="14" /><span>自定义小组件：安全取数、模板显示与缓存</span></div>
              <div class="roadmap-item"><Check :size="14" /><span>磁盘挂载点多选与可选登录验证码</span></div>
            </div>
          </div>
        </section>
      </div>
    </div>
  </RoundCardModal>
</template>

<style scoped>
.settings-shell {
  display: flex;
  height: 100%;
  min-height: 0;
}

.settings-nav {
  width: 196px;
  flex-shrink: 0;
  padding: 14px 10px;
  border-right: 1px solid var(--line);
  overflow-y: auto;
  background: rgba(var(--glass-rgb), 0.22);
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 10px 16px;
}
.brand-dot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 9px;
  background: color-mix(in srgb, var(--accent) 14%, transparent);
}
.brand-dot span {
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 14px color-mix(in srgb, var(--accent) 70%, transparent);
}
.nav-brand strong {
  display: block;
  font-size: 12.5px;
  color: var(--text-strong);
}
.nav-brand small {
  display: block;
  font-size: 10.5px;
  color: var(--text-faint);
  letter-spacing: 0.08em;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 9px 10px;
  margin-bottom: 2px;
  border: 1px solid transparent;
  border-radius: 9px;
  background: transparent;
  color: var(--text-soft);
  font-size: 12.5px;
  text-align: left;
  cursor: pointer;
  transition: all 0.16s;
}
.nav-item:hover {
  background: rgba(var(--glass-rgb), 0.45);
  color: var(--text-strong);
}
.nav-item.active {
  background: color-mix(in srgb, var(--accent) 10%, transparent);
  border-color: color-mix(in srgb, var(--accent) 22%, transparent);
  color: var(--accent-strong, #059669);
  font-weight: 600;
}
.nav-item em {
  margin-left: auto;
  font-style: normal;
  font-size: 9px;
  padding: 1px 5px;
  border-radius: 999px;
  background: rgba(var(--glass-rgb), 0.7);
  color: var(--text-faint);
  border: 1px solid var(--line);
}

.settings-content {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 22px 26px 30px;
}

.pane {
  max-width: 720px;
  margin: 0 auto;
}

.pane-head {
  margin-bottom: 18px;
}
.pane-head h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 650;
  color: var(--text-strong);
}
.pane-head p {
  margin: 5px 0 0;
  font-size: 12px;
  color: var(--text-soft);
}

.profile-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border-radius: 14px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.35);
  margin-bottom: 14px;
}
.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: 15px;
  background: linear-gradient(135deg, var(--accent), color-mix(in srgb, var(--accent) 55%, #3b82f6));
  color: #fff;
  font-size: 22px;
  font-weight: 700;
}
.profile-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.profile-meta strong {
  font-size: 14px;
}
.profile-meta span {
  font-size: 11.5px;
  color: var(--text-soft);
}
.profile-meta em {
  align-self: flex-start;
  font-style: normal;
  font-size: 10px;
  color: var(--accent-strong, #059669);
  background: color-mix(in srgb, var(--accent) 12%, transparent);
  padding: 1px 7px;
  border-radius: 999px;
  margin-top: 3px;
}
.joined {
  margin-left: auto;
  font-size: 11px;
  color: var(--text-faint);
}

.block {
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 14px 16px;
  background: rgba(var(--glass-rgb), 0.28);
  margin-bottom: 14px;
}
.block h3 {
  margin: 0 0 6px;
  font-size: 13px;
  font-weight: 650;
  color: var(--text-strong);
}
.inline-ico {
  display: inline-block;
  vertical-align: -2px;
  margin-right: 5px;
  color: var(--accent);
}
.code-area {
  width: 100%;
  resize: vertical;
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 11.5px;
  line-height: 1.6;
  min-height: 96px;
}
.block-desc {
  margin: 0 0 10px;
  font-size: 12px;
  line-height: 1.65;
  color: var(--text-soft);
}
.block-desc a {
  color: var(--accent-strong, #059669);
}
.block-foot {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
.muted-block {
  background: rgba(var(--glass-rgb), 0.16);
  border-style: dashed;
}

.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 9px 0;
  border-top: 1px dashed var(--line);
}
.block h3 + .row,
.block-desc + .row {
  border-top: none;
}
.row.vertical {
  align-items: flex-start;
  flex-direction: column;
}
.row-label {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--text-strong);
}
.row-hint {
  font-size: 11px;
  color: var(--text-faint);
  margin-top: 2px;
  line-height: 1.5;
}

.field.w-260 {
  width: 260px;
}
.field.w-200 {
  width: 200px;
}
.field.w-90 {
  width: 90px;
}
.field.w-80 {
  width: 80px;
}
.field.w-120 {
  width: 120px;
}

.seg {
  display: inline-flex;
  padding: 3px;
  border-radius: 9px;
  background: rgba(var(--glass-rgb), 0.5);
  border: 1px solid var(--line);
  flex-shrink: 0;
}
.seg button {
  border: none;
  background: transparent;
  padding: 5px 10px;
  font-size: 11.5px;
  color: var(--text-soft);
  border-radius: 6px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s;
}
.seg button.active {
  background: rgba(var(--glass-rgb), 0.96);
  color: var(--text-strong);
  box-shadow: 0 2px 8px -3px rgba(var(--shadow-color), 0.28);
}

.range {
  width: 220px;
  accent-color: var(--accent);
  cursor: pointer;
}
.range-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
}
.range-val {
  font-size: 11px;
  color: var(--text-faint);
  width: 38px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.color-set {
  display: flex;
  align-items: center;
  gap: 8px;
}
.color-input {
  width: 34px;
  height: 28px;
  border: 1px solid var(--line);
  border-radius: 7px;
  background: transparent;
  padding: 2px;
  cursor: pointer;
}
.mono {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 11px;
  color: var(--text-soft);
}

.mini {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 26px;
  padding: 0 9px;
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
.mini.icon {
  width: 26px;
  padding: 0;
}
.mini.danger:hover {
  color: #e5484d;
  border-color: rgba(229, 72, 77, 0.35);
}
.mini:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.inline-set {
  display: flex;
  align-items: center;
  gap: 6px;
}

.group-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px dashed var(--line);
}
.order {
  width: 20px;
  font-size: 11px;
  color: var(--text-faint);
  text-align: center;
}
.group-tools {
  display: flex;
  gap: 4px;
}
.add-row {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}
.add-row .field {
  min-width: 0;
}
.add-row .btn {
  flex-shrink: 0;
  white-space: nowrap;
}

.user-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}
.user-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 8px;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: rgba(var(--glass-rgb), 0.34);
}
.user-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 9px;
  flex-shrink: 0;
  background: linear-gradient(135deg, var(--accent), color-mix(in srgb, var(--accent) 55%, #3b82f6));
  color: #fff;
  font-size: 13px;
  font-weight: 700;
}
.user-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.user-main strong {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: var(--text-strong);
}
.user-main em {
  font-style: normal;
  font-size: 9px;
  font-weight: 600;
  padding: 0 5px;
  border-radius: 999px;
  color: var(--accent-strong, #059669);
  background: color-mix(in srgb, var(--accent) 12%, transparent);
}
.user-main span {
  font-size: 10.5px;
  color: var(--text-faint);
}
.user-role {
  width: 102px;
  flex-shrink: 0;
  padding: 5px 8px;
  font-size: 11.5px;
}
.add-user {
  display: flex;
  gap: 6px;
  padding-top: 10px;
  border-top: 1px dashed var(--line);
}
.add-user .field {
  min-width: 0;
  padding: 6px 9px;
  font-size: 11.5px;
}
.add-user .btn {
  flex-shrink: 0;
  white-space: nowrap;
  height: 30px;
}
.empty-tip.small {
  padding: 18px;
  margin-bottom: 10px;
}

.engine-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 10px;
}
.engine-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.engine-name {
  width: 108px;
  flex-shrink: 0;
  padding: 6px 9px;
  font-size: 11.5px;
}
.engine-url {
  flex: 1;
  min-width: 0;
  padding: 6px 9px;
  font-size: 11.5px;
}
.engine-add {
  display: flex;
  align-items: center;
  gap: 6px;
  padding-top: 10px;
  border-top: 1px dashed var(--line);
}
.engine-add .mini {
  flex-shrink: 0;
  white-space: nowrap;
}

.library-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.library-stat {
  font-size: 11px;
  color: var(--text-faint);
  white-space: nowrap;
}
.library-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin: -4px 0 12px;
  padding: 8px 10px;
  border: 1px dashed color-mix(in srgb, var(--accent) 35%, var(--line));
  border-radius: 10px;
  background: color-mix(in srgb, var(--accent) 5%, transparent);
  font-size: 11px;
  color: var(--text-soft);
}
.spacer {
  flex: 1;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 10px;
}
.file-item {
  position: relative;
  border: 1px solid var(--line);
  border-radius: 12px;
  overflow: hidden;
  background: rgba(var(--glass-rgb), 0.35);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.file-item.selected {
  border-color: color-mix(in srgb, var(--accent) 60%, transparent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 12%, transparent);
}
.file-select {
  position: absolute;
  top: 6px;
  left: 6px;
  z-index: 2;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 1px solid var(--glass-border);
  border-radius: 7px;
  background: rgba(var(--glass-rgb), 0.82);
  color: var(--accent-strong, #059669);
  cursor: pointer;
  backdrop-filter: blur(8px);
}
.thumb {
  aspect-ratio: 4 / 3;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    linear-gradient(45deg, rgba(120, 130, 150, 0.1) 25%, transparent 25%) 0 0/16px 16px,
    linear-gradient(-45deg, rgba(120, 130, 150, 0.1) 25%, transparent 25%) 0 0/16px 16px;
}
.thumb img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  padding: 6px;
}
.file-meta {
  padding: 7px 8px;
  border-top: 1px solid var(--line);
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.file-name {
  border: none;
  background: transparent;
  padding: 0;
  text-align: left;
  font-size: 10.5px;
  color: var(--text-soft);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}
.file-name:hover {
  color: var(--accent-strong, #059669);
}
.file-sub {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  font-size: 9.5px;
  color: var(--text-faint);
}
.sync-badge {
  padding: 1px 6px;
  border-radius: 999px;
  border: 1px solid var(--line);
  white-space: nowrap;
}
.sync-badge.s-2 {
  color: #059669;
  border-color: rgba(5, 150, 105, 0.32);
  background: rgba(5, 150, 105, 0.08);
}
.sync-badge.s-1 {
  color: #d97706;
  border-color: rgba(217, 119, 6, 0.32);
  background: rgba(217, 119, 6, 0.08);
}
.sync-badge.s-3 {
  color: #dc2626;
  border-color: rgba(220, 38, 38, 0.32);
  background: rgba(220, 38, 38, 0.08);
}
.file-actions {
  display: flex;
  justify-content: space-between;
  gap: 4px;
}
.file-actions .mini {
  white-space: nowrap;
}

/* Cloud panels */
.cloud-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 14px;
  padding-bottom: 10px;
  border-bottom: 1px dashed var(--line);
}
.cloud-tabs button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  border: 1px solid var(--line);
  border-radius: 9px;
  background: rgba(var(--glass-rgb), 0.45);
  color: var(--text-soft);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}
.cloud-tabs button.active {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 45%, transparent);
  background: color-mix(in srgb, var(--accent) 10%, transparent);
}
.cloud-count {
  min-width: 16px;
  padding: 0 5px;
  border-radius: 999px;
  background: var(--accent);
  color: #fff;
  font-size: 9.5px;
  line-height: 16px;
  text-align: center;
}
.cloud-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.cloud-toolbar .mini {
  white-space: nowrap;
}
.cloud-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  gap: 10px;
}
.cloud-item {
  position: relative;
  border: 1px solid var(--line);
  border-radius: 12px;
  overflow: hidden;
  background: rgba(var(--glass-rgb), 0.35);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.cloud-item.selected {
  border-color: color-mix(in srgb, var(--accent) 60%, transparent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 12%, transparent);
}
.cloud-thumb {
  display: block;
  aspect-ratio: 1;
  background:
    linear-gradient(45deg, rgba(120, 130, 150, 0.1) 25%, transparent 25%) 0 0/16px 16px,
    linear-gradient(-45deg, rgba(120, 130, 150, 0.1) 25%, transparent 25%) 0 0/16px 16px;
}
.cloud-thumb img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  padding: 6px;
}
.cloud-meta {
  padding: 7px 8px;
  border-top: 1px solid var(--line);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cloud-meta span {
  font-size: 10.5px;
  color: var(--text-soft);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cloud-meta small {
  font-size: 9.5px;
  color: var(--text-faint);
}
.block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.backup-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.backup-row {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 8px 10px;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: rgba(var(--glass-rgb), 0.32);
}
.backup-row > svg {
  flex-shrink: 0;
  color: var(--accent);
}
.backup-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.backup-main strong {
  font-size: 11.5px;
  color: var(--text-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.backup-main span {
  font-size: 10px;
  color: var(--text-faint);
}
.restore-panel {
  margin-top: 12px;
  padding: 12px;
  border: 1px dashed rgba(229, 72, 77, 0.4);
  border-radius: 10px;
  background: rgba(229, 72, 77, 0.05);
}
.test-result {
  max-width: 380px;
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-soft);
}
.file-picked {
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
  color: var(--text-faint);
}
.field.w-220 {
  width: 220px;
}
.field.w-320 {
  width: 320px;
}
.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 12px;
}
.row.compact {
  flex-direction: column;
  align-items: stretch;
  gap: 4px;
  padding: 4px 0;
  border-top: none;
}
.row.compact .field {
  width: 100%;
}

.empty-tip {
  padding: 40px;
  text-align: center;
  font-size: 12px;
  color: var(--text-faint);
  border: 1px dashed var(--line);
  border-radius: 14px;
}

.pwd-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.chips span {
  font-size: 11px;
  padding: 3px 9px;
  border-radius: 999px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.5);
  color: var(--text-soft);
}

.about-hero {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 14px;
}
.about-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 54px;
  height: 54px;
  border-radius: 16px;
  background: linear-gradient(135deg, var(--accent), color-mix(in srgb, var(--accent) 50%, #3b82f6));
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  box-shadow: 0 14px 30px -14px color-mix(in srgb, var(--accent) 70%, transparent);
}
.about-hero strong {
  font-size: 16px;
  color: var(--text-strong);
}
.update-badge {
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.5);
  color: var(--text-soft);
  font-size: 11px;
}
.update-badge.new {
  border-color: color-mix(in srgb, var(--accent) 45%, transparent);
  background: color-mix(in srgb, var(--accent) 12%, transparent);
  color: var(--accent-strong, #059669);
}
.update-error {
  color: var(--text-faint);
  font-size: 11px;
}

.roadmap {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 12px 0;
}
.roadmap-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 12px;
  color: var(--text-soft);
  line-height: 1.5;
}
.roadmap-item svg {
  margin-top: 2px;
  color: var(--accent);
  flex-shrink: 0;
}

.pill {
  display: inline-block;
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--line);
  color: var(--text-faint);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 14px;
  border-radius: 9px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.6);
  color: var(--text-strong);
  font-size: 12.5px;
  cursor: pointer;
  transition: all 0.16s;
}
.btn:hover {
  border-color: color-mix(in srgb, var(--accent) 35%, transparent);
}
.btn.primary {
  background: var(--accent);
  color: #fff;
  border-color: transparent;
}
.btn.danger {
  color: #e5484d;
  border-color: rgba(229, 72, 77, 0.25);
  background: rgba(229, 72, 77, 0.06);
}
.btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

@media (max-width: 760px) {
  .settings-shell {
    flex-direction: column;
  }
  .settings-nav {
    width: 100%;
    display: flex;
    gap: 4px;
    overflow-x: auto;
    border-right: none;
    border-bottom: 1px solid var(--line);
    padding: 8px;
  }
  .nav-brand {
    display: none;
  }
  .nav-item {
    width: auto;
    flex-shrink: 0;
    margin-bottom: 0;
  }
  .nav-item em {
    display: none;
  }
  .settings-content {
    padding: 16px;
  }
  .row {
    flex-wrap: wrap;
  }
  .range {
    width: 100%;
  }
  .field.w-260,
  .field.w-200,
  .field.w-220,
  .field.w-320 {
    width: 100%;
  }
  .grid-2 {
    grid-template-columns: 1fr;
  }
  .cloud-tabs {
    gap: 4px;
  }
  .cloud-tabs button {
    padding: 6px 9px;
    font-size: 11px;
  }
  .file-grid,
  .cloud-grid {
    grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  }
  .pwd-grid {
    grid-template-columns: 1fr;
  }
  .user-row {
    flex-wrap: wrap;
  }
  .user-main {
    flex-basis: calc(100% - 40px);
  }
  .add-user {
    flex-direction: column;
  }
}
</style>
