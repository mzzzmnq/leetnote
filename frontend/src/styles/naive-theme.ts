import type { GlobalThemeOverrides } from 'naive-ui'

/**
 * Naive UI 的主题适配。
 *
 * Naive 自带的亮/暗主题已经不错，但主色、语义色和圆角和我们的设计令牌不一致 ——
 * 不覆盖的话，按钮是 Naive 的蓝、链接是我们的蓝，看着像两个网站拼起来的。
 *
 * 【重要】这里的色值必须与 `main.css` 里的 CSS 变量保持一致，
 * 改一处要同步改另一处（CSS 变量不能直接被 JS 读取，只能两边都写）。
 *
 * 另外这里顺手修掉了 Naive 默认色的一批对比度问题（都按 WCAG AA 实测过）：
 *   - 占位符 #c2c2c2 → #6b7280   （1.8:1 → 4.8:1）
 *   - 标签 info  #2080f0 → #1f5bc4（3.4:1 → 5.5:1）
 *   - 标签 warning #f0a020 → #a35c00（1.9:1 → 4.6:1）
 *   - 头像白字压在 #ccc 上 → 换成主色底（1.6:1 → 4.6:1）
 */

/** 与 main.css 的 font-family 完全一致，避免组件内外字体不一致 */
const FONT_FAMILY = [
  'system-ui',
  '-apple-system',
  'Segoe UI',
  'PingFang SC',
  'Hiragino Sans GB',
  'Microsoft YaHei',
  'sans-serif',
].join(', ')

/** 亮暗共用的部分 */
const SHARED: GlobalThemeOverrides['common'] = {
  fontFamily: FONT_FAMILY,
  fontSize: '14px',
  borderRadius: '8px',
  borderRadiusSmall: '6px',
}

export const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    ...SHARED,
    primaryColor: '#2f6fed',
    primaryColorHover: '#2560d4',
    primaryColorPressed: '#1d4fb5',
    primaryColorSuppl: '#2560d4',

    // 语义色。Naive 会拿它们同时当文字色和 12% 浅底，
    // 所以必须选"当文字也够深"的版本，否则标签文字会糊在底色里。
    infoColor: '#1f5bc4',
    successColor: '#0f7237',
    warningColor: '#96530a',
    errorColor: '#b02c2c',

    placeholderColor: '#6b7280',

    // 卡片、弹窗的底色对齐 --ln-surface
    cardColor: '#ffffff',
    modalColor: '#ffffff',
    popoverColor: '#ffffff',
    tableColor: '#ffffff',
    inputColor: '#ffffff',

    bodyColor: '#f6f7f9',
    textColorBase: '#1a1f2b',
    textColor1: '#1a1f2b',
    textColor2: '#5a6472',
    textColor3: '#6b7280',
    borderColor: '#e4e7ec',
    dividerColor: '#e4e7ec',
  },
  // 头像的文字是写死的白色，所以底色必须够深。
  // Naive 默认用 #ccc 那种浅灰，白字压上去只有 1.6:1。
  Avatar: {
    color: '#2f6fed',
  },
  // n-empty 的说明文字默认取 textColorDisabled（很浅的灰，1.8:1）。
  // 空状态提示是【正常内容】而不是"禁用控件"，不该用禁用色，单独覆盖掉。
  Empty: {
    textColor: '#5a6472',
    iconColor: '#6b7280',
  },
  // 分页「当前页」的页码默认用 primaryColor，压在页面底色（#f6f7f9）上
  // 只有 4.24:1。换成专供文字用的深主色。
  Pagination: {
    itemTextColorActive: '#1f56c4',
    itemBorderActive: '1px solid #1f56c4',
  },
}

export const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    ...SHARED,
    // 暗底上主色要提亮，否则会糊在背景里
    primaryColor: '#6b9bff',
    primaryColorHover: '#8bb2ff',
    primaryColorPressed: '#5a8af0',
    primaryColorSuppl: '#8bb2ff',

    infoColor: '#6b9bff',
    successColor: '#4ec97a',
    warningColor: '#e0a53c',
    errorColor: '#f0736a',

    placeholderColor: '#7d8a98',

    cardColor: '#161b22',
    modalColor: '#161b22',
    popoverColor: '#1c222b',
    tableColor: '#161b22',
    inputColor: '#1b212a',

    bodyColor: '#0e1116',
    // 暗色下不用纯白：纯白配深底会"发光"，看久了很累
    textColorBase: '#e6edf3',
    textColor1: '#e6edf3',
    textColor2: '#9aa7b4',
    textColor3: '#7d8a98',
    borderColor: '#262d38',
    dividerColor: '#262d38',
  },
  // 暗色下【不能】用提亮后的主色 #6b9bff —— 白字压上去只有 2.7:1。
  // 头像底色保持深蓝，和暗色界面也协调。
  Avatar: {
    color: '#2f6fed',
  },
  Empty: {
    textColor: '#9aa7b4',
    iconColor: '#7d8a98',
  },
  // 暗色下 #6b9bff 在页面底色（#0e1116）上已有 ~6:1，用默认值即可，
  // 这里显式写出来是为了亮暗两套结构一致、便于对照。
  Pagination: {
    itemTextColorActive: '#6b9bff',
    itemBorderActive: '1px solid #6b9bff',
  },
}
