<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import RoundCardModal from './RoundCardModal.vue'

const props = defineProps<{
  show: boolean
  title: string
  value?: string
  placeholder?: string
  confirmText?: string
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'confirm', value: string): void
}>()

const text = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

watch(
  () => props.show,
  async (visible) => {
    if (!visible) return
    text.value = props.value || ''
    await nextTick()
    inputRef.value?.focus()
    inputRef.value?.select()
  },
)

function confirm() {
  const value = text.value.trim()
  if (!value) return
  emit('confirm', value)
  emit('update:show', false)
}
</script>

<template>
  <RoundCardModal
    :show="show"
    :title="title"
    :width="420"
    :height="230"
    :min-width="320"
    :min-height="210"
    :resizable="false"
    @update:show="emit('update:show', $event)"
  >
    <div class="prompt-body">
      <input
        ref="inputRef"
        v-model="text"
        class="field"
        :placeholder="placeholder || '请输入名称'"
        maxlength="32"
        @keyup.enter="confirm"
      />
    </div>
    <template #footer>
      <button class="p-btn ghost" type="button" @click="emit('update:show', false)">取消</button>
      <button class="p-btn primary" type="button" @click="confirm">{{ confirmText || '确定' }}</button>
    </template>
  </RoundCardModal>
</template>

<style scoped>
.prompt-body {
  padding: 22px 18px;
}
.p-btn {
  height: 34px;
  padding: 0 16px;
  border-radius: 9px;
  border: 1px solid var(--line);
  background: transparent;
  color: var(--text-strong);
  font-size: 12.5px;
  cursor: pointer;
}
.p-btn.primary {
  background: var(--accent);
  border-color: transparent;
  color: #fff;
}
.p-btn.ghost:hover {
  background: rgba(var(--glass-rgb), 0.5);
}
</style>
