package hash_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/hash"
)

func TestHashAndVerify(t *testing.T) {
	const plain = "correct-horse-battery-staple"

	hashed, err := hash.HashPassword(plain)
	if err != nil {
		t.Fatalf("哈希失败: %v", err)
	}

	if hashed == plain {
		t.Fatal("绝不能明文存储密码")
	}
	if !strings.HasPrefix(hashed, "$2") {
		t.Errorf("不是 bcrypt 格式: %s", hashed)
	}
	if !hash.VerifyPassword(hashed, plain) {
		t.Error("正确密码应校验通过")
	}
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	hashed, _ := hash.HashPassword("right-password")

	if hash.VerifyPassword(hashed, "wrong-password") {
		t.Error("错误密码不应通过校验")
	}
	if hash.VerifyPassword(hashed, "") {
		t.Error("空密码不应通过校验")
	}
	if hash.VerifyPassword(hashed, "right-password ") {
		t.Error("带尾随空格的密码不应通过校验")
	}
}

// bcrypt 自带随机 salt，同一密码两次哈希结果必须不同。
// 否则彩虹表就能一次性破解所有相同密码的账号。
func TestHashIsSalted(t *testing.T) {
	const plain = "same-password"

	h1, _ := hash.HashPassword(plain)
	h2, _ := hash.HashPassword(plain)

	if h1 == h2 {
		t.Fatal("两次哈希结果相同，说明没有加随机 salt")
	}
	if !hash.VerifyPassword(h1, plain) || !hash.VerifyPassword(h2, plain) {
		t.Error("两个哈希都应能校验通过")
	}
}

// bcrypt 只取前 72 字节，超出部分会被静默丢弃。
// 必须直接拒绝，否则 "a"*72+"X" 和 "a"*72+"Y" 会被当成同一个密码。
func TestRejectsTooLongPassword(t *testing.T) {
	tooLong := strings.Repeat("a", hash.MaxPasswordBytes+1)

	if _, err := hash.HashPassword(tooLong); !errors.Is(err, hash.ErrPasswordTooLong) {
		t.Errorf("超长密码应返回 ErrPasswordTooLong, 实际 %v", err)
	}

	// 校验侧也要拒绝，不能拿超长密码去比对
	hashed, _ := hash.HashPassword(strings.Repeat("a", hash.MaxPasswordBytes))
	if hash.VerifyPassword(hashed, tooLong) {
		t.Error("超长密码不应通过校验")
	}
}

func TestBoundaryLength(t *testing.T) {
	exact := strings.Repeat("a", hash.MaxPasswordBytes)

	hashed, err := hash.HashPassword(exact)
	if err != nil {
		t.Fatalf("恰好 72 字节应被接受, 实际 %v", err)
	}
	if !hash.VerifyPassword(hashed, exact) {
		t.Error("边界长度密码应校验通过")
	}
}

// DummyVerify 用于消除「用户不存在」与「密码错误」的响应时间差，
// 本身不应 panic 或返回任何信息。
func TestDummyVerifyDoesNotPanic(t *testing.T) {
	hash.DummyVerify()
}
