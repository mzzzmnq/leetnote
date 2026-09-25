<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import { RouterLink } from 'vue-router'
import DifficultyTag from './DifficultyTag.vue'
import type { NoteListItem } from '@/api/types'
import { formatRelative } from '@/utils/format'

const props = defineProps<{ note: NoteListItem }>()
const emit = defineEmits<{ toggleStar: [id: number] }>()

const problemLabel = computed(() => {
  const p = props.note.problem
  if (!p) return null
  return p.leetcode_id ? `${p.leetcode_id}. ${p.title}` : p.title
})
</script>

<template>
  <article class="note-card">
    <div class="note-card__main">
      <div class="note-card__head">
        <RouterLink :to="{ name: 'note-detail', params: { id: note.id } }" class="note-card__title">
          {{ note.title }}
        </RouterLink>
        <n-tag v-if="note.status === 'draft'" size="small" :bordered="false" type="default">
          草稿
        </n-tag>
      </div>

      <div v-if="problemLabel" class="note-card__problem">
        <DifficultyTag :difficulty="note.problem!.difficulty" />
        <span>{{ problemLabel }}</span>
      </div>

      <p v-if="note.summary" class="note-card__summary">{{ note.summary }}</p>

      <div class="note-card__meta">
        <n-tag
          v-for="tag in note.tags"
          :key="tag.id"
          size="small"
          :bordered="false"
          type="info"
        >
          {{ tag.name }}
        </n-tag>
        <span v-if="note.solution_count > 0" class="note-card__solutions">
          {{ note.solution_count }} 个解法
        </span>
        <span class="note-card__date">{{ formatRelative(note.updated_at) }}</span>
      </div>
    </div>

    <button
      class="note-card__star"
      :class="{ 'note-card__star--on': note.is_starred }"
      :title="note.is_starred ? '取消收藏' : '收藏'"
      type="button"
      @click="emit('toggleStar', note.id)"
    >
      {{ note.is_starred ? '★' : '☆' }}
    </button>
  </article>
</template>

<style scoped>
.note-card {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 18px 20px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 10px;
  transition: border-color 0.15s;
}

.note-card:hover {
  border-color: #cbd5e1;
}

.note-card__main {
  flex: 1;
  min-width: 0;
}

.note-card__head {
  display: flex;
  gap: 8px;
  align-items: center;
}

.note-card__title {
  font-size: 16px;
  font-weight: 600;
  color: var(--ln-text);
  text-decoration: none;
}

.note-card__title:hover {
  color: var(--ln-primary);
  text-decoration: none;
}

.note-card__problem {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-top: 6px;
  font-size: 13px;
  color: var(--ln-text-muted);
}

.note-card__summary {
  margin: 8px 0 0;
  font-size: 14px;
  color: var(--ln-text-muted);
}

.note-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-top: 10px;
  font-size: 12px;
  color: var(--ln-text-muted);
}

.note-card__solutions::before,
.note-card__date::before {
  margin-right: 8px;
  content: '·';
}

.note-card__star {
  padding: 0 4px;
  font-size: 20px;
  line-height: 1;
  color: #cbd5e1;
  cursor: pointer;
  background: none;
  border: 0;
}

.note-card__star:hover {
  color: #f59e0b;
}

.note-card__star--on {
  color: #f59e0b;
}
</style>
