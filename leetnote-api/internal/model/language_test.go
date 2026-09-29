package model

import (
	"strings"
	"testing"
)

func TestNormalizeLanguageCanonical(t *testing.T) {
	// 已经是规范值的，原样返回
	for _, want := range SupportedLanguages {
		got, ok := NormalizeLanguage(string(want))
		if !ok {
			t.Errorf("%q 应该是受支持的语言", want)
		}
		if got != want {
			t.Errorf("NormalizeLanguage(%q) = %q, 期望 %q", want, got, want)
		}
	}
}

func TestNormalizeLanguageAliases(t *testing.T) {
	cases := map[string]Language{
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
		"cxx":       LanguageCPP,
		"cplusplus": LanguageCPP,
		// C
		"gcc": LanguageC,
	}

	for input, want := range cases {
		got, ok := NormalizeLanguage(input)
		if !ok {
			t.Errorf("%q 应该被识别（别名）", input)
			continue
		}
		if got != want {
			t.Errorf("NormalizeLanguage(%q) = %q, 期望 %q", input, got, want)
		}
	}
}

// 大小写与空白是最容易混进脏数据的两个来源，必须归一化掉
func TestNormalizeLanguageCaseAndWhitespace(t *testing.T) {
	cases := map[string]Language{
		"Python":     LanguagePython,
		"PYTHON":     LanguagePython,
		"  python  ": LanguagePython,
		"Go":         LanguageGo,
		"GO":         LanguageGo,
		"\tGo\n":     LanguageGo,
		"JavaScript": LanguageJavaScript,
		"JS":         LanguageJavaScript,
		"C++":        LanguageCPP,
		"CPP":        LanguageCPP,
	}

	for input, want := range cases {
		got, ok := NormalizeLanguage(input)
		if !ok {
			t.Errorf("%q 应该被识别", input)
			continue
		}
		if got != want {
			t.Errorf("NormalizeLanguage(%q) = %q, 期望 %q", input, got, want)
		}
	}
}

// 归一化的核心保证：不同写法必须收敛到同一个值，
// 否则「按语言分组」就会出现两个 Python 分组
func TestNormalizeLanguageConverges(t *testing.T) {
	groups := map[Language][]string{
		LanguagePython:     {"python", "py", "Python", "PYTHON3", " py "},
		LanguageGo:         {"go", "Go", "golang", "GOLANG"},
		LanguageJavaScript: {"javascript", "js", "JS", "node", "NodeJS"},
		LanguageTypeScript: {"typescript", "ts", "TypeScript"},
		LanguageCPP:        {"cpp", "c++", "C++", "cxx", "CPlusPlus"},
		LanguageC:          {"c", "C", "gcc"},
	}

	for want, inputs := range groups {
		for _, in := range inputs {
			got, ok := NormalizeLanguage(in)
			if !ok {
				t.Errorf("%q 应该被识别", in)
				continue
			}
			if got != want {
				t.Errorf("NormalizeLanguage(%q) = %q, 期望收敛到 %q", in, got, want)
			}
		}
	}
}

func TestNormalizeLanguageRejectsUnsupported(t *testing.T) {
	// 空值与非语言字符串都必须被拒绝，
	// 否则会把脏数据写进库（数据库那层还有 CHECK 兜底）
	bad := []string{
		"",
		"   ",
		"rust",
		"swift",
		"kotlin",
		"sql",
		"c#",
		"brainfuck",
		"中文",
	}

	for _, in := range bad {
		if got, ok := NormalizeLanguage(in); ok {
			t.Errorf("NormalizeLanguage(%q) 不该通过，却返回了 %q", in, got)
		}
	}
}

func TestIsSupportedLanguage(t *testing.T) {
	if !IsSupportedLanguage("Go") {
		t.Error("Go 应该受支持")
	}
	if IsSupportedLanguage("rust") {
		t.Error("rust 不该受支持")
	}
}

// SupportedLanguageNames 用于拼错误信息和前端展示，顺序不能乱
func TestSupportedLanguageNames(t *testing.T) {
	names := SupportedLanguageNames()

	if len(names) != len(SupportedLanguages) {
		t.Fatalf("数量对不上：%d vs %d", len(names), len(SupportedLanguages))
	}

	// 常用的排前面（配合前端下拉与标签页）
	if names[0] != "python" || names[1] != "go" {
		t.Errorf("前两个应该是最常用的 python / go，实际 %v", names[:2])
	}

	if strings.Join(names, ",") != "python,go,typescript,javascript,java,cpp,c" {
		t.Errorf("顺序或内容变了：%v", names)
	}
}

// 白名单里不能有重复项或未规范化的值
func TestSupportedLanguagesAreClean(t *testing.T) {
	seen := map[Language]bool{}
	for _, lang := range SupportedLanguages {
		if seen[lang] {
			t.Errorf("重复的语言：%q", lang)
		}
		seen[lang] = true

		if string(lang) != strings.ToLower(string(lang)) {
			t.Errorf("语言标识必须是小写：%q", lang)
		}
		if strings.TrimSpace(string(lang)) != string(lang) {
			t.Errorf("语言标识不能有空白：%q", lang)
		}

		// 每个规范值都必须能被 NormalizeLanguage 认出来
		if got, ok := NormalizeLanguage(string(lang)); !ok || got != lang {
			t.Errorf("SupportedLanguages 里的 %q 无法自洽归一化", lang)
		}
	}
}
