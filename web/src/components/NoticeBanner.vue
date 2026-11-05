<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Megaphone, X } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    title?: string
    content: string
    /** When false the banner is shown on every page load. */
    remember?: boolean
  }>(),
  { title: '公告', content: '', remember: true },
)

function fingerprint(value: string): string {
  let hash = 0
  for (let i = 0; i < value.length; i += 1) {
    hash = (hash * 31 + value.charCodeAt(i)) | 0
  }
  return Math.abs(hash).toString(36)
}

const storageKey = computed(
  () => `firefly_notice_${fingerprint(`${props.title}|${props.content}`)}`,
)

const dismissed = ref(
  props.remember && !!props.content && localStorage.getItem(storageKey.value) === '1',
)

watch(storageKey, () => {
  dismissed.value = props.remember && localStorage.getItem(storageKey.value) === '1'
})

function close() {
  dismissed.value = true
  if (props.remember) localStorage.setItem(storageKey.value, '1')
}
</script>

<template>
  <div v-if="content && !dismissed" class="notice" role="status">
    <span class="notice-icon"><Megaphone :size="15" /></span>
    <div class="notice-body">
      <strong v-if="title">{{ title }}</strong>
      <p>{{ content }}</p>
    </div>
    <button class="notice-close" type="button" title="关闭公告" @click="close">
      <X :size="14" />
    </button>
  </div>
</template>

<style scoped>
.notice {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  width: min(720px, 100%);
  margin: 0 auto 4px;
  padding: 11px 12px 11px 14px;
  border-radius: 13px;
  border: 1px solid color-mix(in srgb, var(--accent) 22%, var(--line));
  background: color-mix(in srgb, var(--accent) 7%, rgba(var(--glass-rgb), 0.5));
  backdrop-filter: blur(18px) saturate(160%);
  -webkit-backdrop-filter: blur(18px) saturate(160%);
  animation: notice-in 0.45s cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes notice-in {
  from {
    opacity: 0;
    transform: translateY(-6px);
  }
}

.notice-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  border-radius: 8px;
  color: var(--accent-strong, #059669);
  background: color-mix(in srgb, var(--accent) 14%, transparent);
}

.notice-body {
  flex: 1;
  min-width: 0;
}
.notice-body strong {
  display: block;
  font-size: 12px;
  color: var(--text-strong);
  margin-bottom: 2px;
}
.notice-body p {
  margin: 0;
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--text-soft);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.notice-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--text-faint);
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}
.notice-close:hover {
  color: var(--text-strong);
  background: rgba(var(--glass-rgb), 0.6);
}
</style>
