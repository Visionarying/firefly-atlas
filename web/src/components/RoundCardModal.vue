<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { Maximize2, Minimize2, X } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    show: boolean
    title?: string
    width?: number
    height?: number
    minWidth?: number
    minHeight?: number
    closeOnMask?: boolean
    resizable?: boolean
    bodyClass?: string
  }>(),
  {
    title: '',
    width: 860,
    height: 620,
    minWidth: 340,
    minHeight: 320,
    closeOnMask: true,
    resizable: true,
    bodyClass: '',
  },
)

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'close'): void
}>()

const rect = reactive({ x: 0, y: 0, w: 860, h: 620 })
const prevRect = ref<{ x: number; y: number; w: number; h: number } | null>(null)
const maximized = ref(false)
const dragging = ref(false)
const resizing = ref(false)

const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max)

function initRect() {
  const vw = window.innerWidth
  const vh = window.innerHeight
  const mobile = vw < 680
  const w = mobile ? vw - 16 : Math.min(props.width, vw - 40)
  const h = mobile ? vh - 16 : Math.min(props.height, vh - 40)
  rect.w = Math.max(props.minWidth, w)
  rect.h = Math.max(props.minHeight, h)
  rect.x = Math.max(8, (vw - rect.w) / 2)
  rect.y = Math.max(8, (vh - rect.h) / 2)
}

function close() {
  emit('update:show', false)
  emit('close')
}

function onMask() {
  if (props.closeOnMask) close()
}

function startDrag(e: PointerEvent) {
  if (e.button !== 0) return
  const target = e.target as HTMLElement
  if (target.closest('button, input, textarea, select, a, [data-no-drag]')) return
  if (maximized.value) return
  e.preventDefault()
  dragging.value = true
  const sx = e.clientX
  const sy = e.clientY
  const ox = rect.x
  const oy = rect.y

  const move = (ev: PointerEvent) => {
    rect.x = clamp(ox + ev.clientX - sx, 4, window.innerWidth - rect.w - 4)
    rect.y = clamp(oy + ev.clientY - sy, 4, window.innerHeight - rect.h - 4)
  }
  const up = () => {
    dragging.value = false
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
}

const handles = ['n', 's', 'e', 'w', 'nw', 'ne', 'sw', 'se'] as const

function startResize(e: PointerEvent, dir: string) {
  if (e.button !== 0 || !props.resizable) return
  e.preventDefault()
  e.stopPropagation()
  if (maximized.value) return
  resizing.value = true
  const start = { ...rect }
  const sx = e.clientX
  const sy = e.clientY

  const move = (ev: PointerEvent) => {
    const dx = ev.clientX - sx
    const dy = ev.clientY - sy
    let { x, y, w, h } = start
    if (dir.includes('e')) {
      w = clamp(start.w + dx, props.minWidth, window.innerWidth - start.x - 4)
    }
    if (dir.includes('s')) {
      h = clamp(start.h + dy, props.minHeight, window.innerHeight - start.y - 4)
    }
    if (dir.includes('w')) {
      w = clamp(start.w - dx, props.minWidth, start.x + start.w - 4)
      x = start.x + start.w - w
    }
    if (dir.includes('n')) {
      h = clamp(start.h - dy, props.minHeight, start.y + start.h - 4)
      y = start.y + start.h - h
    }
    rect.x = x
    rect.y = y
    rect.w = w
    rect.h = h
  }
  const up = () => {
    resizing.value = false
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
}

function toggleMaximize() {
  if (!maximized.value) {
    prevRect.value = { ...rect }
    maximized.value = true
    rect.x = 6
    rect.y = 6
    rect.w = window.innerWidth - 12
    rect.h = window.innerHeight - 12
  } else {
    maximized.value = false
    if (prevRect.value) Object.assign(rect, prevRect.value)
  }
}

const panelStyle = computed(() => ({
  left: `${rect.x}px`,
  top: `${rect.y}px`,
  width: `${rect.w}px`,
  height: `${rect.h}px`,
}))

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.show) close()
}

function onWindowResize() {
  rect.x = clamp(rect.x, 4, Math.max(4, window.innerWidth - rect.w - 4))
  rect.y = clamp(rect.y, 4, Math.max(4, window.innerHeight - rect.h - 4))
}

watch(
  () => props.show,
  (value) => {
    if (value) {
      initRect()
      maximized.value = false
    }
  },
)

onMounted(() => {
  window.addEventListener('keydown', onKey)
  window.addEventListener('resize', onWindowResize)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('resize', onWindowResize)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="rcm">
      <div v-if="show" class="rcm-mask" @pointerdown.self="onMask">
        <div
          class="rcm-panel"
          :class="{ dragging, resizing, maximized }"
          :style="panelStyle"
          role="dialog"
          aria-modal="true"
        >
          <header class="rcm-head" @pointerdown="startDrag" @dblclick="toggleMaximize">
            <div class="rcm-title">
              <slot name="title">
                <span>{{ title }}</span>
              </slot>
            </div>
            <div class="rcm-tools">
              <button class="rcm-btn" type="button" :title="maximized ? '还原' : '最大化'" @click="toggleMaximize">
                <Minimize2 v-if="maximized" :size="14" />
                <Maximize2 v-else :size="14" />
              </button>
              <button class="rcm-btn close" type="button" title="关闭" @click="close">
                <X :size="15" />
              </button>
            </div>
          </header>

          <div class="rcm-body" :class="bodyClass">
            <slot />
          </div>

          <footer v-if="$slots.footer" class="rcm-foot">
            <slot name="footer" />
          </footer>

          <template v-if="resizable">
            <span
              v-for="dir in handles"
              :key="dir"
              class="rcm-handle"
              :class="`h-${dir}`"
              @pointerdown="startResize($event, dir)"
            />
          </template>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.rcm-mask {
  position: fixed;
  inset: 0;
  z-index: 100;
  background: rgba(16, 22, 34, 0.32);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}

.rcm-panel {
  position: fixed;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 18px;
  background: rgba(var(--glass-rgb), 0.88);
  border: 1px solid var(--glass-border);
  backdrop-filter: blur(30px) saturate(180%);
  -webkit-backdrop-filter: blur(30px) saturate(180%);
  box-shadow: 0 40px 100px -30px rgba(10, 16, 28, 0.55);
  transition: box-shadow 0.2s ease;
}

.rcm-panel.dragging,
.rcm-panel.resizing {
  user-select: none;
  box-shadow: 0 50px 120px -30px rgba(10, 16, 28, 0.6);
}

.rcm-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  height: 52px;
  padding: 0 12px 0 18px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--line);
  cursor: grab;
}
.rcm-panel.maximized .rcm-head {
  cursor: default;
}
.rcm-head:active {
  cursor: grabbing;
}

.rcm-title {
  display: flex;
  align-items: center;
  gap: 9px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-strong);
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.rcm-tools {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.rcm-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.rcm-btn:hover {
  background: rgba(var(--glass-rgb), 0.7);
  color: var(--text-strong);
}
.rcm-btn.close:hover {
  background: rgba(229, 72, 77, 0.14);
  color: #e5484d;
}

.rcm-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
}

.rcm-foot {
  flex-shrink: 0;
  padding: 12px 18px;
  border-top: 1px solid var(--line);
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.rcm-handle {
  position: absolute;
  z-index: 4;
}
.h-n,
.h-s {
  left: 12px;
  right: 12px;
  height: 8px;
  cursor: ns-resize;
}
.h-n {
  top: -2px;
}
.h-s {
  bottom: -2px;
}
.h-e,
.h-w {
  top: 12px;
  bottom: 12px;
  width: 8px;
  cursor: ew-resize;
}
.h-e {
  right: -2px;
}
.h-w {
  left: -2px;
}
.h-nw,
.h-ne,
.h-sw,
.h-se {
  width: 14px;
  height: 14px;
  z-index: 5;
}
.h-nw {
  top: -3px;
  left: -3px;
  cursor: nwse-resize;
}
.h-ne {
  top: -3px;
  right: -3px;
  cursor: nesw-resize;
}
.h-sw {
  bottom: -3px;
  left: -3px;
  cursor: nesw-resize;
}
.h-se {
  bottom: -3px;
  right: -3px;
  cursor: nwse-resize;
}

.rcm-enter-active,
.rcm-leave-active {
  transition: opacity 0.2s ease;
}
.rcm-enter-active .rcm-panel,
.rcm-leave-active .rcm-panel {
  transition: transform 0.24s cubic-bezier(0.22, 1, 0.36, 1), opacity 0.2s ease;
}
.rcm-enter-from,
.rcm-leave-to {
  opacity: 0;
}
.rcm-enter-from .rcm-panel,
.rcm-leave-to .rcm-panel {
  transform: translateY(10px) scale(0.985);
  opacity: 0;
}

@media (max-width: 680px) {
  .rcm-panel {
    border-radius: 16px;
  }
}
</style>
