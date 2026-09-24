package dto

import (
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/model"
)

// GitHubAuthorizeResponse 返回给前端的 GitHub 授权地址。
//
// 为什么由后端生成而不是前端自己拼：state 必须由服务端生成并暂存，
// 前端拼的话就失去了防 CSRF 的意义。
type GitHubAuthorizeResponse struct {
	AuthorizeURL string `json:"authorize_url"`
}

// OAuthAccountResponse 是「已绑定的第三方账号」的对外表示。
type OAuthAccountResponse struct {
	Provider      string    `json:"provider"`
	ProviderLogin string    `json:"provider_login"`
	AvatarURL     *string   `json:"avatar_url"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewOAuthAccountResponse(a *model.OAuthAccount) OAuthAccountResponse {
	if a == nil {
		return OAuthAccountResponse{}
	}
	return OAuthAccountResponse{
		Provider:      a.Provider,
		ProviderLogin: a.ProviderLogin,
		AvatarURL:     a.AvatarURL,
		CreatedAt:     a.CreatedAt,
	}
}
