<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { Code2, Globe2, Image, Images, Link2, Upload, Check, Sparkles, Type } from 'lucide-vue-next'
import RoundCardModal from './RoundCardModal.vue'
import AppIcon from './AppIcon.vue'
import { panelApi, fileApi, systemApi } from '@/api'
import { usePanelStore } from '@/store/panel'
import type { FileRecord, IconConfig, ItemGroup, ItemIcon } from '@/typings'

const props = defineProps<{
  show: boolean
  item: ItemIcon | null
  groups: ItemGroup[]
  defaultGroupId?: number
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'saved'): void
}>()

const panel = usePanelStore()
const message = useMessage()
const saving = ref(false)
const uploadInput = ref<HTMLInputElement | null>(null)
const activeType = ref<1 | 2 | 3>(2)
const imageUrl = ref('')
const svgDraft = ref('')
const faviconLoading = ref(false)
const iconifyQuery = ref('')
const iconifyLoading = ref(false)
const iconifyResults = ref<string[]>([])
const iconifySvgs = ref<Record<string, string>>({})
const galleryOpen = ref(false)
const galleryLoading = ref(false)
const galleryIcons = ref<FileRecord[]>([])

const gradients = [
  'linear-gradient(135deg,#5b8cff,#3a5cff)',
  'linear-gradient(135deg,#22d3ee,#0ea5e9)',
  'linear-gradient(135deg,#34d399,#059669)',
  'linear-gradient(135deg,#fbbf24,#f59e0b)',
  'linear-gradient(135deg,#fb7185,#e11d48)',
  'linear-gradient(135deg,#a78bfa,#7c3aed)',
  'linear-gradient(135deg,#f472b6,#db2777)',
  'linear-gradient(135deg,#94a3b8,#475569)',
  '#0f172a',
  '#10b981',
]

const form = reactive({
  id: 0,
  title: '',
  url: '',
  lanUrl: '',
  description: '',
  openMethod: 1,
  healthCheck: false,
  itemIconGroupId: 0,
  icon: {
    itemType: 2,
    src: '',
    text: '',
    backgroundColor: gradients[0],
  } as IconConfig,
})

watch(
  () => props.show,
  (visible) => {
    if (!visible) return
    const item = props.item
    form.id = item?.id || 0
    form.title = item?.title || ''
    form.url = item?.url || ''
    form.lanUrl = item?.lanUrl || ''
    form.description = item?.description || ''
    form.openMethod = item?.openMethod || 1
    form.healthCheck = !!item?.healthCheck
    form.itemIconGroupId = item?.itemIconGroupId || props.defaultGroupId || props.groups[0]?.id || 0
    form.icon = {
      itemType: item?.icon?.itemType || 2,
      src: item?.icon?.src || '',
      text: item?.icon?.text || '',
      backgroundColor: item?.icon?.backgroundColor || gradients[0],
    }
    activeType.value = form.icon.itemType
    imageUrl.value = form.icon.itemType === 1 ? form.icon.src || '' : ''
    svgDraft.value = form.icon.itemType === 3 ? form.icon.src || '' : ''
  },
)

const isEdit = computed(() => form.id > 0)

function groupLabel(group: ItemGroup): string {
  const page = panel.pages.find((item) => item.id === group.pageId)
  return page ? `${page.title} / ${group.title}` : group.title
}

function switchType(type: 1 | 2 | 3) {
  activeType.value = type
  form.icon.itemType = type
  if (type === 1) {
    form.icon.src = imageUrl.value
  } else if (type === 3) {
    form.icon.src = svgDraft.value
  }
}

watch(svgDraft, (value) => {
  if (activeType.value === 3) form.icon.src = value
})

function pickGradient(value: string) {
  form.icon.backgroundColor = value
  form.icon.itemType = 2
  activeType.value = 2
}

function applyImageUrl() {
  if (!imageUrl.value.trim()) return
  form.icon.itemType = 1
  form.icon.src = imageUrl.value.trim()
  activeType.value = 1
}

async function onUploadIcon(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    const record = await fileApi.upload(file, 2)
    form.icon.itemType = 1
    form.icon.src = record.src
    imageUrl.value = record.src
    activeType.value = 1
    message.success('图标上传成功')
  } catch (err: any) {
    message.error(err.message || '上传失败')
  } finally {
    input.value = ''
  }
}

async function fetchFavicon() {
  const target = (form.url || form.lanUrl).trim()
  if (!target) {
    message.warning('请先填写公网或内网地址')
    return
  }
  faviconLoading.value = true
  try {
    const record = await systemApi.favicon(target)
    form.icon.itemType = 1
    form.icon.src = record.src
    imageUrl.value = record.src
    activeType.value = 1
    message.success('已获取网站图标并保存到本地')
  } catch (err: any) {
    message.error(err.message || '获取网站图标失败')
  } finally {
    faviconLoading.value = false
  }
}

function iconifySvgUrl(name: string): string {
  const [prefix, icon] = name.split(':')
  return `https://api.iconify.design/${prefix}/${icon}.svg`
}

async function searchIconify() {
  const query = iconifyQuery.value.trim()
  if (!query) return
  iconifyLoading.value = true
  iconifyResults.value = []
  iconifySvgs.value = {}
  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), 9000)
  try {
    const response = await fetch(
      `https://api.iconify.design/search?query=${encodeURIComponent(query)}&limit=24`,
      { signal: controller.signal },
    )
    if (!response.ok) throw new Error('search failed')
    const payload = (await response.json()) as { icons?: string[] }
    const names = (payload.icons || []).slice(0, 24)
    iconifyResults.value = names
    // Stop the button spinner as soon as the list arrives; previews load in the background.
    iconifyLoading.value = false
    if (!names.length) {
      message.info('没有找到匹配的图标，换个关键词试试')
      return
    }
    void loadIconifyPreviews(names)
  } catch {
    message.error('在线图标搜索失败，请检查网络后重试')
  } finally {
    window.clearTimeout(timer)
    iconifyLoading.value = false
  }
}

async function loadIconifyPreviews(names: string[]) {
  await Promise.all(
    names.map(async (name) => {
      try {
        const response = await fetch(iconifySvgUrl(name))
        if (!response.ok) return
        const svg = await response.text()
        iconifySvgs.value = { ...iconifySvgs.value, [name]: svg }
      } catch {
        /* ignore individual preview failures */
      }
    }),
  )
}

async function pickIconify(name: string) {
  let svg = iconifySvgs.value[name]
  if (!svg) {
    try {
      const response = await fetch(iconifySvgUrl(name))
      if (response.ok) svg = await response.text()
    } catch {
      /* ignore */
    }
  }
  if (!svg) {
    message.error('图标下载失败')
    return
  }
  svgDraft.value = svg
  form.icon.itemType = 3
  form.icon.src = svg
  activeType.value = 3
  message.success(`已使用 ${name}`)
}

async function toggleGallery() {
  galleryOpen.value = !galleryOpen.value
  if (!galleryOpen.value || galleryIcons.value.length > 0) return
  galleryLoading.value = true
  try {
    galleryIcons.value = await fileApi.publicIcons()
  } catch (err: any) {
    message.error(err.message || '读取管理员图库失败')
  } finally {
    galleryLoading.value = false
  }
}

function pickGalleryIcon(file: FileRecord) {
  imageUrl.value = file.src
  form.icon.itemType = 1
  form.icon.src = file.src
  activeType.value = 1
  galleryOpen.value = false
  message.success(`已使用 ${file.fileName}`)
}

async function save() {
  if (!form.title.trim()) {
    message.warning('请填写卡片名称')
    return
  }
  if (!form.itemIconGroupId) {
    message.warning('请选择分组')
    return
  }
  saving.value = true
  try {
    const payload: Partial<ItemIcon> = {
      id: form.id || undefined,
      title: form.title.trim(),
      url: form.url.trim(),
      lanUrl: form.lanUrl.trim(),
      description: form.description.trim(),
      openMethod: form.openMethod,
      healthCheck: form.healthCheck,
      itemIconGroupId: form.itemIconGroupId,
      icon: {
        itemType: form.icon.itemType,
        src: form.icon.src,
        text: form.icon.text || form.title.trim().slice(0, 1).toUpperCase(),
        backgroundColor: form.icon.backgroundColor,
      },
    }
    await panelApi.saveItem(payload)
    message.success(isEdit.value ? '卡片已更新' : '卡片已创建')
    emit('saved')
    emit('update:show', false)
  } catch (err: any) {
    message.error(err.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const previewIcon = computed<IconConfig>(() => ({
  ...form.icon,
  text: form.icon.text || form.title.slice(0, 1).toUpperCase() || '?',
}))
</script>

<template>
  <RoundCardModal
    :show="show"
    :title="isEdit ? '编辑卡片' : '新增卡片'"
    :width="620"
    :height="760"
    :min-width="360"
    :min-height="420"
    @update:show="emit('update:show', $event)"
  >
    <div class="editor">
      <div class="preview-box">
        <AppIcon :icon="previewIcon" :title="form.title" :size="56" :radius="15" />
        <div class="preview-meta">
          <strong>{{ form.title || '未命名卡片' }}</strong>
          <span>{{ form.description || '在这里预览卡片效果' }}</span>
        </div>
      </div>

      <div class="grid-2">
        <div>
          <label class="label">名称</label>
          <input v-model="form.title" class="field" maxlength="32" placeholder="例如：NAS 控制台" />
        </div>
        <div>
          <label class="label">所属分组</label>
          <select v-model.number="form.itemIconGroupId" class="field">
            <option v-for="group in groups" :key="group.id" :value="group.id">
              {{ groupLabel(group) }}
            </option>
          </select>
        </div>
      </div>

      <div>
        <label class="label">公网地址</label>
        <div class="input-icon">
          <Link2 :size="14" />
          <input v-model="form.url" class="field with-icon" placeholder="https://nas.example.com" />
        </div>
      </div>

      <div>
        <label class="label">内网地址 <span class="optional">可选，自动切换</span></label>
        <div class="input-icon">
          <Link2 :size="14" />
          <input v-model="form.lanUrl" class="field with-icon" placeholder="http://192.168.1.10:5000" />
        </div>
      </div>

      <div>
        <label class="label">描述</label>
        <input v-model="form.description" class="field" maxlength="64" placeholder="一句话说明这个服务" />
      </div>

      <div class="inline-setting">
        <div>
          <div class="label no-margin">在线状态探测</div>
          <div class="setting-hint">开启后首页卡片会显示绿色 / 红色状态点，每 60 秒探测一次。</div>
        </div>
        <button
          type="button"
          class="toggle"
          :class="{ active: form.healthCheck }"
          @click="form.healthCheck = !form.healthCheck"
        >
          <span />
        </button>
      </div>

      <div class="grid-2">
        <div>
          <label class="label">打开方式</label>
          <div class="seg">
            <button type="button" :class="{ active: form.openMethod === 1 }" @click="form.openMethod = 1">
              新标签页
            </button>
            <button type="button" :class="{ active: form.openMethod === 2 }" @click="form.openMethod = 2">
              当前页
            </button>
            <button type="button" :class="{ active: form.openMethod === 3 }" @click="form.openMethod = 3">
              页内弹窗
            </button>
          </div>
        </div>
        <div>
          <label class="label">图标文字（文本图标）</label>
          <input
            v-model="form.icon.text"
            class="field"
            maxlength="4"
            placeholder="留空自动取首字"
            @input="switchType(2)"
          />
        </div>
      </div>

      <div>
        <label class="label">图标</label>
        <div class="type-tabs">
          <button type="button" :class="{ active: activeType === 2 }" @click="switchType(2)">
            <Type :size="14" /> 文字
          </button>
          <button type="button" :class="{ active: activeType === 1 }" @click="switchType(1)">
            <Image :size="14" /> 图片
          </button>
          <button type="button" :class="{ active: activeType === 3 }" @click="switchType(3)">
            <Code2 :size="14" /> SVG
          </button>
        </div>

        <div v-if="activeType === 2" class="type-panel">
          <div class="swatches">
            <button
              v-for="color in gradients"
              :key="color"
              type="button"
              class="swatch"
              :style="{ background: color }"
              :class="{ active: form.icon.backgroundColor === color }"
              @click="pickGradient(color)"
            >
              <Check v-if="form.icon.backgroundColor === color" :size="12" />
            </button>
          </div>
          <input
            v-model="form.icon.backgroundColor"
            class="field mt-2"
            placeholder="任意 CSS 背景，例如 #10b981 或 linear-gradient(...)"
          />
        </div>

        <div v-else-if="activeType === 1" class="type-panel">
          <div class="upload-row">
            <input v-model="imageUrl" class="field" placeholder="图片 URL，或从本地上传" @keyup.enter="applyImageUrl" />
            <button type="button" class="btn" @click="applyImageUrl">使用</button>
            <button type="button" class="btn" @click="uploadInput?.click()">
              <Upload :size="14" /> 上传
            </button>
            <input ref="uploadInput" type="file" accept="image/*" hidden @change="onUploadIcon" />
          </div>
          <div class="favicon-row">
            <span class="favicon-hint">自动抓取网页 favicon 并保存到本地，避免外链失效。</span>
            <button type="button" class="btn" :disabled="faviconLoading" @click="fetchFavicon">
              <Globe2 :size="14" /> {{ faviconLoading ? '抓取中…' : '获取网站图标' }}
            </button>
          </div>
          <div class="gallery-row">
            <span class="favicon-hint">也可以直接选用管理员图库中已上传的图标。</span>
            <button type="button" class="btn" @click="toggleGallery">
              <Images :size="14" /> {{ galleryOpen ? '收起图库' : '从图库选择' }}
            </button>
          </div>
          <div v-if="galleryOpen" class="gallery-panel">
            <div v-if="galleryLoading" class="gallery-empty">加载中…</div>
            <div v-else-if="galleryIcons.length === 0" class="gallery-empty">
              管理员还没有上传图标，先到「图库管理」上传吧。
            </div>
            <div v-else class="gallery-grid">
              <button
                v-for="icon in galleryIcons"
                :key="icon.id"
                type="button"
                class="gallery-cell"
                :title="icon.fileName"
                @click="pickGalleryIcon(icon)"
              >
                <img :src="icon.thumb || icon.src" :alt="icon.fileName" loading="lazy" />
              </button>
            </div>
          </div>
        </div>

        <div v-else class="type-panel">
          <textarea
            v-model="svgDraft"
            class="field mono"
            rows="5"
            placeholder='<svg viewBox="0 0 24 24">...</svg>'
          />

          <div class="iconify">
            <div class="iconify-head">
              <Sparkles :size="13" />
              <span>或从 Iconify 在线图标库搜索，选中后保存为内联 SVG（离线可用）</span>
            </div>
            <div class="upload-row">
              <input
                v-model="iconifyQuery"
                class="field"
                placeholder="例如 home、server、nas、music"
                @keyup.enter="searchIconify"
              />
              <button type="button" class="btn" :disabled="iconifyLoading" @click="searchIconify">
                {{ iconifyLoading ? '搜索中…' : '搜索图标' }}
              </button>
            </div>
            <div v-if="iconifyResults.length" class="iconify-grid">
              <button
                v-for="name in iconifyResults"
                :key="name"
                type="button"
                class="iconify-cell"
                :title="name"
                @click="pickIconify(name)"
              >
                <span v-if="iconifySvgs[name]" v-html="iconifySvgs[name]" />
                <span v-else class="iconify-spinner" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <button class="btn ghost" type="button" @click="emit('update:show', false)">取消</button>
      <button class="btn primary" type="button" :disabled="saving" @click="save">
        {{ saving ? '保存中...' : isEdit ? '保存修改' : '创建卡片' }}
      </button>
    </template>
  </RoundCardModal>
</template>

<style scoped>
.editor {
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.preview-box {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  border-radius: 14px;
  border: 1px dashed var(--line);
  background: rgba(var(--glass-rgb), 0.38);
}

.preview-meta {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}
.preview-meta strong {
  font-size: 14px;
  color: var(--text-strong);
}
.preview-meta span {
  font-size: 12px;
  color: var(--text-soft);
}

.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.optional {
  font-weight: 400;
  color: var(--text-faint);
  font-size: 11px;
  margin-left: 4px;
}

.input-icon {
  position: relative;
  display: flex;
  align-items: center;
}
.input-icon > svg {
  position: absolute;
  left: 11px;
  color: var(--text-faint);
  pointer-events: none;
}
.field.with-icon {
  padding-left: 32px;
}

.seg {
  display: flex;
  padding: 3px;
  border-radius: 10px;
  background: rgba(var(--glass-rgb), 0.45);
  border: 1px solid var(--line);
}
.seg button {
  flex: 1;
  border: none;
  background: transparent;
  padding: 6px;
  font-size: 12px;
  color: var(--text-soft);
  border-radius: 7px;
  cursor: pointer;
  transition: all 0.16s;
}
.seg button.active {
  background: rgba(var(--glass-rgb), 0.92);
  color: var(--text-strong);
  box-shadow: 0 2px 8px -3px rgba(var(--shadow-color), 0.25);
}

.type-tabs {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
  margin-bottom: 10px;
}
.type-tabs button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 7px;
  font-size: 12px;
  border-radius: 9px;
  border: 1px solid var(--line);
  background: transparent;
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.16s;
}
.type-tabs button.active {
  border-color: color-mix(in srgb, var(--accent) 55%, transparent);
  background: color-mix(in srgb, var(--accent) 10%, transparent);
  color: var(--accent-strong, #059669);
}

.type-panel {
  padding: 12px;
  border-radius: 12px;
  background: rgba(var(--glass-rgb), 0.34);
  border: 1px solid var(--line);
}

.swatches {
  display: grid;
  grid-template-columns: repeat(10, 1fr);
  gap: 6px;
}
.swatch {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1;
  border-radius: 8px;
  border: 2px solid transparent;
  color: #fff;
  cursor: pointer;
  transition: transform 0.15s, border-color 0.15s;
}
.swatch:hover {
  transform: translateY(-2px);
}
.swatch.active {
  border-color: var(--text-strong);
  box-shadow: 0 0 0 2px rgba(var(--glass-rgb), 0.7);
}

.upload-row {
  display: flex;
  gap: 6px;
}
.upload-row .field {
  flex: 1;
  min-width: 0;
}

.favicon-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 9px;
  padding-top: 9px;
  border-top: 1px dashed var(--line);
}
.favicon-hint {
  font-size: 10.5px;
  color: var(--text-faint);
  line-height: 1.5;
}
.favicon-row .btn {
  flex-shrink: 0;
}

.gallery-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 9px;
  padding-top: 9px;
  border-top: 1px dashed var(--line);
}
.gallery-row .btn {
  flex-shrink: 0;
}
.gallery-panel {
  margin-top: 10px;
  padding: 8px;
  border-radius: 10px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.34);
}
.gallery-empty {
  padding: 14px 8px;
  text-align: center;
  font-size: 11px;
  color: var(--text-faint);
}
.gallery-grid {
  display: grid;
  grid-template-columns: repeat(8, minmax(0, 1fr));
  gap: 6px;
  max-height: 200px;
  overflow-y: auto;
}
.gallery-cell {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1;
  padding: 5px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(var(--glass-rgb), 0.45);
  cursor: pointer;
  transition: all 0.15s;
}
.gallery-cell:hover {
  border-color: color-mix(in srgb, var(--accent) 45%, transparent);
  transform: translateY(-1px);
}
.gallery-cell img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.inline-setting {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 11px 13px;
  border-radius: 11px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.34);
}
.label.no-margin {
  margin: 0;
}
.setting-hint {
  margin-top: 2px;
  font-size: 10.5px;
  line-height: 1.5;
  color: var(--text-faint);
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

.iconify {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--line);
}
.iconify-head {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-bottom: 8px;
  font-size: 10.5px;
  line-height: 1.5;
  color: var(--text-faint);
}
.iconify-head svg {
  flex-shrink: 0;
  margin-top: 1px;
  color: var(--accent);
}
.iconify-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 5px;
  margin-top: 9px;
  max-height: 168px;
  overflow-y: auto;
}
.iconify-cell {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1;
  padding: 6px;
  border: 1px solid var(--line);
  border-radius: 9px;
  background: rgba(var(--glass-rgb), 0.45);
  color: var(--text-strong);
  cursor: pointer;
  transition: all 0.15s;
}
.iconify-cell:hover {
  border-color: color-mix(in srgb, var(--accent) 45%, transparent);
  background: color-mix(in srgb, var(--accent) 9%, rgba(var(--glass-rgb), 0.5));
  transform: translateY(-1px);
}
.iconify-cell :deep(svg) {
  width: 20px;
  height: 20px;
}
.iconify-spinner {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 1.5px solid color-mix(in srgb, var(--accent) 22%, transparent);
  border-top-color: var(--accent);
  animation: iconify-spin 0.8s linear infinite;
}
@keyframes iconify-spin {
  to {
    transform: rotate(360deg);
  }
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 34px;
  padding: 0 14px;
  border-radius: 9px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.6);
  color: var(--text-strong);
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.16s;
}
.btn:hover {
  border-color: color-mix(in srgb, var(--accent) 35%, transparent);
}
.btn.primary {
  background: var(--accent);
  border-color: transparent;
  color: #fff;
  box-shadow: 0 10px 24px -12px color-mix(in srgb, var(--accent) 80%, transparent);
}
.btn.primary:hover {
  filter: brightness(1.05);
}
.btn.ghost {
  background: transparent;
}
.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.mono {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 12px;
  resize: vertical;
}

.mt-2 {
  margin-top: 8px;
}

@media (max-width: 560px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
  .swatches {
    grid-template-columns: repeat(5, 1fr);
  }
  .iconify-grid {
    grid-template-columns: repeat(6, 1fr);
  }
  .gallery-grid {
    grid-template-columns: repeat(5, 1fr);
  }
}
</style>
