package apitoken

import (
	"strings"
	"testing"
)

func TestGenerate_ProducesUsableToken(t *testing.T) {
	raw, hash, err := Generate()
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if !Looks(raw) {
		t.Errorf("生成的令牌应带 %q 前缀: %s", Prefix, raw)
	}
	if hash != Hash(raw) {
		t.Error("返回的 hash 应等于 Hash(raw)")
	}
	if len(hash) != 64 {
		t.Errorf("sha256 十六进制应为 64 个字符，实际 %d", len(hash))
	}
}

func TestGenerate_IsUnique(t *testing.T) {
	a, _, _ := Generate()
	b, _, _ := Generate()
	if a == b {
		t.Error("两次生成不应相同")
	}
}

func TestLooks(t *testing.T) {
	cases := map[string]bool{
		Prefix + strings.Repeat("a", 64): true,
		Prefix:                           false, // 只有前缀不算令牌
		"eyJhbGciOiJIUzI1NiJ9":           false, // JWT 不是 PAT
		"":                               false,
	}
	for in, want := range cases {
		if got := Looks(in); got != want {
			t.Errorf("Looks(%q) = %v，期望 %v", in, got, want)
		}
	}
}

func TestHash_IsStableAndDistinct(t *testing.T) {
	if Hash("ln_x") != Hash("ln_x") {
		t.Error("同一输入应得到同一哈希")
	}
	if Hash("ln_x") == Hash("ln_y") {
		t.Error("不同输入应得到不同哈希")
	}
}
