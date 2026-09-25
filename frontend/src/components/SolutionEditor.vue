<script setup lang="ts">
import { NButton, NInput, NSelect } from 'naive-ui'
import type { SolutionInput } from '@/api/types'

const props = defineProps<{
  modelValue: SolutionInput
  title: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: SolutionInput]
  remove: []
}>()

const LANGUAGES = [
  'go',
  'python',
  'java',
  'cpp',
  'c',
  'javascript',
  'typescript',
  'rust',
  'csharp',
  'kotlin',
  'swift',
  'sql',
]

const languageOptions = LANGUAGES.map((lang) => ({ label: lang, value: lang }))

// 不可变更新：每次 emit 一个新对象，父组件替换数组元素。
// 直接改 props 里的字段是反模式（Vue 会警告，且难以追踪变更）。
function patch(field: keyof SolutionInput, value: string): void {
  emit('update:modelValue', { ...props.modelValue, [field]: value })
}
</script>

<template>
  <div class="sol">
    <div class="sol__head">
      <span class="sol__index">{{ title }}</span>
      <n-button size="tiny" quaternary type="error" @click="emit('remove')">删除</n-button>
    </div>

    <div class="sol__row">
      <n-input
        :value="modelValue.title"
        placeholder="解法名称，如「哈希表 · 一次遍历」"
        maxlength="100"
        @update:value="(v: string) => patch('title', v)"
      />
      <n-select
        :value="modelValue.language"
        :options="languageOptions"
        placeholder="语言"
        style="width: 140px"
        @update:value="(v: string) => patch('language', v)"
      />
    </div>

    <div class="sol__row">
      <n-input
        :value="modelValue.time_complexity ?? ''"
        placeholder="时间复杂度，如 O(n)"
        maxlength="50"
        @update:value="(v: string) => patch('time_complexity', v)"
      />
      <n-input
        :value="modelValue.space_complexity ?? ''"
        placeholder="空间复杂度，如 O(1)"
        maxlength="50"
        @update:value="(v: string) => patch('space_complexity', v)"
      />
    </div>

    <n-input
      :value="modelValue.code"
      type="textarea"
      placeholder="代码"
      :autosize="{ minRows: 6, maxRows: 24 }"
      style="font-family: ui-monospace, Consolas, monospace"
      @update:value="(v: string) => patch('code', v)"
    />
  </div>
</template>

<style scoped>
.sol {
  padding: 16px;
  margin-bottom: 12px;
  background: var(--ln-bg);
  border: 1px solid var(--ln-border);
  border-radius: 8px;
}

.sol__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.sol__index {
  font-size: 13px;
  font-weight: 600;
  color: var(--ln-text-muted);
}

.sol__row {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
}

.sol__row > :first-child {
  flex: 1;
}
</style>
