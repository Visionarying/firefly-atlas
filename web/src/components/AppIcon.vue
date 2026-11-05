<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { IconConfig } from '@/typings'

const props = withDefaults(
  defineProps<{
    icon?: IconConfig
    title?: string
    size?: number
    radius?: number
  }>(),
  { icon: () => ({ itemType: 2 }), title: '', size: 44, radius: 12 },
)

const failed = ref(false)
watch(
  () => props.icon?.src,
  () => (failed.value = false),
)

const isImage = computed(() => props.icon?.itemType === 1 && !!props.icon.src && !failed.value)
const isSvg = computed(() => props.icon?.itemType === 3 && !!props.icon.src)
const text = computed(() => props.icon?.text || props.title?.slice(0, 1)?.toUpperCase() || '?')
const background = computed(
  () => props.icon?.backgroundColor || 'linear-gradient(135deg, #93c5fd, #6366f1)',
)
const style = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`,
  borderRadius: `${props.radius}px`,
  background: isImage.value || isSvg.value ? 'transparent' : background.value,
  fontSize: `${Math.max(12, props.size * (text.value.length > 1 ? 0.3 : 0.42))}px`,
}))
</script>

<template>
  <span class="app-icon" :style="style">
    <img
      v-if="isImage"
      :src="icon?.src"
      :alt="title"
      class="img"
      loading="lazy"
      @error="failed = true"
    />
    <span v-else-if="isSvg" class="svg" v-html="icon?.src" />
    <span v-else class="text">{{ text }}</span>
  </span>
</template>

<style scoped>
.app-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  overflow: hidden;
  color: #fff;
  font-weight: 700;
  letter-spacing: 0.02em;
  user-select: none;
  box-shadow: 0 8px 18px -10px rgba(23, 32, 51, 0.45);
}
.img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.svg {
  display: flex;
  width: 78%;
  height: 78%;
  align-items: center;
  justify-content: center;
}
.svg :deep(svg) {
  width: 100%;
  height: 100%;
}
.text {
  transform: translateY(0.02em);
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.15);
}
</style>
