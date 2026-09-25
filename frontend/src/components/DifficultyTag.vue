<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import type { Difficulty } from '@/api/types'

const props = defineProps<{ difficulty: Difficulty }>()

const LABELS: Record<Difficulty, string> = {
  Easy: '简单',
  Medium: '中等',
  Hard: '困难',
}

const type = computed(() => {
  switch (props.difficulty) {
    case 'Easy':
      return 'success' as const
    case 'Medium':
      return 'warning' as const
    case 'Hard':
      return 'error' as const
    default:
      return 'default' as const
  }
})

const label = computed(() => LABELS[props.difficulty] ?? props.difficulty)
</script>

<template>
  <n-tag :type="type" size="small" :bordered="false">{{ label }}</n-tag>
</template>
