/**
 * 受支持的编程语言。
 *
 * 【必须与后端 internal/model/language.go 的 SupportedLanguages 保持一致】
 * —— 值、顺序、数量都要对得上。后端是权威（它会归一化并拒绝非法值），
 * 这里只是把它镜像过来用于展示。
 *
 * 为什么不用接口动态拉取：语言列表是低频变更的编译期常量，
 * 为它加一次请求不划算；而加语言时改动本来就跨越前后端，
 * 顺手同步这个文件是很小的成本。
 */

/** 规范的小写标识。顺序即下拉与标签页的展示顺序 */
export const LANGUAGES = [
  'python',
  'go',
  'typescript',
  'javascript',
  'java',
  'cpp',
  'c',
] as const

export type Language = (typeof LANGUAGES)[number]

/** 展示名。注意 C++ 这类不能直接首字母大写 */
const LABELS: Record<Language, string> = {
  python: 'Python',
  go: 'Go',
  typescript: 'TypeScript',
  javascript: 'JavaScript',
  java: 'Java',
  cpp: 'C++',
  c: 'C',
}

/**
 * 语言标识色（取各自官方品牌色的近似值，仅用于标签页前面那个小圆点）。
 *
 * 圆点是装饰性的，不承担对比度要求；语言名本身用的是正文色，
 * 所以这里可以放心用品牌色。
 * JavaScript 的官方黄（#f7df1e）在浅色背景上几乎看不见，稍微压深了一点。
 */
const DOT_COLORS: Record<Language, string> = {
  python: '#3776ab',
  go: '#00add8',
  typescript: '#3178c6',
  javascript: '#d1b300',
  java: '#e76f00',
  cpp: '#00599c',
  c: '#7f8ea3',
}

const LANGUAGE_SET = new Set<string>(LANGUAGES)

/** 判断一个字符串是否是受支持的语言 */
export function isLanguage(value: string): value is Language {
  return LANGUAGE_SET.has(value)
}

/**
 * 取展示名。
 *
 * 后端可能返回这里不认识的旧值（比如迁移前遗留的 rust），
 * 那种情况原样展示，总比显示成空白要好。
 */
export function languageLabel(value: string): string {
  if (isLanguage(value)) return LABELS[value]
  return value
}

/** 取圆点颜色；不认识的语言返回中性灰 */
export function languageDotColor(value: string): string {
  if (isLanguage(value)) return DOT_COLORS[value]
  return '#8b95a5'
}

/**
 * 语言的排序权重。
 *
 * 标签页按 LANGUAGES 的顺序排，用户最常用的（Python / Go）在最左边；
 * 顺序表里没有的语言排到最后，而不是被丢掉。
 */
export function languageOrder(value: string): number {
  const i = LANGUAGES.indexOf(value as Language)
  return i === -1 ? LANGUAGES.length : i
}
