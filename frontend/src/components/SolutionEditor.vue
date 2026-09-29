<script setup lang="ts">
import { computed, h } from 'vue'
import { NButton, NInput, NSelect } from 'naive-ui'
import type { SelectRenderLabel } from 'naive-ui'
import type { SolutionInput } from '@/api/types'
import { LANGUAGES, languageDotColor, languageLabel } from '@/utils/languages'

const props = defineProps<{
  modelValue: SolutionInput
  title: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: SolutionInput]
  remove: []
}>()

const languageOptions = computed(() =>
  LANGUAGES.map((lang) => ({ value: lang, label: languageLabel(lang) })),
)

/**
 * 下拉项渲染成「品牌色圆点 + 展示名」。
 *
 * 圆点颜色不走 options 里的自定义字段（Naive 的 SelectOption 类型没有它，
 * 硬塞要到处 as 断言），而是按 value 现查 —— 一份数据一个来源。
 */
const renderLanguageLabel: SelectRenderLabel = (option) => {
  const lang = String(option.value ?? '')
  return h('span', { class: 'sol__lang' }, [
    h('i', { class: 'sol__dot', style: { background: languageDotColor(lang) } }),
    languageLabel(lang),
  ])
}

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
        :input-props="{ 'aria-label': '解法名称' }"
        maxlength="100"
        @update:value="(v: string) => patch('title', v)"
      />
      <n-select
        :value="modelValue.language"
        :options="languageOptions"
        :render-label="renderLanguageLabel"
        placeholder="语言"
        :input-props="{ 'aria-label': '语言' }"
        style="width: 150px"
        @update:value="(v: string) => patch('language', v)"
      />
    </div>

    <div class="sol__row">
      <n-input
        :value="modelValue.time_complexity ?? ''"
        placeholder="时间复杂度，如 O(n)"
        :input-props="{ 'aria-label': '时间复杂度' }"
        maxlength="50"
        @update:value="(v: string) => patch('time_complexity', v)"
      />
      <n-input
        :value="modelValue.space_complexity ?? ''"
        placeholder="空间复杂度，如 O(1)"
        :input-props="{ 'aria-label': '空间复杂度' }"
        maxlength="50"
        @update:value="(v: string) => patch('space_complexity', v)"
      />
    </div>

    <n-input
      :value="modelValue.code"
      type="textarea"
      placeholder="代码"
      :input-props="{ 'aria-label': '代码' }"
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

/* 语言下拉项前面的品牌色小圆点 */
.sol__lang {
  display: inline-flex;
  gap: 8px;
  align-items: center;
}

.sol__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
</style>
