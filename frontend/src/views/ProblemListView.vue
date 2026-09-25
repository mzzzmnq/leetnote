<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NPagination,
  NSelect,
  NSpin,
  useDialog,
  useMessage,
} from 'naive-ui'
import { ApiError } from '@/api/client'
import { createProblem, deleteProblem, listProblems, updateProblem } from '@/api/problems'
import type { Difficulty, Problem, ProblemInput } from '@/api/types'
import DifficultyTag from '@/components/DifficultyTag.vue'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const problems = ref<Problem[]>([])
const total = ref(0)
const pageCount = ref(0)

const DIFFICULTY_OPTIONS = [
  { label: '全部难度', value: '' },
  { label: '简单', value: 'Easy' },
  { label: '中等', value: 'Medium' },
  { label: '困难', value: 'Hard' },
]

const query = reactive({
  keyword: '',
  difficulty: '' as Difficulty | '',
  page: 1,
  size: 15,
})

// ---------- 新建 / 编辑弹窗 ----------
const modalVisible = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const form = reactive<ProblemInput>({
  leetcode_id: null,
  title: '',
  title_slug: '',
  difficulty: 'Easy',
  url: null,
})

async function fetchProblems(): Promise<void> {
  loading.value = true
  try {
    const data = await listProblems({
      keyword: query.keyword.trim() || undefined,
      difficulty: query.difficulty || undefined,
      page: query.page,
      size: query.size,
    })
    problems.value = data.items
    total.value = data.total
    pageCount.value = data.pages
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '加载题目失败')
  } finally {
    loading.value = false
  }
}

function resetAndFetch(): void {
  query.page = 1
  void fetchProblems()
}

function openCreate(): void {
  editingId.value = null
  form.leetcode_id = null
  form.title = ''
  form.title_slug = ''
  form.difficulty = 'Easy'
  form.url = null
  modalVisible.value = true
}

function openEdit(problem: Problem): void {
  editingId.value = problem.id
  form.leetcode_id = problem.leetcode_id
  form.title = problem.title
  form.title_slug = problem.title_slug
  form.difficulty = problem.difficulty
  form.url = problem.url
  modalVisible.value = true
}

async function handleSave(): Promise<void> {
  if (!form.title.trim() || !form.title_slug.trim()) {
    message.warning('标题与 slug 都不能为空')
    return
  }

  saving.value = true
  try {
    const payload: ProblemInput = {
      leetcode_id: form.leetcode_id ?? null,
      title: form.title.trim(),
      title_slug: form.title_slug.trim(),
      difficulty: form.difficulty,
      url: form.url?.trim() || null,
    }

    if (editingId.value !== null) {
      await updateProblem(editingId.value, payload)
      message.success('已更新')
    } else {
      await createProblem(payload)
      message.success('已创建')
    }

    modalVisible.value = false
    await fetchProblems()
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

function handleDelete(problem: Problem): void {
  dialog.warning({
    title: '删除题目',
    content: `确定删除「${problem.title}」吗？关联的笔记会保留，只是不再关联该题目。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteProblem(problem.id)
        message.success('已删除')
        await fetchProblems()
      } catch (error) {
        message.error(error instanceof ApiError ? error.message : '删除失败')
      }
    },
  })
}

onMounted(fetchProblems)
</script>

<template>
  <div class="page">
    <header class="list-head">
      <div>
        <h1 class="page__title">题目库</h1>
        <p class="page__subtitle">共 {{ total }} 道题。题目是全局共享的元数据。</p>
      </div>
      <n-button type="primary" @click="openCreate">新增题目</n-button>
    </header>

    <div class="filters">
      <n-input
        v-model:value="query.keyword"
        placeholder="搜索标题或 slug，回车搜索"
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
    </div>

    <n-spin :show="loading">
      <div v-if="!loading && problems.length === 0" class="empty">
        <n-empty description="还没有题目">
          <template #extra>
            <n-button size="small" @click="openCreate">新增一道</n-button>
          </template>
        </n-empty>
      </div>

      <div v-else class="rows">
        <div v-for="p in problems" :key="p.id" class="row">
          <div class="row__main">
            <div class="row__title">
              <span v-if="p.leetcode_id" class="row__no">{{ p.leetcode_id }}</span>
              <a v-if="p.url" :href="p.url" target="_blank" rel="noopener noreferrer">
                {{ p.title }}
              </a>
              <span v-else>{{ p.title }}</span>
            </div>
            <code class="row__slug">{{ p.title_slug }}</code>
          </div>

          <DifficultyTag :difficulty="p.difficulty" />

          <div class="row__actions">
            <n-button size="tiny" quaternary @click="openEdit(p)">编辑</n-button>
            <n-button size="tiny" quaternary type="error" @click="handleDelete(p)">删除</n-button>
          </div>
        </div>
      </div>
    </n-spin>

    <div v-if="pageCount > 1" class="pager">
      <n-pagination
        :page="query.page"
        :page-count="pageCount"
        @update:page="(p: number) => { query.page = p; fetchProblems() }"
      />
    </div>

    <n-modal
      v-model:show="modalVisible"
      preset="card"
      :title="editingId !== null ? '编辑题目' : '新增题目'"
      style="max-width: 480px"
    >
      <n-form label-placement="top">
        <n-form-item label="题号（可选）">
          <n-input-number
            v-model:value="form.leetcode_id"
            placeholder="如 1"
            :min="1"
            style="width: 100%"
          />
        </n-form-item>
        <n-form-item label="标题">
          <n-input v-model:value="form.title" placeholder="如 Two Sum" maxlength="200" />
        </n-form-item>
        <n-form-item label="Slug">
          <n-input v-model:value="form.title_slug" placeholder="如 two-sum" maxlength="200" />
        </n-form-item>
        <n-form-item label="难度">
          <n-select v-model:value="form.difficulty" :options="DIFFICULTY_OPTIONS.slice(1)" />
        </n-form-item>
        <n-form-item label="链接（可选）">
          <n-input v-model:value="form.url" placeholder="https://leetcode.cn/problems/..." />
        </n-form-item>
      </n-form>

      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 8px">
          <n-button @click="modalVisible = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
        </div>
      </template>
    </n-modal>
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
  gap: 10px;
  padding: 14px 16px;
  margin-bottom: 16px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.empty {
  padding: 60px 0;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.rows {
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.row {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 12px 18px;
  border-bottom: 1px solid var(--ln-border);
}

.row:last-child {
  border-bottom: 0;
}

.row__main {
  flex: 1;
  min-width: 0;
}

.row__title {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 14px;
}

.row__no {
  min-width: 32px;
  font-size: 12px;
  color: var(--ln-text-muted);
}

.row__slug {
  font-size: 12px;
  color: var(--ln-text-muted);
}

.row__actions {
  display: flex;
  gap: 2px;
  opacity: 0;
}

.row:hover .row__actions {
  opacity: 1;
}

.pager {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}
</style>
