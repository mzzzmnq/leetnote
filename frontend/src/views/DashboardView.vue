<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NSpin, useMessage } from 'naive-ui'
import type { EChartsCoreOption } from 'echarts/core'
import { ApiError } from '@/api/client'
import { listNotes, toggleStar } from '@/api/notes'
import { fetchOverview, fetchTrend } from '@/api/stats'
import type { NoteListItem, StatsOverview, TrendPoint } from '@/api/types'
import BaseChart from '@/components/BaseChart.vue'
import NoteCard from '@/components/NoteCard.vue'

const router = useRouter()
const message = useMessage()

const loading = ref(false)
const overview = ref<StatsOverview | null>(null)
const trend = ref<TrendPoint[]>([])
const recent = ref<NoteListItem[]>([])

const summaryCards = computed(() => {
  const s = overview.value
  return [
    { label: '笔记总数', value: s?.total_notes ?? 0, hint: `${s?.drafts ?? 0} 篇草稿` },
    { label: '覆盖题目', value: s?.total_problems ?? 0, hint: `共 ${s?.total_solutions ?? 0} 个解法` },
    { label: '当前连续', value: s?.current_streak ?? 0, hint: `最长 ${s?.longest_streak ?? 0} 天`, unit: '天' },
    { label: '累计打卡', value: s?.active_days ?? 0, hint: `收藏 ${s?.starred ?? 0} 篇`, unit: '天' },
  ]
})

// ---------------- 图表配置 ----------------

const trendOption = computed<EChartsCoreOption>(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: 36, right: 16, top: 20, bottom: 28 },
  xAxis: {
    type: 'category',
    // 只显示 MM-DD，省空间
    data: trend.value.map((p) => p.date.slice(5)),
    boundaryGap: false,
    axisLabel: { interval: Math.max(0, Math.floor(trend.value.length / 10) - 1) },
  },
  yAxis: { type: 'value', minInterval: 1 },
  series: [
    {
      type: 'line',
      name: '新增笔记',
      smooth: true,
      symbolSize: 6,
      data: trend.value.map((p) => p.count),
      itemStyle: { color: '#2f6fed' },
      lineStyle: { width: 2 },
      areaStyle: { opacity: 0.12 },
    },
  ],
}))

const difficultyOption = computed<EChartsCoreOption>(() => {
  const d = overview.value?.difficulty
  return {
    tooltip: { trigger: 'item' },
    legend: { bottom: 0, itemWidth: 10, itemHeight: 10 },
    series: [
      {
        type: 'pie',
        radius: ['48%', '72%'],
        avoidLabelOverlap: true,
        label: { show: false },
        data: [
          { name: '简单', value: d?.easy ?? 0, itemStyle: { color: '#18a058' } },
          { name: '中等', value: d?.medium ?? 0, itemStyle: { color: '#f0a020' } },
          { name: '困难', value: d?.hard ?? 0, itemStyle: { color: '#d03050' } },
        ],
      },
    ],
  }
})

const hasDifficultyData = computed(() => {
  const d = overview.value?.difficulty
  return !!d && d.easy + d.medium + d.hard > 0
})

// ---------------- 数据加载 ----------------

async function load(): Promise<void> {
  loading.value = true
  try {
    const [ov, tr, notes] = await Promise.all([
      fetchOverview(),
      fetchTrend(30),
      listNotes({ page: 1, size: 5, sort: 'updated' }),
    ])
    overview.value = ov
    trend.value = tr.points
    recent.value = notes.items
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '加载统计数据失败')
  } finally {
    loading.value = false
  }
}

async function handleToggleStar(id: number): Promise<void> {
  try {
    const updated = await toggleStar(id)
    const target = recent.value.find((n) => n.id === id)
    if (target) target.is_starred = updated.is_starred
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '操作失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <header class="dash-head">
      <div>
        <h1 class="page__title">概览</h1>
        <p class="page__subtitle">刷题进度与记录</p>
      </div>
      <n-button @click="load">刷新</n-button>
    </header>

    <n-spin :show="loading">
      <div class="cards">
        <div v-for="card in summaryCards" :key="card.label" class="card">
          <div class="card__label">{{ card.label }}</div>
          <div class="card__value">
            {{ card.value }}<span v-if="card.unit" class="card__unit">{{ card.unit }}</span>
          </div>
          <div class="card__hint">{{ card.hint }}</div>
        </div>
      </div>

      <div class="charts">
        <section class="panel panel--wide">
          <h2 class="panel__title">近 30 天新增</h2>
          <BaseChart :option="trendOption" height="260px" />
        </section>

        <section class="panel">
          <h2 class="panel__title">难度分布</h2>
          <BaseChart v-if="hasDifficultyData" :option="difficultyOption" height="260px" />
          <p v-else class="panel__empty">给笔记关联题目后即可看到分布</p>
        </section>
      </div>

      <section class="recent">
        <header class="recent__head">
          <h2 class="panel__title">最近更新</h2>
          <n-button size="small" quaternary @click="router.push({ name: 'notes' })">
            查看全部
          </n-button>
        </header>

        <div v-if="recent.length === 0" class="panel__empty" style="padding: 40px 0">
          还没有笔记，去写下第一篇吧
        </div>

        <div v-else class="recent__list">
          <NoteCard
            v-for="note in recent"
            :key="note.id"
            :note="note"
            @toggle-star="handleToggleStar"
          />
        </div>
      </section>
    </n-spin>
  </div>
</template>

<style scoped>
.dash-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 20px;
}

.dash-head .page__subtitle {
  margin: 0;
}

.cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 14px;
  margin-bottom: 20px;
}

.card {
  padding: 18px 20px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.card__label {
  font-size: 13px;
  color: var(--ln-text-muted);
}

.card__value {
  margin: 6px 0 4px;
  font-size: 28px;
  font-weight: 600;
  line-height: 1.1;
}

.card__unit {
  margin-left: 4px;
  font-size: 14px;
  font-weight: 400;
  color: var(--ln-text-muted);
}

.card__hint {
  font-size: 12px;
  color: var(--ln-text-muted);
}

.charts {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 14px;
  margin-bottom: 20px;
}

.panel {
  padding: 18px 20px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.panel__title {
  margin: 0 0 8px;
  font-size: 15px;
}

.panel__empty {
  margin: 0;
  font-size: 13px;
  color: var(--ln-text-muted);
  text-align: center;
}

.recent {
  padding: 18px 20px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.recent__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.recent__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

@media (max-width: 900px) {
  .charts {
    grid-template-columns: 1fr;
  }
}
</style>
