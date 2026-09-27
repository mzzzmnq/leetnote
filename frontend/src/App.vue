<script setup lang="ts">
import { computed } from 'vue'
import {
  NConfigProvider,
  NDialogProvider,
  NLoadingBarProvider,
  NMessageProvider,
  darkTheme,
  dateZhCN,
  zhCN,
} from 'naive-ui'
import { useThemeStore } from '@/stores/theme'
import { darkThemeOverrides, lightThemeOverrides } from '@/styles/naive-theme'

/**
 * Naive UI 的亮/暗主题由这里统一注入。
 *
 * 我们自己的界面用的是 CSS 变量（见 styles/main.css），
 * Naive 的组件用的是它自己的主题系统 —— 两边必须同步切换，
 * 否则会出现「页面变暗了但弹窗还是白的」这种割裂感。
 *
 * 【关于体积】`darkTheme` 是一个覆盖全库组件的大对象，约 65KB（gzip 后约 13KB），
 * 且无法 tree-shaking。这是 Naive UI 暗色模式的标准代价，换来的是所有组件
 * 开箱即用的暗色适配。相比懒加载省下的十几 KB，不值得让首次切暗色闪一下。
 */
const themeStore = useThemeStore()

const theme = computed(() => (themeStore.isDark ? darkTheme : null))
const themeOverrides = computed(() =>
  themeStore.isDark ? darkThemeOverrides : lightThemeOverrides,
)
</script>

<template>
  <n-config-provider
    :theme="theme"
    :theme-overrides="themeOverrides"
    :locale="zhCN"
    :date-locale="dateZhCN"
  >
    <n-loading-bar-provider>
      <n-message-provider>
        <n-dialog-provider>
          <router-view />
        </n-dialog-provider>
      </n-message-provider>
    </n-loading-bar-provider>
  </n-config-provider>
</template>
