package model

import "strings"

// Language 是受支持的编程语言标识。
//
// 【统一用小写规范形式存储】，避免 Python / PY / py 混着存 ——
// 一旦存进去，前端按语言分组、做统计时就全乱套了，而且历史数据很难清洗。
// 所以入口处（API）统一归一化，数据库再加一层 CHECK 约束兜底。
type Language string

const (
	LanguagePython     Language = "python"
	LanguageGo         Language = "go"
	LanguageJavaScript Language = "javascript"
	LanguageTypeScript Language = "typescript"
	LanguageJava       Language = "java"
	LanguageCPP        Language = "cpp"
	LanguageC          Language = "c"
)

// SupportedLanguages 是全部受支持的语言，**顺序即前端下拉与标签页的展示顺序**。
//
// 排序考虑：把自己常用的（Python 刷题、Go 后端、TS 前端）放前面，
// 减少下拉里的滚动。
var SupportedLanguages = []Language{
	LanguagePython,
	LanguageGo,
	LanguageTypeScript,
	LanguageJavaScript,
	LanguageJava,
	LanguageCPP,
	LanguageC,
}

// languageAliases 把常见别名与写法变体映射到规范值。
//
// 用户手快会打成 `py`、`golang`、`c++`、`JS`，这些都不该报错，
// 而是静默归一化成规范形式 —— 毕竟它们的含义毫无歧义。
var languageAliases = map[string]Language{
	// Python
	"py":      LanguagePython,
	"python3": LanguagePython,
	"python2": LanguagePython,

	// Go
	"golang": LanguageGo,

	// JavaScript
	"js":     LanguageJavaScript,
	"node":   LanguageJavaScript,
	"nodejs": LanguageJavaScript,

	// TypeScript
	"ts": LanguageTypeScript,

	// C++
	"c++":       LanguageCPP,
	"cplusplus": LanguageCPP,
	"cxx":       LanguageCPP,

	// C
	"gcc": LanguageC,
}

// NormalizeLanguage 把用户输入转成规范形式。
//
// 第二个返回值表示是否是受支持的语言；为 false 时调用方应该返回
// 一个明确的参数错误，而不是把脏数据写进库。
//
// 归一化步骤：去空白 → 转小写 → 查别名 → 查规范表。
func NormalizeLanguage(raw string) (Language, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "", false
	}

	if canonical, ok := languageAliases[s]; ok {
		return canonical, true
	}

	for _, lang := range SupportedLanguages {
		if string(lang) == s {
			return lang, true
		}
	}

	return Language(s), false
}

// IsSupportedLanguage 判断一个值是否已经是规范形式且受支持。
func IsSupportedLanguage(raw string) bool {
	_, ok := NormalizeLanguage(raw)
	return ok
}

// SupportedLanguageNames 返回规范形式的字符串列表，方便拼错误信息或做校验。
func SupportedLanguageNames() []string {
	out := make([]string, 0, len(SupportedLanguages))
	for _, lang := range SupportedLanguages {
		out = append(out, string(lang))
	}
	return out
}
