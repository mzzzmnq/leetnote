// Package apitoken 生成与校验长期 API 令牌（PAT）。
//
// 与 JWT 的分工：
//   - JWT：交互式登录用，短期（15min access）+ refresh Cookie
//   - PAT：本机工具 / 浏览器脚本用，长期、可吊销、无刷新概念
//
// 两者格式不同，Auth 中间件靠 Prefix 分流。
package apitoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Prefix 是长期令牌的固定前缀。
//
// 有前缀才能在 Auth 里一眼区分「这是 PAT 还是 JWT」——
// 两种凭据的校验方式完全不同，靠格式试探容易出隐蔽 bug。
const Prefix = "ln_"

// Generate 生成一把新令牌，返回「明文」与「入库用的哈希」。
//
// 明文只在此刻返回一次，之后无法从库里还原 —— 这正是令牌类凭据应有的形态。
func Generate() (raw string, hash string, err error) {
	b := make([]byte, 32) // 256 bit 随机，足够抵抗暴力枚举
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("生成随机令牌失败: %w", err)
	}
	raw = Prefix + hex.EncodeToString(b)
	return raw, Hash(raw), nil
}

// Hash 计算入库用的 sha256 十六进制串。
//
// 【为什么不用 bcrypt 这类慢哈希】令牌是 256 bit 高熵随机串，
// 不存在字典可猜，慢哈希只会让每次鉴权都多花几十毫秒。慢哈希是针对
// 「低熵口令」的防御，这里用不上。
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Looks 判断一个字符串是否像长期令牌（用于 Auth 分流）。
func Looks(raw string) bool {
	return strings.HasPrefix(raw, Prefix) && len(raw) > len(Prefix)
}
