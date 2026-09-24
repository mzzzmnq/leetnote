package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token 类型。放进 claims 里，防止「用 refresh token 当 access token 使」这类混用攻击。
const (
	TypeAccess  = "access"
	TypeRefresh = "refresh"
)

var (
	ErrInvalidToken = errors.New("token 无效或已过期")
	ErrWrongType    = errors.New("token 类型不匹配")
)

// Claims 是自定义载荷。
//
// 注意：JWT 的 payload 只是 Base64 编码，【任何人都能解开看】。
// 所以绝不能往里塞敏感信息（密码、手机号等），只放 ID 这类无所谓的标识。
type Claims struct {
	UserID int64  `json:"-"`
	Type   string `json:"typ"`
	jwt.RegisteredClaims
}

// TokenPair 是一次登录签发的双 Token。
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// Manager 负责签发与校验 Token。
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		issuer:     "leetnote-api",
	}
}

// GeneratePair 同时签发 access 与 refresh token。
func (m *Manager) GeneratePair(userID int64) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(m.accessTTL)
	refreshExp := now.Add(m.refreshTTL)

	access, err := m.sign(userID, TypeAccess, now, accessExp)
	if err != nil {
		return nil, err
	}

	refresh, err := m.sign(userID, TypeRefresh, now, refreshExp)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func (m *Manager) sign(userID int64, typ string, now, exp time.Time) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", err
	}

	claims := Claims{
		Type: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10), // sub：用户 ID
			ID:        jti,                           // jti：唯一标识，后续做黑名单吊销用
			Issuer:    m.issuer,                      // iss：签发者
			IssuedAt:  jwt.NewNumericDate(now),       // iat
			ExpiresAt: jwt.NewNumericDate(exp),       // exp
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("签发 token 失败: %w", err)
	}
	return signed, nil
}

// Parse 校验 Token 并返回 Claims。
//
// 安全要点（面试常问「JWT 怎么防篡改」）：
//   - WithValidMethods 显式限定 HS256，防止 alg 混淆攻击
//     （攻击者把 header 改成 alg=none 或 RS256，诱使服务端用错误方式验签）
//   - WithIssuer 校验签发者，防止拿别的系统的 Token 来冒用
//   - WithExpirationRequired 强制要求 exp，防止签发永不过期的 Token
func (m *Manager) Parse(tokenStr, expectedType string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Type != expectedType {
		return nil, ErrWrongType
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return nil, ErrInvalidToken
	}
	claims.UserID = userID

	return claims, nil
}

func newJTI() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成 jti 失败: %w", err)
	}
	return hex.EncodeToString(b), nil
}
