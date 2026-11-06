<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ChevronDown, CornerDownLeft, Search } from 'lucide-vue-next'
import AppIcon from './AppIcon.vue'
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
const activeIndex = ref(-1)

// Bang prefixes: !g Google, !b Bing, !d DuckDuckGo, !gh GitHub, !yt YouTube.
// Unknown prefixes are treated as normal text so users can still search for "!foo".
const prefixAliases: Record<string, string[]> = {
  g: ['google', '谷歌'],
  b: ['baidu', '百度'],
  bi: ['bing', '必应'],
  d: ['duckduckgo'],
  gh: ['github'],
  yt: ['youtube'],
}

function engineForKey(key: string) {
  const aliases = prefixAliases[key] || [key]
  return engines.value.find((engine) => {
    const haystack = `${engine.name} ${engine.url}`.toLowerCase()
    return aliases.some((alias) => haystack.includes(alias))
  })
}

const prefix = computed(() => {
  const match = keyword.value.match(/^\s*!([a-zA-Z]{1,6})\s+([\s\S]+)$/)
  if (!match) return null
  const engine = engineForKey(match[1].toLowerCase())
  if (!engine) return null
  return { engine, query: match[2].trim() }
})

const effectiveKeyword = computed(() => prefix.value?.query || keyword.value)

const currentEngine = computed(() => prefix.value?.engine || engines.value[engineIndex.value] || engines.value[0])
const isDirect = computed(() => isUrlLike(effectiveKeyword.value))

const matches = computed(() => {
  const query = effectiveKeyword.value.trim().toLowerCase()
  if (!query) return []
  return panel.items
    .filter((item) =>
      `${item.title} ${item.description} ${item.url} ${item.lanUrl}`.toLowerCase().includes(query),
    )
    .slice(0, 8)
})

const filterVisible = computed(
  () =>
    !!config.searchFilterItems &&
    focused.value &&
    !engineOpen.value &&
    keyword.value.trim() !== '' &&
    matches.value.length > 0,
)

watch(keyword, () => {
  activeIndex.value = -1
})

function matchHost(url: string): string {
  try {
    return new URL(normalizeUrl(url)).host
  } catch {
    return url
  }
}

function openMatch(index: number) {
  const item = matches.value[index]
  if (!item) return
  panel.openItem(item)
  inputRef.value?.blur()
}

function pickEngine(index: number) {
  engineIndex.value = index
  localStorage.setItem('firefly_engine', String(index))
  engineOpen.value = false
  inputRef.value?.focus()
}

function submit() {
  const value = effectiveKeyword.value.trim()
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
    activeIndex.value = -1
    return
  }
  if (!filterVisible.value) {
    if (e.key === 'Enter') submit()
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIndex.value = (activeIndex.value + 1) % matches.value.length
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIndex.value = activeIndex.value <= 0 ? matches.value.length - 1 : activeIndex.value - 1
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (activeIndex.value >= 0) openMatch(activeIndex.value)
    else submit()
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
      @keydown="onKeydown"
    />

    <div v-if="keyword" class="hint">
      <span v-if="prefix" class="hint-prefix">!{{ prefix.engine.name }}</span>
      <span v-if="isDirect" class="hint-text">直接打开</span>
      <CornerDownLeft :size="13" class="hint-enter" />
    </div>
    <button v-else class="search-btn" type="button" @click="submit">
      <Search :size="17" />
    </button>

    <transition name="pop">
      <div v-if="filterVisible" class="match-menu" @mousedown.prevent>
        <div class="match-head">
          <span>面板中的项目</span>
          <span class="match-count">{{ matches.length }}</span>
        </div>
        <button
          v-for="(item, index) in matches"
          :key="item.id"
          class="match-item"
          :class="{ active: index === activeIndex }"
          type="button"
          @click="openMatch(index)"
          @mouseenter="activeIndex = index"
        >
          <AppIcon :icon="item.icon" :title="item.title" :size="27" :radius="8" />
          <span class="match-main">
            <strong>{{ item.title }}</strong>
            <em>{{ item.description || matchHost(item.url || item.lanUrl) }}</em>
          </span>
          <CornerDownLeft v-if="index === activeIndex" :size="13" class="match-enter" />
        </button>
      </div>
    </transition>

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
  z-index: 20;
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
.hint-prefix {
  padding: 2px 7px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--accent) 14%, transparent);
  color: var(--accent-strong, #059669);
  font-size: 10px;
  font-weight: 600;
  white-space: nowrap;
}

.match-menu {
  position: absolute;
  z-index: 38;
  top: calc(100% + 8px);
  left: 0;
  right: 0;
  max-height: 336px;
  overflow-y: auto;
  padding: 6px;
  border-radius: 14px;
  background: rgba(var(--glass-rgb), 0.92);
  border: 1px solid var(--glass-border);
  backdrop-filter: blur(28px) saturate(180%);
  -webkit-backdrop-filter: blur(28px) saturate(180%);
  box-shadow: 0 22px 60px -24px rgba(var(--shadow-color), 0.5);
}

.match-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 9px 7px;
  font-size: 10px;
  letter-spacing: 0.06em;
  color: var(--text-faint);
  border-bottom: 1px dashed var(--line);
  margin-bottom: 4px;
}
.match-count {
  padding: 0 6px;
  border-radius: 999px;
  border: 1px solid var(--line);
  background: rgba(var(--glass-rgb), 0.5);
}

.match-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 6px 8px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: var(--text-soft);
  text-align: left;
  cursor: pointer;
  transition: background 0.15s;
}
.match-item.active,
.match-item:hover {
  background: color-mix(in srgb, var(--accent) 9%, rgba(var(--glass-rgb), 0.6));
}

.match-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.match-main strong {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.match-main em {
  font-style: normal;
  font-size: 10.5px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.match-enter {
  flex-shrink: 0;
  color: var(--accent-strong, #059669);
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
