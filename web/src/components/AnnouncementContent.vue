<script setup lang="ts">
import { computed } from 'vue'
import DOMPurify from 'dompurify'

const props = withDefaults(defineProps<{
  html?: string
  emptyText?: string
}>(), {
  html: '',
  emptyText: '暂无内容',
})

const sanitizedHtml = computed(() => {
  const raw = props.html?.trim() || ''
  if (!raw) return ''
  return DOMPurify.sanitize(raw, {
    USE_PROFILES: { html: true },
    FORBID_TAGS: ['script', 'style', 'iframe'],
  })
})
</script>

<template>
  <div v-if="sanitizedHtml" class="announcement-content" v-html="sanitizedHtml"></div>
  <el-empty v-else :description="emptyText" :image-size="80" />
</template>

<style scoped>
.announcement-content {
  color: var(--el-text-color-regular);
  line-height: 1.75;
  word-break: break-word;
}
.announcement-content :deep(h1),
.announcement-content :deep(h2),
.announcement-content :deep(h3),
.announcement-content :deep(h4) {
  margin: 0 0 12px;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}
.announcement-content :deep(p),
.announcement-content :deep(ul),
.announcement-content :deep(ol),
.announcement-content :deep(blockquote),
.announcement-content :deep(pre) {
  margin: 0 0 12px;
}
.announcement-content :deep(ul),
.announcement-content :deep(ol) {
  padding-left: 20px;
}
.announcement-content :deep(a) {
  color: var(--el-color-primary);
  text-decoration: none;
}
.announcement-content :deep(a:hover) {
  text-decoration: underline;
}
.announcement-content :deep(code) {
  padding: 2px 6px;
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
.announcement-content :deep(pre) {
  overflow-x: auto;
  padding: 12px;
  border-radius: 12px;
  background: var(--el-fill-color-lighter);
}
.announcement-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 12px;
}
.announcement-content :deep(blockquote) {
  padding: 10px 14px;
  border-left: 4px solid var(--el-color-primary-light-5);
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-secondary);
}
</style>