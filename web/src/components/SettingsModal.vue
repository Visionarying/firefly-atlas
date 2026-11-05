<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NSwitch, useDialog, useMessage } from 'naive-ui'
import {
  ArrowDown,
  ArrowUp,
  Check,
  Cloud,
  DatabaseBackup,
  Download,
  Images,
  Info,
  LayoutGrid,
  LogOut,
  Palette,
  Plus,
  ShieldCheck,
  Trash2,
  Upload,
  User,
  X,
} from 'lucide-vue-next'
import RoundCardModal from './RoundCardModal.vue'
import { authApi, backupApi, fileApi, panelApi, settingApi, userApi } from '@/api'
import { useAuthStore } from '@/store/auth'
import { usePanelStore } from '@/store/panel'
import { fontOptions } from '@/store/defaults'
import type { FileRecord, User as PanelUser } from '@/typings'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', value: boolean): void }>()

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
  { key: 'library', label: '图库管理', icon: Images },
  { key: 'data', label: '导出导入', icon: DatabaseBackup },
  { key: 'account', label: '账号管理', icon: ShieldCheck },
  { key: 'cloud', label: '云存储与漫游', icon: Cloud, badge: 'V2' },
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

/* ---------- groups ---------- */
const newGroup = ref('')

async function addGroup() {
  const title = newGroup.value.trim()
  if (!title) return
  try {
    const res = await panelApi.saveGroup({ title })
    panel.groups.push({ id: res.id, title, sort: panel.groups.length })
    newGroup.value = ''
    message.success('分组已创建')
  } catch (e: any) {
    message.error(e.message || '创建失败')
  }
}

async function renameGroup(group: { id: number; title: string }) {
  if (!group.title.trim()) return
  try {
    await panelApi.saveGroup({ id: group.id, title: group.title.trim() })
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
  const target = index + delta
  if (target < 0 || target >= panel.groups.length) return
  const list = [...panel.groups]
  const [item] = list.splice(index, 1)
  list.splice(target, 0, item)
  list.forEach((g, i) => (g.sort = i))
  panel.groups = list
  try {
    await panelApi.sortGroups(list.map((g) => g.id))
  } catch {
    /* ignore */
  }
}

/* ---------- library ---------- */
const libraryType = ref<1 | 2>(1)
const files = ref<FileRecord[]>([])
const libraryLoading = ref(false)
const libraryInput = ref<HTMLInputElement | null>(null)

async function loadFiles() {
  if (!auth.isLoggedIn) return
  libraryLoading.value = true
  try {
    files.value = await fileApi.list(libraryType.value)
  } catch {
    files.value = []
  } finally {
    libraryLoading.value = false
  }
}

async function uploadLibrary(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    await fileApi.upload(file, libraryType.value)
    message.success('上传成功')
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

function removeFile(file: FileRecord) {
  dialog.warning({
    title: '删除文件',
    content: `确定删除「${file.fileName}」吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await fileApi.remove(file.id)
        files.value = files.value.filter((f) => f.id !== file.id)
        if (config.backgroundImage === file.src) {
          config.backgroundImage = ''
          touch()
        }
      } catch (e: any) {
        message.error(e.message || '删除失败')
      }
    },
  })
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

/* ---------- settings / account ---------- */
const publicMode = ref(panel.settings.public_visit_mode === '1')
const siteTitle = ref(panel.settings.site_title || 'Firefly Panel')

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
    siteTitle.value = panel.settings.site_title || 'Firefly Panel'
    if (active.value === 'library') void loadFiles()
    if (active.value === 'account') void loadUsers()
  },
)

watch(active, (value) => {
  if (value === 'library') void loadFiles()
  if (value === 'account') void loadUsers()
})

watch(libraryType, () => {
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

async function logout() {
  await auth.logout()
  emit('update:show', false)
  router.push('/login')
}

const version = '1.0.0'
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
            <strong>Firefly Panel</strong>
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
          <em v-if="section.badge">{{ section.badge }}</em>
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
            <h2>分组管理</h2>
            <p>调整分组顺序与名称，删除会同时移除组内卡片。</p>
          </header>

          <div class="block">
            <div class="group-row" v-for="(group, index) in panel.groups" :key="group.id">
              <span class="order">{{ index + 1 }}</span>
              <input v-model="group.title" class="field" @change="renameGroup(group)" />
              <div class="group-tools">
                <button class="mini icon" title="上移" :disabled="index === 0" @click="moveGroup(index, -1)">
                  <ArrowUp :size="13" />
                </button>
                <button
                  class="mini icon"
                  title="下移"
                  :disabled="index === panel.groups.length - 1"
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

        <!-- 图库管理 -->
        <section v-else-if="active === 'library'" class="pane">
          <header class="pane-head">
            <h2>图库管理</h2>
            <p>壁纸与应用图标保存在本地数据目录，不会上传到第三方。</p>
          </header>

          <div class="library-bar">
            <div class="seg">
              <button :class="{ active: libraryType === 1 }" @click="libraryType = 1">壁纸图库</button>
              <button :class="{ active: libraryType === 2 }" @click="libraryType = 2">应用图标</button>
            </div>
            <div class="spacer" />
            <button class="btn" type="button" @click="libraryInput?.click()">
              <Upload :size="14" /> 上传
            </button>
            <input ref="libraryInput" type="file" accept="image/*" hidden @change="uploadLibrary" />
          </div>

          <div v-if="libraryLoading" class="empty-tip">加载中...</div>
          <div v-else-if="files.length === 0" class="empty-tip">还没有文件，先上传一张吧。</div>
          <div v-else class="file-grid">
            <div v-for="file in files" :key="file.id" class="file-item">
              <div class="thumb">
                <img :src="file.src" :alt="file.fileName" loading="lazy" />
              </div>
              <div class="file-meta">
                <span class="file-name" :title="file.fileName">{{ file.fileName }}</span>
                <div class="file-actions">
                  <button
                    v-if="libraryType === 1"
                    class="mini"
                    type="button"
                    @click="applyWallpaper(file)"
                  >
                    设为背景
                  </button>
                  <button class="mini icon danger" type="button" @click="removeFile(file)">
                    <Trash2 :size="12" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- 导出导入 -->
        <section v-else-if="active === 'data'" class="pane">
          <header class="pane-head">
            <h2>导出导入</h2>
            <p>备份包含分组、卡片、风格与系统设置。图片文件本身不包含在 JSON 中。</p>
          </header>

          <div class="block">
            <h3>导出备份</h3>
            <p class="block-desc">下载一份 JSON 快照，可用于迁移到另一台设备。</p>
            <button class="btn primary" type="button" @click="exportBackup">
              <Download :size="14" /> 导出 JSON 备份
            </button>
          </div>

          <div class="block">
            <h3>还原备份</h3>
            <p class="block-desc">导入会覆盖当前账号的分组、卡片与风格配置，请谨慎操作。</p>
            <button class="btn" type="button" :disabled="importing" @click="importInput?.click()">
              <Upload :size="14" /> {{ importing ? '导入中...' : '选择备份文件' }}
            </button>
            <input ref="importInput" type="file" accept="application/json,.json" hidden @change="onImportFile" />
          </div>

          <div class="block muted-block">
            <h3>云端加密备份</h3>
            <p class="block-desc">
              V2 计划接入 GitHub 私有仓库与 Cloudflare R2，支持 AES-256 加密备份、跨设备漫游与本地优先回源。
            </p>
            <span class="pill">规划中</span>
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

        <!-- 云存储 -->
        <section v-else-if="active === 'cloud'" class="pane">
          <header class="pane-head">
            <h2>云存储与漫游</h2>
            <p>把图库与备份放到你自己的云端。</p>
          </header>
          <div class="block muted-block">
            <h3>V2 能力规划</h3>
            <div class="roadmap">
              <div class="roadmap-item">
                <Check :size="14" />
                <span>GitHub 仓库存储：Contents / Blobs API，支持私有仓库与 Token</span>
              </div>
              <div class="roadmap-item">
                <Check :size="14" />
                <span>Cloudflare R2 / S3 兼容对象存储，自定义 Endpoint</span>
              </div>
              <div class="roadmap-item">
                <Check :size="14" />
                <span>本地优先 + 云端透明回源 + 本地磁盘自愈</span>
              </div>
              <div class="roadmap-item">
                <Check :size="14" />
                <span>AES-256-GCM 全站加密备份与一键还原</span>
              </div>
              <div class="roadmap-item">
                <Check :size="14" />
                <span>普通用户图库上传审核与漫游审批</span>
              </div>
            </div>
            <span class="pill">规划中</span>
          </div>
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
              <strong>Firefly Panel</strong>
              <div class="row-hint">Version {{ version }} · V1</div>
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
              本项目基于
              <a href="https://github.com/hslr-s/sun-panel" target="_blank" rel="noopener noreferrer">hslr-s/sun-panel</a>
              的 MIT 许可进行二次开发与架构重构，保留原作者版权与署名。维护者：Roland。
            </p>
          </div>

          <div class="block">
            <h3>V2 候选方向</h3>
            <div class="roadmap">
              <div class="roadmap-item"><Check :size="14" /><span>多用户与子账号独立空间</span></div>
              <div class="roadmap-item"><Check :size="14" /><span>云端漫游与加密备份</span></div>
              <div class="roadmap-item"><Check :size="14" /><span>服务健康探测与在线状态灯</span></div>
              <div class="roadmap-item"><Check :size="14" /><span>书签导入、浏览器插件与 PWA</span></div>
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
.spacer {
  flex: 1;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 10px;
}
.file-item {
  border: 1px solid var(--line);
  border-radius: 12px;
  overflow: hidden;
  background: rgba(var(--glass-rgb), 0.35);
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
  font-size: 10.5px;
  color: var(--text-soft);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-actions {
  display: flex;
  justify-content: space-between;
  gap: 4px;
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
  .field.w-200 {
    width: 100%;
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
