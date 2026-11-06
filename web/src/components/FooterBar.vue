<script setup lang="ts">
import { computed } from 'vue'
import { usePanelStore } from '@/store/panel'

const panel = usePanelStore()
const config = panel.config

const html = computed(
  () => config.footerHtml || '<span>Firefly Atlas · 私有云导航</span>',
)
const style = computed(() => ({
  transform: `translateY(${config.footerOffsetY}px)`,
  opacity: Math.max(0, config.headerOpacity) / 100,
}))
</script>

<template>
  <footer v-if="config.showFooter" class="footer" :style="style">
    <div class="footer-inner" v-html="html" />
  </footer>
</template>

<style scoped>
.footer {
  margin-top: auto;
  padding: 46px 8px 22px;
  display: flex;
  justify-content: center;
  transition: transform 0.2s ease;
}

.footer-inner {
  font-size: 12px;
  letter-spacing: 0.06em;
  color: var(--text-soft);
  text-align: center;
  line-height: 1.8;
  text-shadow: 0 2px 14px rgba(var(--shadow-color), 0.12);
}

.footer-inner :deep(a) {
  color: inherit;
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: color 0.2s, border-color 0.2s, text-shadow 0.2s;
}

.footer-inner :deep(a:hover) {
  color: var(--accent);
  border-color: color-mix(in srgb, var(--accent) 55%, transparent);
  text-shadow: 0 0 16px color-mix(in srgb, var(--accent) 55%, transparent);
}
</style>
