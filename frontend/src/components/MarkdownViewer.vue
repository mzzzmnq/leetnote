<script setup lang="ts">
import { computed } from 'vue'
import { renderMarkdown } from '@/utils/markdown'

const props = defineProps<{
  source: string | null | undefined
}>()

// 渲染结果已在 utils/markdown.ts 里做过处理：
// markdown-it 关闭了原始 HTML，highlight.js 的输出也已转义。
// 因此这里的 v-html 不会执行用户脚本。
const html = computed(() => renderMarkdown(props.source))
</script>

<template>
  <!-- eslint-disable-next-line vue/no-v-html -- 内容已由 markdown-it 转义 -->
  <div class="markdown-body" v-html="html" />
</template>

<style>
/* 故意不加 scoped：v-html 生成的内容不带 data-v 属性，scoped 样式选不中 */
.markdown-body {
  font-size: 15px;
  line-height: 1.75;
  color: var(--ln-text);
  word-wrap: break-word;
}

.markdown-body h1,
.markdown-body h2,
.markdown-body h3 {
  margin: 24px 0 12px;
  font-weight: 600;
  line-height: 1.4;
}

.markdown-body h1 {
  font-size: 22px;
}

.markdown-body h2 {
  padding-bottom: 6px;
  font-size: 18px;
  border-bottom: 1px solid var(--ln-border);
}

.markdown-body h3 {
  font-size: 16px;
}

.markdown-body p {
  margin: 12px 0;
}

.markdown-body ul,
.markdown-body ol {
  padding-left: 24px;
  margin: 12px 0;
}

.markdown-body li {
  margin: 4px 0;
}

.markdown-body a {
  color: var(--ln-primary);
}

.markdown-body blockquote {
  padding: 4px 14px;
  margin: 12px 0;
  color: var(--ln-text-muted);
  border-left: 3px solid var(--ln-border);
}

.markdown-body code {
  padding: 2px 5px;
  font-family: ui-monospace, 'Cascadia Code', Consolas, monospace;
  font-size: 13px;
  background: rgb(15 23 42 / 6%);
  border-radius: 4px;
}

.markdown-body pre.hljs {
  padding: 14px 16px;
  margin: 14px 0;
  overflow-x: auto;
  background: #f6f8fa;
  border: 1px solid var(--ln-border);
  border-radius: 8px;
}

.markdown-body pre.hljs code {
  padding: 0;
  font-size: 13px;
  background: none;
}

.markdown-body table {
  width: 100%;
  margin: 14px 0;
  border-collapse: collapse;
}

.markdown-body th,
.markdown-body td {
  padding: 8px 12px;
  border: 1px solid var(--ln-border);
}

.markdown-body th {
  font-weight: 600;
  background: var(--ln-bg);
}
</style>
