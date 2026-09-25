<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NButton,
  NEmpty,
  NInput,
  NPagination,
  NSelect,
  NSpin,
  NSwitch,
  useMessage,
} from 'naive-ui'
import { ApiError } from '@/api/client'
import { listNotes, toggleStar } from '@/api/notes'
import { listTags } from '@/api/tags'
import type { Difficulty, NoteListItem, NoteStatus, Tag } from '@/api/types'
import NoteCard from '@/components/NoteCard.vue'

const router = useRouter()
const message = useMessage()

const loading = ref(false)
const notes = ref<NoteListItem[]>([])
const tags = ref<Tag[]>([])
const total = ref(0)
const pageCount = ref(0)

const DIFFICULTY_OPTIONS = [
  { label: '全部难度', value: '' },
  { label: '简单', value: 'Easy' },
  { label: '中等', value: 'Medium' },
  { label: '困难', value: 'Hard' },
]

const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '已发布', value: 'published' },
  { label: '草稿', value: 'draft' },
]

const SORT_OPTIONS = [
  { label: '最近更新', value: 'updated' },
  { label: '最近创建', value: 'created' },
  { label: '最早创建', value: 'created_asc' },
  { label: '按标题', value: 'title' },
]

// tagId = 0 表示「全部标签」（Naive UI 的 Select 不接受 null 作为 option value）
const ALL_TAGS = 0

const query = reactive({
  keyword: '',
  difficulty: '' as Difficulty | '',
  tagId: ALL_TAGS,
  status: '' as NoteStatus | '',
  starredOnly: false,
  sort: 'updated' as 'created' | 'updated' | 'created_asc' | 'title',
  page: 1,
  size: 10,
})

const tagOptions = computed(() => [
  { label: '全部标签', value: ALL_TAGS },
  ...tags.value.map((t) => ({ label: t.name, value: t.id })),
])

const isEmpty = computed(() => !loading.value && notes.value.length === 0)

async function fetchNotes(): Promise<void> {
  loading.value = true
  try {
    const data = await listNotes({
      keyword: query.keyword.trim() || undefined,
      difficulty: query.difficulty || undefined,
      tag_id: query.tagId > 0 ? query.tagId : undefined,
      status: query.status || undefined,
      // 只在开关打开时才传 starred=true。
      // 传 false 会被后端理解成「只看未收藏」，不是我们想要的行为。
      ...(query.starredOnly ? { starred: true } : {}),
      sort: query.sort,
      page: query.page,
      size: query.size,
    })

    notes.value = data.items
    total.value = data.total
    pageCount.value = data.pages
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '加载笔记失败')
  } finally {
    loading.value = false
  }
}

async function fetchTags(): Promise<void> {
  try {
    tags.value = await listTags()
  } catch {
    // 标签加载失败不影响主流程，静默降级
  }
}

function resetAndFetch(): void {
  query.page = 1
  void fetchNotes()
}

async function handleToggleStar(id: number): Promise<void> {
  try {
    const updated = await toggleStar(id)
    const target = notes.value.find((n) => n.id === id)
    if (target) {
      target.is_starred = updated.is_starred
    }
    // 正在按收藏筛选时，取消收藏后该条应从列表消失
    if (query.starredOnly && !updated.is_starred) {
      await fetchNotes()
    }
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '操作失败')
  }
}

function handlePageChange(page: number): void {
  query.page = page
  void fetchNotes()
}

onMounted(async () => {
  await Promise.all([fetchTags(), fetchNotes()])
})
</script>

<template>
  <div class="page">
    <header class="list-head">
      <div>
        <h1 class="page__title">我的笔记</h1>
        <p class="page__subtitle">共 {{ total }} 篇</p>
      </div>
      <n-button type="primary" @click="router.push({ name: 'note-new' })">写笔记</n-button>
    </header>

    <div class="filters">
      <n-input
        v-model:value="query.keyword"
        placeholder="搜索标题或正文，回车搜索"
        clearable
        style="max-width: 260px"
        @keyup.enter="resetAndFetch"
        @clear="resetAndFetch"
      />
      <n-select
        v-model:value="query.difficulty"
        :options="DIFFICULTY_OPTIONS"
        style="width: 130px"
        @update:value="resetAndFetch"
      />
      <n-select
        v-model:value="query.tagId"
        :options="tagOptions"
        style="width: 150px"
        @update:value="resetAndFetch"
      />
      <n-select
        v-model:value="query.status"
        :options="STATUS_OPTIONS"
        style="width: 130px"
        @update:value="resetAndFetch"
      />
      <n-select
        v-model:value="query.sort"
        :options="SORT_OPTIONS"
        style="width: 140px"
        @update:value="resetAndFetch"
      />
      <label class="starred">
        <n-switch v-model:value="query.starredOnly" size="small" @update:value="resetAndFetch" />
        <span>只看收藏</span>
      </label>
    </div>

    <n-spin :show="loading">
      <div v-if="isEmpty" class="empty">
        <n-empty description="还没有笔记">
          <template #extra>
            <n-button size="small" @click="router.push({ name: 'note-new' })">写第一篇</n-button>
          </template>
        </n-empty>
      </div>

      <div v-else class="note-list">
        <NoteCard
          v-for="note in notes"
          :key="note.id"
          :note="note"
          @toggle-star="handleToggleStar"
        />
      </div>
    </n-spin>

    <div v-if="pageCount > 1" class="pager">
      <n-pagination
        :page="query.page"
        :page-count="pageCount"
        :page-slot="7"
        @update:page="handlePageChange"
      />
    </div>
  </div>
</template>

<style scoped>
.list-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 20px;
}

.list-head .page__subtitle {
  margin: 0;
}

.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  padding: 14px 16px;
  margin-bottom: 16px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.starred {
  display: flex;
  gap: 6px;
  align-items: center;
  font-size: 13px;
  color: var(--ln-text-muted);
  cursor: pointer;
}

.note-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.empty {
  padding: 60px 0;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.pager {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}
</style>
