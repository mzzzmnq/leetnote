<script setup lang="ts">
import { computed } from 'vue'

/**
 * 难度分徽标。
 *
 * LeetCode 官方只给 Easy/Medium/Hard 三档 —— 同为 Medium，难度分可能
 * 是 1400 也可能是 2600，实际难度天差地别。这个分（来自社区项目
 * zerotrac/leetcode_problem_rating）能真正区分开。
 *
 * 分数从竞赛的通过率与参赛者表现反推得到，数值越大越难：
 *   1200 上下 ≈ 纯模板题        1700 上下 ≈ 需要一点思考
 *   2000 以上 ≈ 需要组合多个技巧  2400 以上 ≈ 竞赛压轴水平
 */
const props = defineProps<{
  /** null 表示没有这个数据（竞赛时代之前的老题没有分） */
  rating: number | null
}>()

/** 分档配色，与难度分本身的量级保持一致 */
const tone = computed(() => {
  const r = props.rating
  if (r === null) return 'none'
  if (r < 1400) return 'easy'
  if (r < 1700) return 'normal'
  if (r < 2000) return 'hard'
  return 'extreme'
})

/** 只取整数部分：难度分带 10 位小数，展示出来太吵 */
const label = computed(() =>
  props.rating === null ? '' : String(Math.round(props.rating)),
)
</script>

<template>
  <span v-if="rating !== null" class="rating" :class="`rating--${tone}`" :title="`难度分 ${rating.toFixed(1)}`">
    {{ label }}
  </span>
  <!-- 没有数据时留一个占位符，保证各行右侧对齐、不会看起来像排版错乱 -->
  <span v-else class="rating rating--none" title="该题没有难度分（早于 LeetCode 竞赛时代的题目不在统计范围内）">—</span>
</template>

<style scoped>
.rating {
  display: inline-block;
  min-width: 42px;
  padding: 1px 7px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  text-align: center;
  border-radius: 4px;
  background: #f0f0f3;
  color: #6b6b76;
}

.rating--easy {
  background: #e8f7ee;
  color: #1f8a4c;
}

.rating--normal {
  background: #e8f1fd;
  color: #2b6cb0;
}

.rating--hard {
  background: #fdf1e3;
  color: #b5600d;
}

.rating--extreme {
  background: #fdeaea;
  color: #c0392b;
}
</style>
