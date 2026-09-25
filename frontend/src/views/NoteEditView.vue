<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import {
  NButton,
  NCard,
  NInput,
  NSelect,
  NSpace,
  NSpin,
  useDialog,
  useMessage,
} from 'naive-ui'
import { ApiError } from '@/api/client'
import { createNote, getNote, updateNote } from '@/api/notes'
import { listProblems } from '@/api/problems'
import { listTags } from '@/api/tags'
import type { NoteInput, SolutionInput, Tag } from '@/api/types'
import MarkdownViewer from '@/components/MarkdownViewer.vue'
import SolutionEditor from '@/components/SolutionEditor.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const noteId = route.params.id ? Number(route.params.id) : null
const isEdit = noteId !== null

const loading = ref(false)
const saving = ref(false)
const saved = ref(false)
const showPreview = ref(true)

const allTags = ref<Tag[]>([])
const problemOptions = ref<{ label: string; value: number }[]>([])
const problemLoading = ref(false)

const form = reactive<NoteInput>({
  problem_id: null,
  title: '',
  content_md: '',
  summary: null,
  status: 'published',
  is_starred: false,
  tag_ids: [],
  solutions: [],
})

const solutions = ref<SolutionInput[]>([])

const statusOptions = [
  { label: '已发布', value: 'published' },
  { label: '草稿', value: 'draft' },
]

const tagOptions = computed(() => allTags.value.map((t) => ({ label: t.name, value: t.id })))

const hasContent = computed(
  () => form.title.trim() !== '' || form.content_md.trim() !== '' || solutions.value.length > 0,
)

// ---------------------------------------------------------------
// 题目：远程搜索
// ---------------------------------------------------------------
async function searchProblems(keyword = ''): Promise<void> {
  problemLoading.value = true
  try {
    const data = await listProblems({ keyword: keyword.trim() || undefined, size: 30 })
    problemOptions.value = data.items.map((p) => ({
      label: p.leetcode_id ? `${p.leetcode_id}. ${p.title}` : p.title,
      value: p.id,
    }))
  } catch {
    // 搜索失败不阻塞编辑，静默降级
  } finally {
    problemLoading.value = false
  }
}

// ---------------------------------------------------------------
// 解法
// ---------------------------------------------------------------
function addSolution(): void {
  solutions.value = [
    ...solutions.value,
    { title: '', language: 'go', code: '', time_complexity: null, space_complexity: null },
  ]
}

function updateSolution(index: number, value: SolutionInput): void {
  const next = [...solutions.value]
  next[index] = value
  solutions.value = next
}

function removeSolution(index: number): void {
  solutions.value = solutions.value.filter((_, i) => i !== index)
}

// ---------------------------------------------------------------
// 加载与保存
// ---------------------------------------------------------------
async function load(): Promise<void> {
  loading.value = true
  try {
    allTags.value = await listTags()
    await searchProblems()

    if (noteId !== null) {
      const note = await getNote(noteId)
      form.problem_id = note.problem_id
      form.title = note.title
      form.content_md = note.content_md
      form.summary = note.summary
      form.status = note.status
      form.is_starred = note.is_starred
      form.tag_ids = note.tags.map((t) => t.id)

      solutions.value = note.solutions.map((s) => ({
        title: s.title,
        language: s.language,
        code: s.code,
        time_complexity: s.time_complexity,
        space_complexity: s.space_complexity,
      }))

      // 编辑已有笔记时，把它的题目也加进候选，避免下拉里找不到
      if (note.problem && !problemOptions.value.some((o) => o.value === note.problem!.id)) {
        problemOptions.value = [
          {
            label: note.problem.leetcode_id
              ? `${note.problem.leetcode_id}. ${note.problem.title}`
              : note.problem.title,
            value: note.problem.id,
          },
          ...problemOptions.value,
        ]
      }
    }
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function handleSave(): Promise<void> {
  if (!form.title.trim()) {
    message.warning('标题不能为空')
    return
  }

  // 过滤掉完全空白的解法，避免误提交空行
  const cleanedSolutions = solutions.value.filter((s) => s.title.trim() || s.code.trim())

  saving.value = true
  try {
    const payload: NoteInput = {
      ...form,
      title: form.title.trim(),
      summary: form.summary?.trim() || null,
      tag_ids: form.tag_ids ?? [],
      solutions: cleanedSolutions,
    }

    const note = isEdit && noteId !== null
      ? await updateNote(noteId, payload)
      : await createNote(payload)

    saved.value = true
    message.success(isEdit ? '已保存' : '已创建')
    await router.replace({ name: 'note-detail', params: { id: note.id } })
  } catch (error) {
    if (error instanceof ApiError) {
      message.error(error.message)
    } else {
      message.error('保存失败')
    }
  } finally {
    saving.value = false
  }
}

// 有未保存内容时离开页面给个提醒，避免手滑丢失编辑
onBeforeRouteLeave(async () => {
  if (saved.value || !hasContent.value) {
    return true
  }

  return await new Promise<boolean>((resolve) => {
    dialog.warning({
      title: '放弃编辑？',
      content: '当前改动尚未保存，离开后将丢失。',
      positiveText: '放弃',
      negativeText: '继续编辑',
      onPositiveClick: () => resolve(true),
      onNegativeClick: () => resolve(false),
      onClose: () => resolve(false),
    })
  })
})

onMounted(load)
</script>

<template>
  <div class="page">
    <n-spin :show="loading">
      <header class="edit-head">
        <h1 class="page__title">{{ isEdit ? '编辑笔记' : '写笔记' }}</h1>
        <n-space>
          <n-button @click="router.back()">取消</n-button>
          <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
        </n-space>
      </header>

      <n-card size="small" class="edit-meta">
        <n-input
          v-model:value="form.title"
          placeholder="笔记标题"
          maxlength="200"
          show-count
          size="large"
        />

        <div class="edit-meta__row">
          <n-select
            v-model:value="form.problem_id"
            :options="problemOptions"
            :loading="problemLoading"
            filterable
            remote
            clearable
            placeholder="关联题目（可搜索，可留空）"
            style="flex: 1"
            @search="searchProblems"
          />
          <n-select
            v-model:value="form.status"
            :options="statusOptions"
            style="width: 120px"
          />
        </div>

        <div class="edit-meta__row">
          <n-select
            v-model:value="form.tag_ids"
            :options="tagOptions"
            multiple
            filterable
            clearable
            placeholder="选择标签"
            style="flex: 1"
          />
        </div>

        <n-input
          v-model:value="form.summary"
          placeholder="一句话摘要（可选，显示在列表页）"
          maxlength="500"
          clearable
        />
      </n-card>

      <section class="editor">
        <header class="editor__head">
          <span class="editor__label">正文（Markdown）</span>
          <n-button size="tiny" quaternary @click="showPreview = !showPreview">
            {{ showPreview ? '隐藏预览' : '显示预览' }}
          </n-button>
        </header>

        <div class="editor__panes" :class="{ 'editor__panes--single': !showPreview }">
          <n-input
            v-model:value="form.content_md"
            type="textarea"
            placeholder="支持 Markdown。用 ```go 包裹代码块会高亮。"
            class="editor__input"
            :autosize="{ minRows: 20, maxRows: 40 }"
          />

          <div v-if="showPreview" class="editor__preview">
            <MarkdownViewer v-if="form.content_md" :source="form.content_md" />
            <p v-else class="editor__placeholder">预览区</p>
          </div>
        </div>
      </section>

      <section class="solutions-edit">
        <header class="solutions-edit__head">
          <h2 class="solutions-edit__title">解法（{{ solutions.length }}）</h2>
          <n-button size="small" @click="addSolution">添加解法</n-button>
        </header>

        <SolutionEditor
          v-for="(sol, index) in solutions"
          :key="index"
          :model-value="sol"
          :title="`解法 ${index + 1}`"
          @update:model-value="(v) => updateSolution(index, v)"
          @remove="removeSolution(index)"
        />

        <p v-if="solutions.length === 0" class="solutions-edit__empty">
          还没有解法。可以记录同一道题的多种思路（暴力 / 优化 / 最优），方便日后复习对比。
        </p>
      </section>
    </n-spin>
  </div>
</template>

<style scoped>
.edit-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.edit-head .page__title {
  margin: 0;
}

.edit-meta {
  margin-bottom: 16px;
}

.edit-meta__row {
  display: flex;
  gap: 10px;
  margin-top: 12px;
}

.editor {
  padding: 16px;
  margin-bottom: 16px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.editor__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.editor__label {
  font-size: 13px;
  font-weight: 600;
  color: var(--ln-text-muted);
}

.editor__panes {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.editor__panes--single {
  grid-template-columns: 1fr;
}

.editor__preview {
  padding: 14px 16px;
  overflow-y: auto;
  background: var(--ln-bg);
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  max-height: 620px;
}

.editor__placeholder {
  margin: 0;
  color: var(--ln-text-muted);
}

.solutions-edit {
  padding: 16px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.solutions-edit__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.solutions-edit__title {
  margin: 0;
  font-size: 16px;
}

.solutions-edit__empty {
  margin: 0;
  font-size: 13px;
  color: var(--ln-text-muted);
}

@media (max-width: 900px) {
  .editor__panes {
    grid-template-columns: 1fr;
  }
}
</style>
