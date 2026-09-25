<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NInput, NSpin, NTag, useMessage } from 'naive-ui'
import { ApiError } from '@/api/client'
import { toggleStar } from '@/api/notes'
import { search } from '@/api/search'
import type { NoteListItem, Problem } from '@/api/types'
import DifficultyTag from '@/components/DifficultyTag.vue'
import NoteCard from '@/components/NoteCard.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()

const keyword = ref(typeof route.query.q === 'string' ? route.query.q : '')
const loading = ref(false)
const searched = ref(false)
const notes = ref<NoteListItem[]>([])
const problems = ref<Problem[]>([])

let debounceTimer: ReturnType<typeof setTimeout> | null = null

const isEmpty = ref(false)

async function runSearch(raw: string): Promise<void> {
  const q = raw.trim()

  // 同步到 URL，这样能直接分享/刷新搜索结果
  if (q) {
    void router.replace({ name: 'search', query: { q } })
  } else {
    void router.replace({ name: 'search' })
  }

  if (!q) {
    notes.value = []
    problems.value = []
    searched.value = false
    isEmpty.value = false
    return
  }

  loading.value = true
  try {
    const result = await search({ q })
    notes.value = result.notes
    problems.value = result.problems
    isEmpty.value = result.notes.length === 0 && result.problems.length === 0
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '搜索失败')
  } finally {
    loading.value = false
    searched.value = true
  }
}

// 输入防抖：每敲一个字都请求会打出一堆无用查询，
// 而且响应乱序时可能把旧结果覆盖到新结果上。
watch(keyword, (value) => {
  if (debounceTimer !== null) {
    clearTimeout(debounceTimer)
  }
  debounceTimer = setTimeout(() => {
    void runSearch(value)
  }, 300)
})

async function handleToggleStar(id: number): Promise<void> {
  try {
    const updated = await toggleStar(id)
    const target = notes.value.find((n) => n.id === id)
    if (target) target.is_starred = updated.is_starred
  } catch (error) {
    message.error(error instanceof ApiError ? error.message : '操作失败')
  }
}

onMounted(() => {
  if (keyword.value.trim()) {
    void runSearch(keyword.value)
  }
})

onBeforeUnmount(() => {
  if (debounceTimer !== null) {
    clearTimeout(debounceTimer)
  }
})
</script>

<template>
  <div class="page">
    <h1 class="page__title">搜索</h1>
    <p class="page__subtitle">同时检索你的笔记与题目库</p>

    <n-input
      v-model:value="keyword"
      size="large"
      clearable
      placeholder="输入关键词，如「哈希表」「滑动窗口」「two sum」"
      class="search-box"
    />

    <n-spin :show="loading">
      <div v-if="!searched" class="state">输入关键词开始搜索</div>

      <div v-else-if="isEmpty" class="state">
        没有找到与「{{ keyword }}」相关的内容
      </div>

      <template v-else>
        <section v-if="notes.length" class="group">
          <h2 class="group__title">
            笔记
            <span class="group__count">{{ notes.length }}</span>
          </h2>
          <div class="group__list">
            <NoteCard
              v-for="note in notes"
              :key="note.id"
              :note="note"
              @toggle-star="handleToggleStar"
            />
          </div>
        </section>

        <section v-if="problems.length" class="group">
          <h2 class="group__title">
            题目
            <span class="group__count">{{ problems.length }}</span>
          </h2>
          <div class="group__list">
            <div v-for="p in problems" :key="p.id" class="problem-row">
              <span v-if="p.leetcode_id" class="problem-row__no">{{ p.leetcode_id }}</span>
              <a
                v-if="p.url"
                :href="p.url"
                target="_blank"
                rel="noopener noreferrer"
                class="problem-row__title"
              >
                {{ p.title }}
              </a>
              <span v-else class="problem-row__title">{{ p.title }}</span>
              <DifficultyTag :difficulty="p.difficulty" />
              <n-tag size="small" :bordered="false" type="default">
                {{ p.title_slug }}
              </n-tag>
              <n-button
                size="tiny"
                quaternary
                style="margin-left: auto"
                @click="router.push({ name: 'note-new', query: { problem: p.id } })"
              >
                写笔记
              </n-button>
            </div>
          </div>
        </section>
      </template>
    </n-spin>
  </div>
</template>

<style scoped>
.search-box {
  margin-bottom: 20px;
}

.state {
  padding: 60px 0;
  font-size: 14px;
  color: var(--ln-text-muted);
  text-align: center;
}

.group {
  margin-bottom: 28px;
}

.group__title {
  display: flex;
  gap: 8px;
  align-items: center;
  margin: 0 0 12px;
  font-size: 16px;
}

.group__count {
  padding: 1px 8px;
  font-size: 12px;
  font-weight: 400;
  color: var(--ln-text-muted);
  background: var(--ln-bg);
  border-radius: 10px;
}

.group__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.problem-row {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 12px 18px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
}

.problem-row__no {
  min-width: 32px;
  font-size: 12px;
  color: var(--ln-text-muted);
}

.problem-row__title {
  font-size: 14px;
  color: var(--ln-text);
}
</style>
