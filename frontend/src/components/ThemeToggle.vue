<script setup lang="ts">
import { computed, h } from 'vue'
import { NDropdown, NTooltip } from 'naive-ui'
import { useThemeStore, type ThemeMode } from '@/stores/theme'

/**
 * 外观切换。
 *
 * 做成「按钮 + 下拉」而不是单键循环：三个选项一眼可见，
 * 不用点两下才知道自己在哪个模式。按钮上的图标反映当前模式。
 */
const themeStore = useThemeStore()

const MODES: { key: ThemeMode; label: string }[] = [
  { key: 'auto', label: '跟随系统' },
  { key: 'light', label: '浅色' },
  { key: 'dark', label: '深色' },
]

/** 下拉里给当前生效的模式打勾 */
const CheckIcon = () =>
  h(
    'svg',
    {
      viewBox: '0 0 24 24',
      width: 15,
      height: 15,
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': 2.4,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
    },
    [h('path', { d: 'M4.5 12.5l5 5 10-11' })],
  )

const options = computed(() =>
  MODES.map((m) => ({
    key: m.key,
    label: m.label,
    icon: m.key === themeStore.mode ? CheckIcon : undefined,
  })),
)

const currentLabel = computed(
  () => MODES.find((m) => m.key === themeStore.mode)?.label ?? '跟随系统',
)

function handleSelect(key: string): void {
  themeStore.setMode(key as ThemeMode)
}
</script>

<template>
  <n-dropdown :options="options" trigger="click" placement="bottom-end" @select="handleSelect">
    <n-tooltip trigger="hover" placement="bottom" :delay="400">
      <template #trigger>
        <button class="theme-toggle" type="button" :aria-label="`外观：${currentLabel}`">
          <!-- 图标全部用内联 SVG：不引图标库，体积为零，任意尺寸都清晰 -->

          <!-- 浅色：太阳 -->
          <svg
            v-if="themeStore.mode === 'light'"
            class="theme-toggle__icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
            stroke-linecap="round"
          >
            <circle cx="12" cy="12" r="4.1" />
            <path
              d="M12 2.6v2.2M12 19.2v2.2M2.6 12h2.2M19.2 12h2.2M5.4 5.4l1.6 1.6M17 17l1.6 1.6M18.6 5.4L17 7M7 17l-1.6 1.6"
            />
          </svg>

          <!-- 深色：月牙 -->
          <svg
            v-else-if="themeStore.mode === 'dark'"
            class="theme-toggle__icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M20.4 14.4A8.6 8.6 0 1 1 9.6 3.6a6.9 6.9 0 0 0 10.8 10.8Z" />
          </svg>

          <!-- 跟随系统：半明半暗的圆 -->
          <svg
            v-else
            class="theme-toggle__icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
          >
            <circle cx="12" cy="12" r="8.4" />
            <path d="M12 3.6v16.8a8.4 8.4 0 0 0 0-16.8Z" fill="currentColor" stroke="none" />
          </svg>
        </button>
      </template>
      外观：{{ currentLabel }}
    </n-tooltip>
  </n-dropdown>
</template>

<style scoped>
.theme-toggle {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  padding: 0;
  color: var(--ln-text-muted);
  cursor: pointer;
  background: none;
  border: 0;
  border-radius: 8px;
}

.theme-toggle:hover {
  color: var(--ln-text);
  background: var(--ln-surface-active);
}

.theme-toggle__icon {
  width: 18px;
  height: 18px;
}
</style>
