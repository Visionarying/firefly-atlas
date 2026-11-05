<script setup lang="ts">
import { ref, watch } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { Plus, Pencil, Trash2 } from 'lucide-vue-next'
import NavCard from './NavCard.vue'
import { usePanelStore } from '@/store/panel'
import type { ItemGroup, ItemIcon } from '@/typings'

const props = defineProps<{
  group: ItemGroup
  items: ItemIcon[]
  editMode?: boolean
}>()

const emit = defineEmits<{
  (e: 'open', item: ItemIcon, newTab?: boolean): void
  (e: 'edit', item: ItemIcon): void
  (e: 'delete', item: ItemIcon): void
  (e: 'add', group: ItemGroup): void
  (e: 'rename', group: ItemGroup): void
  (e: 'remove', group: ItemGroup): void
  (e: 'sort', ids: number[]): void
}>()

const panel = usePanelStore()
const config = panel.config
const localItems = ref<ItemIcon[]>([])

watch(
  () => props.items,
  (value) => (localItems.value = [...value]),
  { immediate: true, deep: false },
)

const styleClass = () => `grid-${Number(config.iconStyle) || 0}`

function onEnd() {
  emit('sort', localItems.value.map((item) => item.id))
}
</script>

<template>
  <section class="group fade-up">
    <header class="group-head">
      <div class="group-title">
        <span class="dot" />
        <h2>{{ group.title }}</h2>
        <span class="count">{{ items.length }}</span>
      </div>
      <div v-if="editMode" class="group-actions">
        <button class="ghost-btn" title="新增卡片" @click="emit('add', group)">
          <Plus :size="14" />
        </button>
        <button class="ghost-btn" title="重命名分组" @click="emit('rename', group)">
          <Pencil :size="13" />
        </button>
        <button class="ghost-btn danger" title="删除分组" @click="emit('remove', group)">
          <Trash2 :size="13" />
        </button>
      </div>
    </header>

    <div v-if="items.length === 0 && !editMode" class="empty">这个分组还没有卡片</div>

    <VueDraggable
      v-else
      v-model="localItems"
      class="cards-grid"
      :class="styleClass()"
      :group="`nav-${group.id}`"
      :animation="180"
      :disabled="!editMode"
      handle=".drag-handle"
      ghost-class="drag-ghost"
      @end="onEnd"
    >
      <NavCard
        v-for="item in localItems"
        :key="item.id"
        :item="item"
        :edit-mode="editMode"
        @open="(newTab?: boolean) => emit('open', item, newTab)"
        @edit="emit('edit', item)"
        @delete="emit('delete', item)"
      />
    </VueDraggable>
  </section>
</template>

<style scoped>
.group {
  margin-top: 34px;
}

.group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
  padding: 0 4px;
}

.group-title {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--accent) 12%, transparent);
}

.group-title h2 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--text-strong);
  opacity: 0.92;
}

.count {
  font-size: 11px;
  color: var(--text-faint);
  background: rgba(var(--glass-rgb), 0.5);
  border: 1px solid var(--line);
  border-radius: 999px;
  padding: 1px 7px;
  line-height: 1.5;
}

.group-actions {
  display: flex;
  gap: 4px;
  opacity: 0;
  transform: translateX(4px);
  transition: opacity 0.2s, transform 0.2s;
}
.group:hover .group-actions {
  opacity: 1;
  transform: translateX(0);
}

.ghost-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(var(--glass-rgb), 0.5);
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.ghost-btn:hover {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 35%, transparent);
  background: rgba(var(--glass-rgb), 0.8);
}
.ghost-btn.danger:hover {
  color: #e5484d;
  border-color: rgba(229, 72, 77, 0.35);
}

.cards-grid {
  display: grid;
  gap: var(--item-gap, 16px);
}

.grid-0 {
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
}

.grid-1 {
  grid-template-columns: repeat(auto-fill, minmax(104px, 1fr));
  gap: calc(var(--item-gap, 16px) + 6px);
}

.grid-2 {
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
}

.empty {
  padding: 26px;
  text-align: center;
  font-size: 12px;
  color: var(--text-faint);
  border: 1px dashed var(--line);
  border-radius: 14px;
}

.drag-ghost {
  opacity: 0.35;
  transform: scale(0.98);
}

@media (max-width: 560px) {
  .group {
    margin-top: 26px;
  }
  .grid-0 {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 10px;
  }
  .grid-1 {
    grid-template-columns: repeat(auto-fill, minmax(84px, 1fr));
  }
  .grid-2 {
    grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
    gap: 10px;
  }
  .group-actions {
    opacity: 1;
    transform: none;
  }
}
</style>
