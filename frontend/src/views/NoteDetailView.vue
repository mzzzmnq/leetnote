<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NSpin, NTag, useDialog, useMessage } from 'naive-ui'
import { ApiError } from '@/api/client'
import { deleteNote, getNote, toggleStar } from '@/api/notes'
import type { Note } from '@/api/types'
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

const noteId = Number(route.params.id)

const problemLabel = computed(() => {
  const p = note.value?.problem
  if (!p) return null
  return p.leetcode_id ? `${p.leetcode_id}. ${p.title}` : p.title
})

async function load(): Promise<void> {
  loading.value = true
  failed.value = ''

  try {
    note.value = await getNote(noteId)
  } catch (error) {
    failed.value = error instanceof ApiError ? error.message : '加载失败'
  } finally {
    loading.value = false
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
        await deleteNote(noteId)
        message.success('已删除')
        await router.push({ name: 'notes' })
      } catch (error) {
        message.error(error instanceof ApiError ? error.message : '删除失败')
      }
    },
  })
}

onMounted(load)
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
</style>
