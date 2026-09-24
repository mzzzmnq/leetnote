package hash

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// MaxPasswordBytes 是 bcrypt 的硬限制。
//
// bcrypt 只取密码的前 72 字节，超出部分会被【静默丢弃】。
// 这意味着 "a"*72 + "X" 和 "a"*72 + "Y" 会被认为是同一个密码——
// 一个真实存在的安全隐患。所以必须在入口处直接拒绝，而不是截断。
const MaxPasswordBytes = 72

// ErrPasswordTooLong 在密码超过 bcrypt 上限时返回。
var ErrPasswordTooLong = fmt.Errorf("密码长度不能超过 %d 字节", MaxPasswordBytes)

// HashPassword 用 bcrypt 生成密码哈希。
//
// 为什么用 bcrypt 而不是 MD5/SHA256：
//   - 自带随机 salt，同一个密码每次哈希结果都不同（防彩虹表）
//   - 故意设计成慢（可调 cost），大幅抬高暴力破解成本
//   - SHA256 这类通用哈希追求快，用在密码上是错误选择
func HashPassword(plain string) (string, error) {
	if len(plain) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成密码哈希失败: %w", err)
	}
	return string(hashed), nil
}

// VerifyPassword 校验明文密码与哈希是否匹配。
//
// 注意：bcrypt.CompareHashAndPassword 内部使用【恒定时间比较】，
// 不会因为前缀匹配长度不同而泄漏信息，可以放心用于认证。
func VerifyPassword(hashed, plain string) bool {
	if len(plain) > MaxPasswordBytes {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

// DummyVerify 用于「用户不存在」时消耗与真实校验相当的时间。
//
// 如果不做这一步，攻击者可以通过响应时间差异判断用户名是否存在
// （用户不存在时立即返回，存在时要跑一次 bcrypt，差几十毫秒）。
// 这是典型的时序侧信道（timing side-channel）。
func DummyVerify() {
	// 一个固定的合法 bcrypt 哈希，比对必然失败，但耗时与真实校验一致
	const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte("dummy-password"))
}
