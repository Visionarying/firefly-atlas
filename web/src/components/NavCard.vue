<script setup lang="ts">
import { computed } from 'vue'
import { Pencil, Trash2, GripVertical } from 'lucide-vue-next'
import AppIcon from './AppIcon.vue'
import { usePanelStore } from '@/store/panel'
import type { ItemIcon } from '@/typings'

const props = defineProps<{
  item: ItemIcon
  editMode?: boolean
}>()

const emit = defineEmits<{
  (e: 'open', newTab?: boolean): void
  (e: 'edit'): void
  (e: 'delete'): void
}>()

const panel = usePanelStore()
const config = panel.config

const iconStyle = computed(() => Number(config.iconStyle) || 0)

const cardStyle = computed(() => {
  if (iconStyle.value !== 0) return {}
  const alpha = Math.min(100, Math.max(0, config.cardGlassOpacity)) / 100
  if (alpha === 0) {
    return {
      background: 'transparent',
      borderColor: 'transparent',
      backdropFilter: 'none',
      boxShadow: 'none',
    }
  }
  const dark = panel.isDark
  const rgb = dark ? '28, 26, 52' : '255, 255, 255'
  return {
    background: `rgba(${rgb}, ${alpha})`,
    borderColor: dark
      ? `rgba(255, 255, 255, ${0.06 + alpha * 0.16})`
      : `rgba(255, 255, 255, ${0.4 + alpha * 0.5})`,
    backdropFilter: `blur(${Math.max(6, Math.round(alpha * 26))}px) saturate(165%)`,
    WebkitBackdropFilter: `blur(${Math.max(6, Math.round(alpha * 26))}px) saturate(165%)`,
    borderRadius: `${config.cardRadius}px`,
    boxShadow: `0 12px 32px -20px rgba(var(--shadow-color), ${0.18 + alpha * 0.4})`,
  }
})

function onAux(e: MouseEvent) {
  if (e.button === 1) {
    e.preventDefault()
    emit('open', true)
  }
}
</script>

<template>
  <article
    class="nav-card"
    :class="[`style-${iconStyle}`, { editing: editMode }]"
    :style="cardStyle"
    @click="emit('open')"
    @auxclick="onAux"
  >
    <button v-if="editMode" class="drag-handle" title="拖动排序" @click.stop>
      <GripVertical :size="14" />
    </button>

    <div v-if="editMode" class="edit-actions">
      <button class="mini-btn" title="编辑" @click.stop="emit('edit')">
        <Pencil :size="13" />
      </button>
      <button class="mini-btn danger" title="删除" @click.stop="emit('delete')">
        <Trash2 :size="13" />
      </button>
    </div>

    <AppIcon
      :icon="item.icon"
      :title="item.title"
      :size="iconStyle === 1 ? 58 : iconStyle === 2 ? 30 : 44"
      :radius="iconStyle === 1 ? 16 : iconStyle === 2 ? 9 : 12"
    />

    <div class="meta">
      <h3 class="title" :class="{ ellipsis: iconStyle !== 0 }">{{ item.title }}</h3>
      <p v-if="iconStyle === 0 && item.description" class="desc">{{ item.description }}</p>
      <span v-if="item.lanUrl" class="lan-dot" title="已配置内网地址" />
    </div>
  </article>
</template>

<style scoped>
.nav-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 13px 14px;
  border: 1px solid transparent;
  cursor: pointer;
  outline: none;
  transition:
    transform 0.22s cubic-bezier(0.22, 1, 0.36, 1),
    box-shadow 0.22s ease,
    border-color 0.22s ease,
    background 0.22s ease;
  -webkit-tap-highlight-color: transparent;
}

.nav-card::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.28), transparent 46%);
  opacity: 0;
  transition: opacity 0.22s ease;
  pointer-events: none;
}

.nav-card:hover {
  transform: translateY(-3px);
}

.nav-card:hover::after {
  opacity: 1;
}

.nav-card:hover .lan-dot {
  opacity: 1;
}

.nav-card.editing {
  cursor: default;
}

.meta {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-strong);
  letter-spacing: 0.01em;
  line-height: 1.3;
}

.ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.desc {
  margin: 0;
  font-size: 11.5px;
  color: var(--text-soft);
  line-height: 1.4;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  opacity: 0.82;
}

.lan-dot {
  position: absolute;
  right: 12px;
  bottom: 10px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--accent);
  opacity: 0.45;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 16%, transparent);
  transition: opacity 0.2s;
}

/* Large icon style */
.style-1 {
  flex-direction: column;
  gap: 10px;
  padding: 18px 10px;
  background: transparent !important;
  border-color: transparent !important;
  backdrop-filter: none !important;
  box-shadow: none !important;
}
.style-1:hover {
  transform: translateY(-5px);
}
.style-1 .meta {
  flex: none;
  align-items: center;
}
.style-1 .title {
  font-size: 13px;
  font-weight: 500;
  text-align: center;
  max-width: 100%;
  text-shadow: 0 2px 12px rgba(23, 32, 51, 0.18);
}

/* Small icon style */
.style-2 {
  gap: 9px;
  padding: 9px 11px;
}
.style-2 .title {
  font-size: 13px;
}

.drag-handle {
  position: absolute;
  top: 6px;
  left: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 7px;
  background: rgba(var(--glass-rgb), 0.7);
  color: var(--text-faint);
  cursor: grab;
  opacity: 0;
  transition: opacity 0.18s;
  z-index: 2;
}
.drag-handle:active {
  cursor: grabbing;
}
.nav-card:hover .drag-handle {
  opacity: 1;
}

.edit-actions {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.18s;
  z-index: 2;
}
.nav-card:hover .edit-actions {
  opacity: 1;
}

.mini-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid var(--glass-border);
  border-radius: 7px;
  background: rgba(var(--glass-rgb), 0.78);
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.mini-btn:hover {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 40%, transparent);
}
.mini-btn.danger:hover {
  color: #e5484d;
  border-color: rgba(229, 72, 77, 0.4);
}

@media (max-width: 560px) {
  .nav-card {
    padding: 11px 12px;
  }
  .style-1 {
    padding: 14px 6px;
  }
}
</style>
