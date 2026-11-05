<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ExternalLink, RefreshCw } from 'lucide-vue-next'
import RoundCardModal from './RoundCardModal.vue'
import { usePanelStore } from '@/store/panel'
import { openExternal } from '@/utils/url'

const panel = usePanelStore()

const show = computed({
  get: () => !!panel.viewer,
  set: (value: boolean) => {
    if (!value) panel.closeViewer()
  },
})

const url = computed(() => panel.viewer?.url || '')
const title = computed(() => panel.viewer?.title || url.value)
const host = computed(() => {
  try {
    return new URL(url.value).host
  } catch {
    return url.value
  }
})

const loading = ref(false)
const frameKey = ref(0)

watch(url, () => {
  loading.value = !!url.value
  frameKey.value += 1
})

function reload() {
  loading.value = true
  frameKey.value += 1
}

function openInNewTab() {
  if (!url.value) return
  openExternal(url.value, true)
}
</script>

<template>
  <RoundCardModal
    v-model:show="show"
    :title="title"
    :width="1120"
    :height="760"
    :min-width="420"
    :min-height="360"
    body-class="webview-body"
  >
    <template #title>
      <span class="wv-title">{{ title }}</span>
      <span class="wv-host">{{ host }}</span>
    </template>

    <div class="webview">
      <div class="wv-bar">
        <span class="wv-dots" aria-hidden="true"><i /><i /><i /></span>
        <span class="wv-url" :title="url">{{ url }}</span>
        <div class="wv-actions">
          <button class="wv-btn" type="button" title="重新加载" @click="reload">
            <RefreshCw :size="14" />
          </button>
          <button class="wv-btn" type="button" title="在新标签页打开" @click="openInNewTab">
            <ExternalLink :size="14" />
          </button>
        </div>
      </div>

      <div class="wv-stage">
        <div v-if="loading" class="wv-loading"><span class="wv-spinner" />正在加载…</div>
        <iframe
          :key="frameKey"
          class="wv-frame"
          :src="url"
          :title="title"
          referrerpolicy="no-referrer"
          allow="clipboard-read; clipboard-write; fullscreen"
          @load="loading = false"
        />
      </div>

      <footer class="wv-foot">
        <span class="wv-tip">部分网站（如 Google、B 站）会拒绝 iframe 嵌入。</span>
        <button class="wv-link" type="button" @click="openInNewTab">在新标签页打开</button>
      </footer>
    </div>
  </RoundCardModal>
</template>

<style scoped>
:deep(.webview-body) {
  display: flex;
  overflow: hidden;
  padding: 0;
}

.wv-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wv-host {
  font-size: 10.5px;
  font-weight: 400;
  color: var(--text-faint);
  padding: 1px 7px;
  border-radius: 999px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.45);
}

.webview {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.wv-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.38);
}

.wv-dots {
  display: inline-flex;
  gap: 4px;
  flex-shrink: 0;
}
.wv-dots i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--line);
}
.wv-dots i:first-child {
  background: color-mix(in srgb, var(--accent) 60%, transparent);
}

.wv-url {
  flex: 1;
  min-width: 0;
  font-size: 11px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: 'JetBrains Mono', Consolas, monospace;
}

.wv-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.wv-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 27px;
  height: 27px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: transparent;
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.wv-btn:hover {
  color: var(--accent-strong, #059669);
  border-color: color-mix(in srgb, var(--accent) 35%, transparent);
}

.wv-stage {
  position: relative;
  flex: 1;
  min-height: 0;
  background: #fff;
}

.wv-frame {
  width: 100%;
  height: 100%;
  border: none;
  display: block;
  background: #fff;
}

.wv-loading {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  font-size: 12px;
  color: var(--text-faint);
  background: rgba(var(--glass-rgb), 0.6);
}
.wv-spinner {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid color-mix(in srgb, var(--accent) 25%, transparent);
  border-top-color: var(--accent);
  animation: wv-spin 0.8s linear infinite;
}
@keyframes wv-spin {
  to {
    transform: rotate(360deg);
  }
}

.wv-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 7px 14px;
  flex-shrink: 0;
  border-top: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.32);
}
.wv-tip {
  font-size: 10.5px;
  color: var(--text-faint);
}
.wv-link {
  border: none;
  background: transparent;
  color: var(--accent-strong, #059669);
  font-size: 11.5px;
  cursor: pointer;
  white-space: nowrap;
}
.wv-link:hover {
  text-decoration: underline;
}
</style>
