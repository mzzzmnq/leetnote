package model

import "time"

// OAuth provider 名称。
const (
	ProviderGitHub = "github"
)

// OAuthAccount 对应 oauth_accounts 表，表示「某个本地用户关联了某个第三方账号」。
type OAuthAccount struct {
	ID            int64
	UserID        int64
	Provider      string // github
	ProviderUID   string // provider 侧的用户唯一 ID（GitHub 的数字 id，转成字符串存）
	ProviderLogin string // provider 侧的登录名（展示用）
	AvatarURL     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
