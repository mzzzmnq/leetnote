package jwt_test

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
)

const testSecret = "unit-test-secret-do-not-use-in-production"

func newManager() *jwt.Manager {
	return jwt.NewManager(testSecret, 15*time.Minute, 24*time.Hour)
}

func TestGenerateAndParse(t *testing.T) {
	m := newManager()

	pair, err := m.GeneratePair(42)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("token 不应为空")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Fatal("access 与 refresh token 不应相同")
	}

	claims, err := m.Parse(pair.AccessToken, jwt.TypeAccess)
	if err != nil {
		t.Fatalf("校验失败: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("user_id 错误: 期望 42, 实际 %d", claims.UserID)
	}
	if claims.ID == "" {
		t.Error("jti 不应为空（后续做黑名单吊销要用）")
	}
}

// 用 refresh token 冒充 access token 访问业务接口，必须被拒绝。
func TestParseRejectsWrongType(t *testing.T) {
	m := newManager()
	pair, _ := m.GeneratePair(1)

	_, err := m.Parse(pair.RefreshToken, jwt.TypeAccess)
	if err == nil {
		t.Fatal("refresh token 不应能当作 access token 使用")
	}
	if !errors.Is(err, jwt.ErrWrongType) {
		t.Errorf("期望 ErrWrongType, 实际 %v", err)
	}
}

// 用别的密钥签发的 token 必须校验失败（防伪造）。
func TestParseRejectsForeignSignature(t *testing.T) {
	other := jwt.NewManager("a-completely-different-secret", time.Hour, time.Hour)
	pair, err := other.GeneratePair(1)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	if _, err := newManager().Parse(pair.AccessToken, jwt.TypeAccess); err == nil {
		t.Fatal("其他密钥签发的 token 不应通过校验")
	}
}

// 篡改 payload 必须导致签名校验失败。
func TestParseRejectsTamperedPayload(t *testing.T) {
	m := newManager()
	pair, _ := m.GeneratePair(1)

	parts := splitToken(t, pair.AccessToken)
	// 把 payload 换成一个把 user_id 改成 999 的版本
	forged := parts[0] + "." + b64(`{"sub":"999","typ":"access","iss":"leetnote-api","exp":9999999999}`) + "." + parts[2]

	if _, err := m.Parse(forged, jwt.TypeAccess); err == nil {
		t.Fatal("篡改后的 token 不应通过校验")
	}
}

// 经典攻击：把 header 的 alg 改成 none，诱使服务端跳过验签。
// 防御手段是 jwt.WithValidMethods 显式限定 HS256。
func TestParseRejectsNoneAlgorithm(t *testing.T) {
	forged := b64(`{"alg":"none","typ":"JWT"}`) + "." +
		b64(`{"sub":"1","typ":"access","iss":"leetnote-api","exp":9999999999}`) + "."

	if _, err := newManager().Parse(forged, jwt.TypeAccess); err == nil {
		t.Fatal("alg=none 的 token 绝不应通过校验")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	// TTL 为负数 → 签出来就已经过期
	m := jwt.NewManager(testSecret, -time.Minute, -time.Minute)
	pair, err := m.GeneratePair(1)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	if _, err := m.Parse(pair.AccessToken, jwt.TypeAccess); err == nil {
		t.Fatal("过期 token 不应通过校验")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	m := newManager()
	for _, bad := range []string{"", "abc", "a.b.c", "not.a.token"} {
		if _, err := m.Parse(bad, jwt.TypeAccess); err == nil {
			t.Errorf("垃圾输入 %q 不应通过校验", bad)
		}
	}
}

func b64(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func splitToken(t *testing.T, token string) []string {
	t.Helper()
	parts := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	if len(parts) != 3 {
		t.Fatalf("token 格式异常: %s", token)
	}
	return parts
}
