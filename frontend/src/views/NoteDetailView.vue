<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { NAlert, NButton, NSpin, NTag, useDialog, useMessage } from 'naive-ui'
import { ApiError } from '@/api/client'
import { deleteNote, explainNote, fetchSimilarNotes, getNote, toggleStar } from '@/api/notes'
import type { Note, SimilarNote } from '@/api/types'
import DifficultyTag from '@/components/DifficultyTag.vue'
import MarkdownViewer from '@/components/MarkdownViewer.vue'
import { formatDate } from '@/utils/format'
import { renderCodeBlock } from '@/utils/markdown'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const loading = ref(true)
const failed = ref('')
const note = ref<Note | null>(null)
const starring = ref(false)

// 相似题推荐（来自 leetnote-ai 服务）
const similar = ref<SimilarNote[]>([])
const similarModel = ref('')
const similarLoading = ref(false)

// AI 讲解（同步等 LLM 生成，实测十几秒）
const explainContent = ref('')
const explainModel = ref('')
const explainUsage = ref('')
const explaining = ref(false)
const explainError = ref('')

// 用 computed 而不是一次性取值。
//
// 【为什么】从「相似题推荐」点进另一篇笔记时，只有路由参数变了，
// Vue Router 会【复用同一个组件实例】（不重新挂载），于是 onMounted
// 不会再触发，页面就会一直停在上一篇的内容上。
// 必须 watch 参数变化才能重新加载。
const noteId = computed(() => Number(route.params.id))

const problemLabel = computed(() => {
  const p = note.value?.problem
  if (!p) return null
  return p.leetcode_id ? `${p.leetcode_id}. ${p.title}` : p.title
})

async function load(): Promise<void> {
  loading.value = true
  failed.value = ''
  // 清掉上一篇的内容，避免切换笔记时闪现旧数据
  note.value = null
  similar.value = []
  explainContent.value = ''
  explainError.value = ''

  try {
    note.value = await getNote(noteId.value)
  } catch (error) {
    failed.value = error instanceof ApiError ? error.message : '加载失败'
  } finally {
    loading.value = false
  }
}

// 相似题是附加功能：失败就静默不显示，不该影响笔记正文的阅读。
// 向量生成是异步的，刚创建的笔记可能还没算好，这时也会返回空列表。
async function loadSimilar(): Promise<void> {
  similarLoading.value = true
  try {
    const data = await fetchSimilarNotes(noteId.value, 5)
    similar.value = data.items
    similarModel.value = data.model
  } catch {
    similar.value = []
  } finally {
    similarLoading.value = false
  }
}

// AI 讲解：同步等待 LLM 生成（实测十几秒），必须给足加载反馈。
// 失败时展示错误而不是静默 —— 用户点了按钮却什么都没发生会很困惑。
async function handleExplain(): Promise<void> {
  explaining.value = true
  explainError.value = ''

  try {
    const data = await explainNote(noteId.value)
    explainContent.value = data.content
    explainModel.value = data.model
    explainUsage.value = `prompt ${data.prompt_tokens} / completion ${data.completion_tokens}`
  } catch (error) {
    explainError.value = error instanceof ApiError ? error.message : '生成失败，请稍后重试'
  } finally {
    explaining.value = false
  }
}

async function handleToggleStar(): Promise<void> {
  if (!note.value) return
  starring.value = true
  try {
    const updated = await toggleStar(note.value.id)
    note.value.is_starred = updated.is_starred
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '操作失败')
  } finally {
    starring.value = false
  }
}

function handleDelete(): void {
  if (!note.value) return

  dialog.warning({
    title: '删除笔记',
    content: `确定要删除「${note.value.title}」吗？该操作不可撤销。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteNote(noteId.value)
        message.success('已删除')
        await router.push({ name: 'notes' })
      } catch (error) {
        message.error(error instanceof ApiError ? error.message : '删除失败')
      }
    },
  })
}

// immediate: true 让它替代 onMounted —— 首次挂载时立即加载，
// 之后每次路由参数变化（比如点「相似题推荐」跳转）也会重新加载。
watch(
  noteId,
  async (id) => {
    if (!Number.isInteger(id) || id <= 0) {
      failed.value = '笔记 ID 无效'
      return
    }
    await load()
    // 正文先展示出来，相似题在后台加载 —— 不阻塞首屏
    void loadSimilar()
  },
  { immediate: true },
)
</script>

<template>
  <div class="page">
    <n-spin :show="loading">
      <div v-if="failed" class="state-box">
        <p>{{ failed }}</p>
        <n-button size="small" @click="router.push({ name: 'notes' })">返回列表</n-button>
      </div>

      <template v-else-if="note">
        <div class="detail-head">
          <n-button size="small" quaternary @click="router.push({ name: 'notes' })">← 返回</n-button>
          <div class="detail-head__actions">
            <n-button size="small" :loading="starring" @click="handleToggleStar">
              {{ note.is_starred ? '★ 已收藏' : '☆ 收藏' }}
            </n-button>
            <n-button size="small" @click="router.push({ name: 'note-edit', params: { id: note.id } })">
              编辑
            </n-button>
            <n-button size="small" type="error" quaternary @click="handleDelete">删除</n-button>
          </div>
        </div>

        <article class="detail-card">
          <h1 class="detail-title">{{ note.title }}</h1>

          <div class="detail-meta">
            <n-tag v-if="note.status === 'draft'" size="small" :bordered="false">草稿</n-tag>
            <template v-if="problemLabel">
              <DifficultyTag :difficulty="note.problem!.difficulty" />
              <span>{{ problemLabel }}</span>
            </template>
            <span>{{ formatDate(note.created_at) }}</span>
          </div>

          <div v-if="note.tags.length" class="detail-tags">
            <n-tag v-for="tag in note.tags" :key="tag.id" size="small" :bordered="false" type="info">
              {{ tag.name }}
            </n-tag>
          </div>

          <n-empty v-if="!note.content_md" description="还没有正文" style="padding: 32px 0" />
          <MarkdownViewer v-else :source="note.content_md" class="detail-content" />
        </article>

        <section v-if="note.solutions.length" class="solutions">
          <h2 class="solutions__title">
            解法
            <span class="solutions__count">{{ note.solutions.length }}</span>
          </h2>

          <article v-for="(sol, index) in note.solutions" :key="sol.id" class="solution">
            <header class="solution__head">
              <span class="solution__index">#{{ index + 1 }}</span>
              <span class="solution__title">{{ sol.title }}</span>
              <n-tag size="small" :bordered="false">{{ sol.language }}</n-tag>
              <span v-if="sol.time_complexity" class="solution__cx">
                时间 {{ sol.time_complexity }}
              </span>
              <span v-if="sol.space_complexity" class="solution__cx">
                空间 {{ sol.space_complexity }}
              </span>
            </header>

            <!-- eslint-disable-next-line vue/no-v-html -- 由 highlight.js 转义后输出 -->
            <div v-html="renderCodeBlock(sol.code, sol.language)" />
          </article>
        </section>

        <section class="explain">
          <header class="explain__head">
            <h2 class="explain__title">
              AI 讲解
              <span v-if="explainModel" class="explain__meta">
                {{ explainModel }} · {{ explainUsage }}
              </span>
            </h2>
            <n-button size="small" :loading="explaining" @click="handleExplain">
              {{ explainContent ? '重新生成' : '生成讲解' }}
            </n-button>
          </header>

          <n-alert v-if="explainError" type="error" :show-icon="false">
            {{ explainError }}
          </n-alert>

          <p v-else-if="explaining" class="explain__hint">
            正在让模型点评这篇笔记……通常需要 10 秒左右，请稍候
          </p>

          <p v-else-if="!explainContent" class="explain__hint">
            让 AI 按「思路 / 关键点 / 复杂度 / 易错点」四个角度点评这篇笔记
          </p>

          <MarkdownViewer v-else :source="explainContent" />
        </section>

        <section v-if="similarLoading || similar.length" class="similar">
          <h2 class="similar__heading">
            相似题推荐
            <span v-if="similarModel" class="similar__model">{{ similarModel }}</span>
          </h2>

          <n-spin v-if="similarLoading" size="small" />

          <div v-else class="similar__list">
            <RouterLink
              v-for="item in similar"
              :key="item.note_id"
              :to="{ name: 'note-detail', params: { id: item.note_id } }"
              class="similar__item"
            >
              <div class="similar__row">
                <span class="similar__title">{{ item.title }}</span>
                <span class="similar__score">
                  {{ Math.round(item.similarity * 100) }}%
                </span>
              </div>
              <p v-if="item.summary" class="similar__summary">{{ item.summary }}</p>
            </RouterLink>
          </div>
        </section>
      </template>
    </n-spin>
  </div>
</template>

<style scoped>
.state-box {
  padding: 60px;
  text-align: center;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.detail-head__actions {
  display: flex;
  gap: 8px;
}

.detail-card {
  padding: 28px 32px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 12px;
}

.detail-title {
  margin: 0 0 12px;
  font-size: 24px;
  line-height: 1.4;
}

.detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  font-size: 13px;
  color: var(--ln-text-muted);
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.detail-content {
  margin-top: 24px;
  border-top: 1px solid var(--ln-border);
  padding-top: 8px;
}

.solutions {
  margin-top: 24px;
}

.solutions__title {
  display: flex;
  gap: 8px;
  align-items: center;
  margin: 0 0 14px;
  font-size: 17px;
}

.solutions__count {
  padding: 1px 8px;
  font-size: 12px;
  color: var(--ln-text-muted);
  background: var(--ln-bg);
  border-radius: 10px;
}

.solution {
  padding: 18px 20px;
  margin-bottom: 14px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.solution__head {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
}

.solution__index {
  font-size: 12px;
  color: var(--ln-text-muted);
}

.solution__title {
  font-weight: 600;
}

.solution__cx {
  font-size: 12px;
  color: var(--ln-text-muted);
}

/* ---------- AI 讲解 ---------- */
.explain {
  margin-top: 24px;
  padding: 18px 20px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.explain__head {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.explain__title {
  display: flex;
  gap: 8px;
  align-items: center;
  margin: 0;
  font-size: 15px;
}

.explain__meta {
  padding: 1px 8px;
  font-size: 11px;
  font-weight: 400;
  color: var(--ln-text-muted);
  background: var(--ln-bg);
  border-radius: 10px;
}

.explain__hint {
  margin: 0;
  font-size: 13px;
  color: var(--ln-text-muted);
}

/* ---------- 相似题推荐 ---------- */
.similar {
  margin-top: 24px;
  padding: 18px 20px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.similar__heading {
  display: flex;
  gap: 8px;
  align-items: center;
  margin: 0 0 12px;
  font-size: 15px;
}

.similar__model {
  padding: 1px 8px;
  font-size: 11px;
  font-weight: 400;
  color: var(--ln-text-muted);
  background: var(--ln-bg);
  border-radius: 10px;
}

.similar__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.similar__item {
  padding: 10px 12px;
  color: inherit;
  background: var(--ln-bg);
  border-radius: 8px;
}

.similar__item:hover {
  background: rgb(47 111 237 / 8%);
  text-decoration: none;
}

.similar__row {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
}

.similar__title {
  font-size: 14px;
  font-weight: 500;
}

.similar__score {
  flex-shrink: 0;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  color: var(--ln-primary);
}

.similar__summary {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--ln-text-muted);
}
</style>
