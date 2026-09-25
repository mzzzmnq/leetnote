<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { BarChart, LineChart, PieChart } from 'echarts/charts'
import {
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
} from 'echarts/components'
import * as echarts from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsCoreOption } from 'echarts/core'

// 【按需注册】只引入真正用到的图表与组件。
// 全量 `import * as echarts from 'echarts'` 会把所有图表类型都打进产物（1MB+），
// 按需引入后只剩几十 KB。
echarts.use([
  LineChart,
  BarChart,
  PieChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
  CanvasRenderer,
])

const props = defineProps<{
  option: EChartsCoreOption
  height?: string
}>()

const el = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let observer: ResizeObserver | null = null

function render(): void {
  if (!chart) return
  // notMerge = true：整体替换配置。
  // 否则切换数据时上一次的 series 会残留，图表越叠越乱。
  chart.setOption(props.option, true)
}

onMounted(() => {
  if (!el.value) return

  chart = echarts.init(el.value)
  render()

  // 容器尺寸变化时重绘（窗口缩放、布局变化）
  observer = new ResizeObserver(() => chart?.resize())
  observer.observe(el.value)
})

watch(() => props.option, render, { deep: true })

onBeforeUnmount(() => {
  // 不清理会泄漏 canvas 与事件监听
  observer?.disconnect()
  chart?.dispose()
  chart = null
})
</script>

<template>
  <div ref="el" class="chart" :style="{ height: height ?? '280px' }" />
</template>

<style scoped>
.chart {
  width: 100%;
}
</style>
