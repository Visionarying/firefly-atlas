<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ChevronDown, Search, CornerDownLeft } from 'lucide-vue-next'
import { usePanelStore } from '@/store/panel'
import { isUrlLike, normalizeUrl, openExternal } from '@/utils/url'

const panel = usePanelStore()
const config = panel.config
const engines = computed(() => panel.searchEngines)

const keyword = ref('')
const inputRef = ref<HTMLInputElement | null>(null)
const engineIndex = ref(Number(localStorage.getItem('firefly_engine') || 0))
const engineOpen = ref(false)
const focused = ref(false)

const currentEngine = computed(() => engines.value[engineIndex.value] || engines.value[0])
const isDirect = computed(() => isUrlLike(keyword.value))

function pickEngine(index: number) {
  engineIndex.value = index
  localStorage.setItem('firefly_engine', String(index))
  engineOpen.value = false
  inputRef.value?.focus()
}

function submit() {
  const value = keyword.value.trim()
  if (!value) return
  if (isUrlLike(value)) {
    openExternal(normalizeUrl(value), config.searchOpenInNewTab)
  } else if (currentEngine.value) {
    openExternal(currentEngine.value.url + encodeURIComponent(value), config.searchOpenInNewTab)
  }
}

function onFocus() {
  focused.value = true
}
function onBlur() {
  focused.value = false
  window.setTimeout(() => (engineOpen.value = false), 120)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    inputRef.value?.blur()
  }
}

function onGlobalKey(e: KeyboardEvent) {
  if (e.key === '/' && document.activeElement?.tagName !== 'INPUT' && document.activeElement?.tagName !== 'TEXTAREA') {
    e.preventDefault()
    inputRef.value?.focus()
  }
}

onMounted(() => window.addEventListener('keydown', onGlobalKey))
onUnmounted(() => window.removeEventListener('keydown', onGlobalKey))
</script>

<template>
  <div class="search-wrap" :class="{ focused }">
    <div class="engine" @mousedown.prevent="engineOpen = !engineOpen">
      <span class="engine-badge">{{ currentEngine?.name?.slice(0, 1) || 'S' }}</span>
      <span class="engine-name">{{ currentEngine?.name || '搜索' }}</span>
      <ChevronDown :size="14" class="chev" :class="{ open: engineOpen }" />
    </div>

    <div class="divider" />

    <input
      ref="inputRef"
      v-model="keyword"
      class="search-input"
      type="text"
      :placeholder="config.searchPlaceholder"
      autocomplete="off"
      spellcheck="false"
      @focus="onFocus"
      @blur="onBlur"
      @keydown.enter="submit"
      @keydown="onKeydown"
    />

    <div v-if="keyword" class="hint">
      <span v-if="isDirect" class="hint-text">直接打开</span>
      <CornerDownLeft :size="13" class="hint-enter" />
    </div>
    <button v-else class="search-btn" type="button" @click="submit">
      <Search :size="17" />
    </button>

    <transition name="pop">
      <div v-if="engineOpen" class="engine-menu" @mousedown.prevent>
        <button
          v-for="(engine, index) in engines"
          :key="engine.name + index"
          class="engine-option"
          :class="{ active: index === engineIndex }"
          type="button"
          @click="pickEngine(index)"
        >
          <span class="engine-badge small">{{ engine.name.slice(0, 1) }}</span>
          <span>{{ engine.name }}</span>
        </button>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.search-wrap {
  position: relative;
  display: flex;
  align-items: center;
  width: min(620px, 100%);
  height: 54px;
  margin: 26px auto 0;
  padding: 0 8px 0 10px;
  border-radius: 16px;
  background: rgba(var(--glass-rgb), 0.55);
  border: 1px solid var(--glass-border);
  backdrop-filter: blur(24px) saturate(170%);
  -webkit-backdrop-filter: blur(24px) saturate(170%);
  box-shadow: 0 16px 44px -26px rgba(var(--shadow-color), 0.55);
  transition: box-shadow 0.25s ease, border-color 0.25s ease, background 0.25s ease;
}

.search-wrap.focused {
  border-color: color-mix(in srgb, var(--accent) 45%, transparent);
  box-shadow:
    0 16px 44px -24px rgba(var(--shadow-color), 0.55),
    0 0 0 4px color-mix(in srgb, var(--accent) 12%, transparent);
}

.engine {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 10px;
  cursor: pointer;
  user-select: none;
  transition: background 0.18s;
  flex-shrink: 0;
}
.engine:hover {
  background: rgba(var(--glass-rgb), 0.45);
}

.engine-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 7px;
  background: color-mix(in srgb, var(--accent) 16%, transparent);
  color: var(--accent-strong, #059669);
  font-size: 11px;
  font-weight: 700;
}
.engine-badge.small {
  width: 20px;
  height: 20px;
  border-radius: 6px;
}

.engine-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-soft);
  max-width: 72px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chev {
  color: var(--text-faint);
  transition: transform 0.2s;
}
.chev.open {
  transform: rotate(180deg);
}

.divider {
  width: 1px;
  height: 22px;
  background: var(--line);
  margin: 0 6px;
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  font-size: 14px;
  color: var(--text-strong);
}
.search-input::placeholder {
  color: var(--text-faint);
}

.search-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border: none;
  border-radius: 11px;
  background: color-mix(in srgb, var(--accent) 14%, transparent);
  color: var(--accent-strong, #059669);
  cursor: pointer;
  transition: all 0.18s;
}
.search-btn:hover {
  background: color-mix(in srgb, var(--accent) 24%, transparent);
  transform: translateY(-1px);
}

.hint {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  color: var(--text-faint);
  font-size: 11px;
  flex-shrink: 0;
}
.hint-text {
  letter-spacing: 0.04em;
}

.engine-menu {
  position: absolute;
  z-index: 40;
  top: calc(100% + 8px);
  left: 8px;
  width: 190px;
  padding: 6px;
  border-radius: 14px;
  background: rgba(var(--glass-rgb), 0.92);
  border: 1px solid var(--glass-border);
  backdrop-filter: blur(28px) saturate(180%);
  box-shadow: 0 22px 60px -24px rgba(var(--shadow-color), 0.5);
}

.engine-option {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: var(--text-soft);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.engine-option:hover {
  background: rgba(var(--glass-rgb), 0.6);
  color: var(--text-strong);
}
.engine-option.active {
  color: var(--accent-strong, #059669);
  background: color-mix(in srgb, var(--accent) 10%, transparent);
}

.pop-enter-active,
.pop-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}
.pop-enter-from,
.pop-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}

@media (max-width: 560px) {
  .search-wrap {
    height: 50px;
    margin-top: 20px;
  }
  .engine-name {
    display: none;
  }
  .hint-text {
    display: none;
  }
}
</style>
