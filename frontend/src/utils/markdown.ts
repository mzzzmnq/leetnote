import MarkdownIt from 'markdown-it'
// 【按需引入】highlight.js 默认会把 190+ 种语言全部打包（约 1MB）。
// 用 lib/core + 手动注册，只带上真正会用到的语言，产物能小一个数量级。
import hljs from 'highlight.js/lib/core'
import c from 'highlight.js/lib/languages/c'
import cpp from 'highlight.js/lib/languages/cpp'
import csharp from 'highlight.js/lib/languages/csharp'
import go from 'highlight.js/lib/languages/go'
import java from 'highlight.js/lib/languages/java'
import javascript from 'highlight.js/lib/languages/javascript'
import kotlin from 'highlight.js/lib/languages/kotlin'
import python from 'highlight.js/lib/languages/python'
import rust from 'highlight.js/lib/languages/rust'
import sql from 'highlight.js/lib/languages/sql'
import swift from 'highlight.js/lib/languages/swift'
import typescript from 'highlight.js/lib/languages/typescript'

// 别名（js / py / golang 等）由 highlight.js 的语言定义自带，注册后即可识别
const LANGUAGES = {
  c,
  cpp,
  csharp,
  go,
  java,
  javascript,
  kotlin,
  python,
  rust,
  sql,
  swift,
  typescript,
} as const

for (const [name, definition] of Object.entries(LANGUAGES)) {
  hljs.registerLanguage(name, definition)
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

/** 语言名只保留字母数字和连字符，避免拼进 class 属性时出问题 */
function escapeAttr(value: string): string {
  return value.replace(/[^\w-]/g, '')
}

/** 把一段代码渲染成带高亮的 HTML */
function highlightCode(code: string, lang: string): string {
  if (lang && hljs.getLanguage(lang)) {
    try {
      const { value } = hljs.highlight(code, { language: lang, ignoreIllegals: true })
      return `<pre class="hljs"><code class="language-${escapeAttr(lang)}">${value}</code></pre>`
    } catch {
      // 高亮失败就退回纯文本，不能让整个渲染挂掉
    }
  }
  return `<pre class="hljs"><code>${escapeHtml(code)}</code></pre>`
}

const md = new MarkdownIt({
  // 【安全关键】禁用原始 HTML。
  //
  // 打开的话，笔记里的 <script> 会被原样渲染并执行。
  // 虽然笔记是用户自己写的，但一旦将来支持分享/多人协作，
  // 这就是一个存储型 XSS 漏洞。默认关掉是最省心的选择。
  html: false,

  linkify: true,
  breaks: true,

  highlight: highlightCode,
})

// 外部链接加 target="_blank" + rel="noopener noreferrer"。
//
// 只加 target 不加 rel 的话，新页面能通过 window.opener 操作原页面
// （tabnabbing 攻击）。这是常被忽略的一个细节。
const defaultLinkOpen =
  md.renderer.rules.link_open ??
  ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))

md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  if (token) {
    token.attrSet('target', '_blank')
    token.attrSet('rel', 'noopener noreferrer')
  }
  return defaultLinkOpen(tokens, idx, options, env, self)
}

/** 把 Markdown 渲染成 HTML */
export function renderMarkdown(source: string | null | undefined): string {
  return md.render(source ?? '')
}

/** 单独渲染一个代码块（用于解法展示） */
export function renderCodeBlock(code: string | null | undefined, lang: string | null | undefined): string {
  return highlightCode(code ?? '', lang ?? '')
}

/** 取纯文本摘要（用于列表页） */
export function plainText(source: string | null | undefined, maxLength = 120): string {
  const text = (source ?? '')
    .replace(/```[\s\S]*?```/g, ' ') // 去掉代码块
    .replace(/[#>*_`~\-[\]()]/g, ' ') // 去掉常见标记符
    .replace(/\s+/g, ' ')
    .trim()

  return text.length > maxLength ? `${text.slice(0, maxLength)}…` : text
}
