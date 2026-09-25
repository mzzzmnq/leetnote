package repository

import "strings"

// likeEscaper 转义 LIKE / ILIKE 模式里的通配符。
//
// 为什么需要：ILIKE 里 % 匹配任意字符、_ 匹配单个字符。
// 用户搜索「时间复杂度 O(n) 的 50% 情况」时，那个 % 会被当成通配符，
// 导致匹配出一堆无关结果。必须转义成字面量。
//
// PostgreSQL 的 LIKE 默认转义字符是反斜杠，所以先把 \ 自己转义，
// 再把 % 和 _ 各加一个反斜杠前缀。
var likeEscaper = strings.NewReplacer(
	`\`, `\\`,
	`%`, `\%`,
	`_`, `\_`,
)

// escapeLike 把用户输入转成可以安全嵌入 ILIKE 模式的字面量。
func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

// likeContains 把关键词转成「包含匹配」的模式串，如 "abc" -> "%abc%"。
func likeContains(s string) string {
	return "%" + escapeLike(s) + "%"
}
