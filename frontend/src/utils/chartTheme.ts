import * as echarts from 'echarts/core'

/**
 * ECharts 主题。
 *
 * 【为什么必须自己写】ECharts 的默认主题是"深色文字 + 浅色网格"，
 * 直接用在暗色页面上，坐标轴文字和分割线几乎看不见。
 * 而 `echarts/core` 不会自动注册内置的 dark 主题（那是完整包 `echarts` 才做的），
 * 所以这里按我们的设计令牌手工定义两套并注册。
 *
 * 色值要和 `main.css` 里的 CSS 变量对应，改一处要同步改另一处。
 */

export type ChartThemeName = 'leetnote-light' | 'leetnote-dark'

/** 主题名与当前是否暗色一一对应 */
export function chartThemeName(isDark: boolean): ChartThemeName {
  return isDark ? 'leetnote-dark' : 'leetnote-light'
}

/**
 * 语义色（主色 / 成功 / 警告 / 危险）。
 *
 * 图表里表示"难度"这类含义时要用语义色而不是调色板顺序色 ——
 * 否则"简单"可能被画成红色，读图的人会误解。
 *
 * 取值与 `main.css` 的同名变量、以及难度标签的配色保持一致，
 * 这样"难度分布"饼图和列表里的"简单/中等/困难"标签是同一套颜色。
 */
export function chartColors(isDark: boolean) {
  return isDark
    ? { primary: '#6b9bff', success: '#4ec97a', warning: '#e0a53c', danger: '#f0736a' }
    : { primary: '#2f6fed', success: '#0f7237', warning: '#96530a', danger: '#b02c2c' }
}

interface Palette {
  /** 分类调色板：多个 series 依次取色 */
  series: string[]
  text: string
  textMuted: string
  border: string
  tooltipBg: string
  tooltipBorder: string
}

const LIGHT: Palette = {
  series: ['#2f6fed', '#0f7237', '#a35c00', '#c03434', '#7c4ddb', '#0369a1'],
  text: '#1a1f2b',
  textMuted: '#5a6472',
  border: '#e4e7ec',
  tooltipBg: '#ffffff',
  tooltipBorder: '#e4e7ec',
}

const DARK: Palette = {
  // 暗底上色相要提亮、饱和度略降，否则会显得脏
  series: ['#6b9bff', '#4ec97a', '#e0a53c', '#f0736a', '#a78bfa', '#38bdf8'],
  text: '#e6edf3',
  textMuted: '#9aa7b4',
  border: '#262d38',
  tooltipBg: '#1c222b',
  tooltipBorder: '#3a4452',
}

function buildTheme(p: Palette): Record<string, unknown> {
  // 坐标轴共用一套样式，避免 categoryAxis / valueAxis 写两遍
  const axis = {
    axisLine: { show: true, lineStyle: { color: p.border } },
    axisTick: { show: false },
    axisLabel: { color: p.textMuted },
    // 只留横向分割线（值轴），分类轴的分割线会和网格叠在一起显得很乱
    splitLine: { show: true, lineStyle: { color: p.border, type: 'dashed' } },
  }

  return {
    // 透明背景：让图表融进卡片，而不是自带一块底色
    backgroundColor: 'transparent',
    color: p.series,

    textStyle: { color: p.textMuted, fontFamily: 'inherit' },

    title: {
      textStyle: { color: p.text, fontWeight: 600 },
      subtextStyle: { color: p.textMuted },
    },

    legend: {
      textStyle: { color: p.textMuted },
      icon: 'roundRect',
      itemWidth: 10,
      itemHeight: 10,
    },

    tooltip: {
      backgroundColor: p.tooltipBg,
      borderColor: p.tooltipBorder,
      borderWidth: 1,
      textStyle: { color: p.text, fontSize: 13 },
      // 默认的黑色阴影在暗色下看不出层次，直接用更柔和的
      extraCssText: 'border-radius: 8px; box-shadow: 0 4px 16px rgb(0 0 0 / 18%);',
      axisPointer: {
        lineStyle: { color: p.border },
        crossStyle: { color: p.border },
      },
    },

    categoryAxis: { ...axis, splitLine: { show: false } },
    valueAxis: axis,

    line: {
      symbol: 'circle',
      symbolSize: 6,
      smooth: true,
      lineStyle: { width: 2 },
      itemStyle: { borderWidth: 0 },
    },

    pie: {
      itemStyle: { borderColor: 'transparent', borderWidth: 0 },
      label: { color: p.textMuted },
    },
  }
}

// 模块加载时注册一次即可，重复注册会覆盖
echarts.registerTheme('leetnote-light', buildTheme(LIGHT))
echarts.registerTheme('leetnote-dark', buildTheme(DARK))
