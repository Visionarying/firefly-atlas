<script setup lang="ts">
import { computed } from 'vue'
import { usePanelStore } from '@/store/panel'

const panel = usePanelStore()
const config = panel.config

const hasWallpaper = computed(() => !!config.backgroundImage)
const wallpaperStyle = computed(() => ({
  backgroundImage: `url("${config.backgroundImage}")`,
  filter: `blur(${config.backgroundBlur}px)`,
  transform: config.backgroundBlur > 0 ? 'scale(1.06)' : 'scale(1)',
}))
const maskStyle = computed(() => ({
  opacity: Math.min(100, Math.max(0, config.backgroundMask)) / 100,
}))
</script>

<template>
  <div class="bg-root" aria-hidden="true">
    <template v-if="hasWallpaper">
      <div class="wallpaper" :style="wallpaperStyle" />
      <div class="wallpaper-mask" :style="maskStyle" />
    </template>
    <template v-else>
      <div class="base-gradient" />
      <div class="blob blob-a" />
      <div class="blob blob-b" />
      <div class="blob blob-c" />
    </template>
    <div class="grain" />
  </div>
</template>

<style scoped>
.bg-root {
  position: fixed;
  inset: 0;
  z-index: -1;
  overflow: hidden;
  background: var(--panel-bg);
}

.wallpaper {
  position: absolute;
  inset: -24px;
  background-position: center;
  background-size: cover;
  transition: filter 0.4s ease;
}

.wallpaper-mask {
  position: absolute;
  inset: 0;
  background: var(--panel-bg);
  transition: opacity 0.3s ease;
}

.base-gradient {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(1200px 800px at 12% 8%, rgba(255, 255, 255, 0.9), transparent 60%),
    linear-gradient(150deg, #eef3fa 0%, #f7f4fb 42%, #eefaf6 100%);
}

.blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(90px);
  opacity: 0.62;
  will-change: transform;
}

.blob-a {
  width: 46vw;
  height: 46vw;
  left: -8vw;
  top: -10vw;
  background: radial-gradient(circle at 30% 30%, #a7c7ff, #c4b5fd 60%, transparent 72%);
  animation: floatSlow 18s ease-in-out infinite;
}

.blob-b {
  width: 38vw;
  height: 38vw;
  right: -6vw;
  top: 6vh;
  background: radial-gradient(circle at 60% 40%, #b8f5df, #a5e8ff 55%, transparent 72%);
  animation: floatSlow 22s ease-in-out infinite reverse;
}

.blob-c {
  width: 34vw;
  height: 34vw;
  right: 14vw;
  bottom: -14vw;
  background: radial-gradient(circle at 50% 50%, #ffd9ec, #ffe7c2 55%, transparent 72%);
  animation: floatSlow 26s ease-in-out infinite;
  opacity: 0.45;
}

.grain {
  position: absolute;
  inset: 0;
  opacity: 0.035;
  pointer-events: none;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 220 220' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
}
</style>
