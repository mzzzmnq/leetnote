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
      type="button"
      :title="note.is_starred ? '取消收藏' : '收藏'"
      :aria-label="note.is_starred ? '取消收藏' : '收藏'"
      :aria-pressed="note.is_starred"
      @click="emit('toggleStar', note.id)"
    >
      <!-- 用 SVG 而不是 ★/☆ 字符：字符在不同平台上的字形、粗细、基线都不一样，
           SVG 能保证各处渲染完全一致，也便于精确控制描边与填充。 -->
      <svg
        class="note-card__star-icon"
        viewBox="0 0 24 24"
        :fill="note.is_starred ? 'currentColor' : 'none'"
        stroke="currentColor"
        stroke-width="1.6"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path
          d="M12 3.4l2.63 5.33 5.87.86-4.25 4.14 1 5.87L12 16.87l-5.25 2.73 1-5.87L3.5 9.59l5.87-.86z"
        />
      </svg>
    </button>
  </article>
</template>

<style scoped>
.note-card {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 18px 20px;
  background: var(--ln-surface);
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius);
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
}

.note-card:hover {
  border-color: var(--ln-border-strong);
  box-shadow: var(--ln-shadow-sm);
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
  color: var(--ln-primary-text);
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
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  padding: 0;
  color: var(--ln-text-subtle);
  cursor: pointer;
  background: none;
  border: 0;
  border-radius: 6px;
  transition:
    color 0.15s,
    background-color 0.15s;
}

.note-card__star:hover {
  color: var(--ln-star);
  background: var(--ln-surface-active);
}

.note-card__star--on {
  color: var(--ln-star);
}

.note-card__star-icon {
  width: 18px;
  height: 18px;
}
</style>
