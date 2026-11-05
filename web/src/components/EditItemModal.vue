<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { Image, Type, Code2, Upload, Link2, Check } from 'lucide-vue-next'
import RoundCardModal from './RoundCardModal.vue'
import AppIcon from './AppIcon.vue'
import { panelApi, fileApi } from '@/api'
import { usePanelStore } from '@/store/panel'
import type { IconConfig, ItemGroup, ItemIcon } from '@/typings'

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
    form.itemIconGroupId = item?.itemIconGroupId || props.defaultGroupId || props.groups[0]?.id || 0
    form.icon = {
      itemType: item?.icon?.itemType || 2,
      src: item?.icon?.src || '',
      text: item?.icon?.text || '',
      backgroundColor: item?.icon?.backgroundColor || gradients[0],
    }
    activeType.value = form.icon.itemType
    imageUrl.value = form.icon.itemType === 1 ? form.icon.src || '' : ''
  },
)

const isEdit = computed(() => form.id > 0)

function switchType(type: 1 | 2 | 3) {
  activeType.value = type
  form.icon.itemType = type
  if (type === 1 && form.icon.src && !foldersSafe(form.icon.src)) {
    imageUrl.value = form.icon.src
  }
}
function foldersSafe(value: string) {
  return value.startsWith('/uploads/')
}

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
              {{ group.title }}
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
        </div>

        <div v-else class="type-panel">
          <textarea
            v-model="form.icon.src"
            class="field mono"
            rows="5"
            placeholder='<svg viewBox="0 0 24 24">...</svg>'
          />
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
}
</style>
