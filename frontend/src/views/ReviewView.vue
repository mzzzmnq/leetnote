<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NEmpty, NSpin, NTag, useMessage } from 'naive-ui'
import type { EChartsCoreOption } from 'echarts/core'
import { ApiError } from '@/api/client'
import { getNote } from '@/api/notes'
import { fetchDueCards, fetchReviewStats, submitReview } from '@/api/reviews'
import type { Note, ReviewCard, ReviewStats } from '@/api/types'
import BaseChart from '@/components/BaseChart.vue'
import DifficultyTag from '@/components/DifficultyTag.vue'
import MarkdownViewer from '@/components/MarkdownViewer.vue'
import { useThemeStore } from '@/stores/theme'
import { chartColors } from '@/utils/chartTheme'
import { renderCodeBlock } from '@/utils/markdown'

const router = useRouter()
const message = useMessage()
const themeStore = useThemeStore()

const palette = computed(() => chartColors(themeStore.isDark))

const loading = ref(true)
const stats = ref<ReviewStats | null>(null)

// ---------- 复习会话状态 ----------
const reviewing = ref(false)
const queue = ref<ReviewCard[]>([])
const index = ref(0)
const current = ref<Note | null>(null)
const loadingNote = ref(false)
const revealed = ref(false)
const submitting = ref(false)

// 本次会话的成绩，结束时展示
const session = ref({ again: 0, good: 0 })

const currentCard = computed(() => queue.value[index.value] ?? null)
const progress = computed(() => {
  if (queue.value.length === 0) return 0
  return Math.round(((index.value) / queue.value.length) * 100)
})
const finished = computed(() => reviewing.value && index.value >= queue.value.length)

// SM-2 的评分刻度（含义尽量直白，避免用户不知道该选哪个）
const RATINGS = [
  { value: 0, label: '完全忘了', desc: '重来', type: 'error' as const },
  { value: 2, label: '想不起来', desc: '重来', type: 'warning' as const },
  { value: 3, label: '勉强答对', desc: '间隔缩短', type: 'default' as const },
  { value: 4, label: '答对了', desc: '正常推进', type: 'primary' as const },
  { value: 5, label: '很轻松', desc: '间隔拉长', type: 'success' as const },
]

const forecastOption = computed<EChartsCoreOption>(() => {
  const points = stats.value?.forecast ?? []
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 30, right: 12, top: 16, bottom: 24 },
    xAxis: {
      type: 'category',
      data: points.map((p) => p.date.slice(5)),
    },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      {
        type: 'bar',
        name: '到期卡片',
        data: points.map((p) => p.count),
        barMaxWidth: 28,
        itemStyle: { color: palette.value.primary, borderRadius: [4, 4, 0, 0] },
      },
    ],
  }
})

async function loadStats(): Promise<void> {
  try {
    stats.value = await fetchReviewStats()
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '加载复习统计失败')
  }
}

async function load(): Promise<void> {
  loading.value = true
  await loadStats()
  loading.value = false
}

async function startReview(): Promise<void> {
  try {
    const cards = await fetchDueCards(50)
    if (cards.length === 0) {
      message.info('当前没有到期需要复习的卡片')
      return
    }
    queue.value = cards
    index.value = 0
    revealed.value = false
    current.value = null
    session.value = { again: 0, good: 0 }
    reviewing.value = true
    void loadCurrentNote()
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '加载待复习卡片失败')
  }
}

async function loadCurrentNote(): Promise<void> {
  const card = currentCard.value
  if (!card) return

  loadingNote.value = true
  try {
    current.value = await getNote(card.note_id)
  } catch {
    // 笔记可能已被删除；卡片会随外键级联清理，这里静默跳过
    current.value = null
  } finally {
    loadingNote.value = false
  }
}

async function rate(value: number): Promise<void> {
  const card = currentCard.value
  if (!card || submitting.value) return

  submitting.value = true
  try {
    const updated = await submitReview(card.id, value)
    if (value < 3) {
      session.value.again += 1
    } else {
      session.value.good += 1
    }

    message.success(
      updated.interval_days <= 1
        ? '已重置，明天再见'
        : `记住了，${updated.interval_days} 天后再复习`,
    )

    index.value += 1
    revealed.value = false
    current.value = null

    if (index.value < queue.value.length) {
      void loadCurrentNote()
    } else {
      await loadStats()
    }
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '提交失败')
  } finally {
    submitting.value = false
  }
}

function exitReview(): void {
  reviewing.value = false
  void loadStats()
}

onMounted(load)
</script>

<template>
  <div class="page">
    <n-spin :show="loading">
      <!-- ============ 复习会话 ============ -->
      <template v-if="reviewing">
        <header class="session-head">
          <n-button size="small" quaternary @click="exitReview">← 退出复习</n-button>
          <div class="session-progress">
            <span>{{ Math.min(index + 1, queue.length) }} / {{ queue.length }}</span>
            <div class="session-bar">
              <div class="session-bar__fill" :style="{ width: `${progress}%` }" />
            </div>
          </div>
        </header>

        <div v-if="finished" class="finish">
          <h2 class="finish__title">本轮复习完成 🎉</h2>
          <p class="finish__desc">
            记住 {{ session.good }} 张，需要重来 {{ session.again }} 张
          </p>
          <n-button type="primary" @click="exitReview">返回</n-button>
        </div>

        <template v-else-if="currentCard">
          <article class="flashcard">
            <div class="flashcard__head">
              <h2 class="flashcard__title">{{ currentCard.note_title }}</h2>
              <n-tag v-if="currentCard.repetitions === 0" size="small" :bordered="false">
                新卡片
              </n-tag>
              <n-tag v-else size="small" :bordered="false" type="info">
                第 {{ currentCard.repetitions + 1 }} 次
              </n-tag>
            </div>

            <p v-if="currentCard.note_summary && !revealed" class="flashcard__summary">
              {{ currentCard.note_summary }}
            </p>

            <!-- 先自己想，再显示答案 —— 这是间隔重复的关键：
                 主动回忆的效果远好于直接重读 -->
            <div v-if="!revealed" class="flashcard__prompt">
              <p>先在心里回忆一下思路，再显示答案</p>
              <n-button type="primary" @click="revealed = true">显示答案</n-button>
            </div>

            <template v-else>
              <n-spin v-if="loadingNote" size="small" />

              <template v-else-if="current">
                <div v-if="current.problem" class="flashcard__problem">
                  <DifficultyTag :difficulty="current.problem.difficulty" />
                  <span>
                    {{ current.problem.leetcode_id ? `${current.problem.leetcode_id}. ` : '' }}{{
                      current.problem.title
                    }}
                  </span>
                </div>

                <MarkdownViewer v-if="current.content_md" :source="current.content_md" />

                <div v-for="sol in current.solutions" :key="sol.id" class="flashcard__solution">
                  <div class="flashcard__solution-head">
                    <strong>{{ sol.title }}</strong>
                    <n-tag size="tiny" :bordered="false">{{ sol.language }}</n-tag>
                    <span v-if="sol.time_complexity" class="flashcard__cx">
                      {{ sol.time_complexity }}
                    </span>
                  </div>
                  <!-- eslint-disable-next-line vue/no-v-html -- 由 highlight.js 转义后输出 -->
                  <div v-html="renderCodeBlock(sol.code, sol.language)" />
                </div>
              </template>

              <p v-else class="flashcard__missing">笔记内容已不可用</p>

              <div class="ratings">
                <p class="ratings__hint">刚才回忆得怎么样？</p>
                <div class="ratings__row">
                  <n-button
                    v-for="r in RATINGS"
                    :key="r.value"
                    :type="r.type"
                    :disabled="submitting"
                    class="ratings__btn"
                    @click="rate(r.value)"
                  >
                    <span class="ratings__label">{{ r.label }}</span>
                    <span class="ratings__desc">{{ r.desc }}</span>
                  </n-button>
                </div>
              </div>
            </template>
          </article>
        </template>
      </template>

      <!-- ============ 概览 ============ -->
      <template v-else>
        <header class="review-head">
          <div>
            <h1 class="page__title">复习</h1>
            <p class="page__subtitle">基于 SM-2 算法的间隔重复</p>
          </div>
          <n-button type="primary" :disabled="(stats?.due_count ?? 0) === 0" @click="startReview">
            开始复习（{{ stats?.due_count ?? 0 }}）
          </n-button>
        </header>

        <div class="cards">
          <div class="card">
            <div class="card__label">待复习</div>
            <div class="card__value">{{ stats?.due_count ?? 0 }}</div>
            <div class="card__hint">现在就该看的</div>
          </div>
          <div class="card">
            <div class="card__label">卡片总数</div>
            <div class="card__value">{{ stats?.total_cards ?? 0 }}</div>
            <div class="card__hint">每篇笔记一张</div>
          </div>
          <div class="card">
            <div class="card__label">已复习过</div>
            <div class="card__value">{{ stats?.learned_cards ?? 0 }}</div>
            <div class="card__hint">至少看过一次</div>
          </div>
          <div class="card">
            <div class="card__label">累计复习</div>
            <div class="card__value">{{ stats?.total_reviews ?? 0 }}</div>
            <div class="card__hint">次</div>
          </div>
        </div>

        <section class="panel">
          <h2 class="panel__title">未来 7 天复习压力</h2>
          <BaseChart :option="forecastOption" height="200px" />
        </section>

        <section v-if="(stats?.due_count ?? 0) === 0" class="panel panel--empty">
          <n-empty description="今天没有需要复习的内容">
            <template #extra>
              <n-button size="small" @click="router.push({ name: 'notes' })">
                去看笔记
              </n-button>
            </template>
          </n-empty>
        </section>
      </template>
    </n-spin>
  </div>
</template>

<style scoped>
.review-head,
.session-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.review-head .page__subtitle {
  margin: 0;
}

.cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 14px;
  margin-bottom: 20px;
}

.card {
  padding: 18px 20px;
  background: var(--ln-surface);
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius);
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

.card__hint {
  font-size: 12px;
  color: var(--ln-text-muted);
}

.panel {
  padding: 18px 20px;
  background: var(--ln-surface);
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius);
}

.panel--empty {
  margin-top: 20px;
  padding: 40px 20px;
}

.panel__title {
  margin: 0 0 8px;
  font-size: 15px;
}

/* ---------- 复习会话 ---------- */
.session-progress {
  display: flex;
  gap: 10px;
  align-items: center;
  font-size: 13px;
  color: var(--ln-text-muted);
}

.session-bar {
  width: 160px;
  height: 6px;
  overflow: hidden;
  background: var(--ln-bg);
  border-radius: 3px;
}

.session-bar__fill {
  height: 100%;
  background: var(--ln-primary);
  transition: width 0.2s;
}

.flashcard {
  padding: 28px 32px;
  background: var(--ln-surface);
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius-lg);
}

.flashcard__head {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 8px;
}

.flashcard__title {
  margin: 0;
  font-size: 22px;
}

.flashcard__summary {
  margin: 0 0 16px;
  font-size: 14px;
  color: var(--ln-text-muted);
}

.flashcard__prompt {
  display: flex;
  flex-direction: column;
  gap: 14px;
  align-items: center;
  padding: 48px 0;
  color: var(--ln-text-muted);
  background: var(--ln-bg);
  border-radius: 8px;
}

.flashcard__prompt p {
  margin: 0;
  font-size: 14px;
}

.flashcard__problem {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 14px;
  font-size: 13px;
  color: var(--ln-text-muted);
}

.flashcard__solution {
  padding-top: 12px;
  margin-top: 16px;
  border-top: 1px solid var(--ln-border);
}

.flashcard__solution-head {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
  font-size: 14px;
}

.flashcard__cx {
  font-size: 12px;
  color: var(--ln-text-muted);
}

.flashcard__missing {
  padding: 24px 0;
  color: var(--ln-text-muted);
  text-align: center;
}

.ratings {
  padding-top: 20px;
  margin-top: 24px;
  border-top: 1px solid var(--ln-border);
}

.ratings__hint {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--ln-text-muted);
}

.ratings__row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.ratings__btn {
  flex: 1;
  min-width: 110px;
  height: auto;
  padding: 10px 8px;
}

.ratings__label,
.ratings__desc {
  display: block;
  line-height: 1.4;
}

.ratings__desc {
  font-size: 11px;
  opacity: 0.7;
}

.finish {
  padding: 60px 20px;
  text-align: center;
  background: var(--ln-surface);
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius-lg);
}

.finish__title {
  margin: 0 0 8px;
  font-size: 22px;
}

.finish__desc {
  margin: 0 0 20px;
  color: var(--ln-text-muted);
}
</style>
