import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

/**
 * 主题模式。
 *
 * - `light` / `dark`：用户明确指定，不再跟随系统
 * - `auto`：跟随操作系统的深色偏好，并在系统切换时实时响应
 */
export type ThemeMode = 'light' | 'dark' | 'auto'

/** 实际生效的外观（auto 解析之后的结果） */
export type ResolvedTheme = 'light' | 'dark'

/** localStorage 的键名。改这里要同步改 index.html 里的内联脚本 */
const STORAGE_KEY = 'leetnote:theme'

const DARK_QUERY = '(prefers-color-scheme: dark)'

function readStoredMode(): ThemeMode {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'light' || saved === 'dark' || saved === 'auto') {
      return saved
    }
  } catch {
    // 隐私模式下 localStorage 可能直接抛异常，忽略即可
  }
  return 'auto'
}

function systemPrefersDark(): boolean {
  return typeof window !== 'undefined' && window.matchMedia(DARK_QUERY).matches
}

export const useThemeStore = defineStore('theme', () => {
  const mode = ref<ThemeMode>(readStoredMode())

  // 系统的深色偏好单独用一个 ref 存，这样 matchMedia 变化时 computed 会重算
  const systemDark = ref(systemPrefersDark())

  const resolved = computed<ResolvedTheme>(() => {
    if (mode.value === 'auto') return systemDark.value ? 'dark' : 'light'
    return mode.value
  })

  const isDark = computed(() => resolved.value === 'dark')

  /**
   * 把主题写到 <html> 上。
   *
   * 加 `.dark` 类而不是直接改样式，是因为 CSS 变量全部挂在 `html.dark` 下，
   * 一次类名切换就能让整站换肤。
   *
   * `theme-switching` 只在切换的那 200ms 挂上：它给所有元素加过渡动画。
   * 如果常驻，平时每一次悬停都会带上颜色过渡，界面会显得黏糊糊的。
   */
  function apply(animate = true): void {
    const root = document.documentElement
    const dark = resolved.value === 'dark'

    if (animate) {
      root.classList.add('theme-switching')
      window.setTimeout(() => root.classList.remove('theme-switching'), 220)
    }

    root.classList.toggle('dark', dark)

    // 让浏览器知道当前配色，原生控件（滚动条、输入框、日期选择器）会跟着变
    root.style.colorScheme = dark ? 'dark' : 'light'
  }

  function setMode(next: ThemeMode): void {
    mode.value = next
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      // 存不进去也不影响本次会话
    }
    apply()
  }

  /** 在 light → dark → auto 之间循环，供单按钮切换使用 */
  function cycle(): void {
    const order: ThemeMode[] = ['auto', 'light', 'dark']
    const next = order[(order.indexOf(mode.value) + 1) % order.length] ?? 'auto'
    setMode(next)
  }

  /**
   * 监听系统配色变化。
   *
   * 只在 `auto` 模式下生效 —— 用户手动选了亮色，系统切成深色时不该跟着变。
   */
  function watchSystem(): void {
    const mq = window.matchMedia(DARK_QUERY)
    mq.addEventListener('change', (e) => {
      systemDark.value = e.matches
      if (mode.value === 'auto') apply()
    })
  }

  /**
   * 启动时调用。
   *
   * 这里【不做】首次应用的动画：页面刚加载完就播一段颜色过渡会显得很突兀。
   * 真正的首屏主题由 index.html 里的内联脚本提前写好，不会闪白。
   */
  function init(): void {
    apply(false)
    watchSystem()
  }

  return { mode, resolved, isDark, setMode, cycle, init }
})
