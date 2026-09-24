// Package oauth 封装第三方 OAuth 提供方的交互细节。
//
// 之所以独立成包（而不是塞进 service）：这是【外部依赖】，
// 涉及 HTTP 调用、超时、重试、响应格式解析。把它隔离出来，
// service 层就只需要面对干净的领域对象，也方便在测试里替换成假实现。
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
	githubUserURL      = "https://api.github.com/user"
	githubEmailsURL    = "https://api.github.com/user/emails"

	// 只申请必需的最小权限：读基本资料 + 读邮箱。
	// 权限要得越多，用户越犹豫，泄漏面也越大。
	githubScopes = "read:user user:email"

	// GitHub 要求所有 API 请求必须带 User-Agent，否则返回 403
	userAgent = "leetnote-api"

	maxBodyBytes = 1 << 20 // 1MB，防止对方返回超大响应把内存打爆
)

// GitHubUser 是归一化后的 GitHub 用户信息。
type GitHubUser struct {
	ID        int64  // GitHub 的数字 ID —— 稳定不变，是唯一可信的标识
	Login     string // 登录名 —— 用户随时可以改，不能用来做标识
	Name      string
	Email     string // 已验证的主邮箱，可能为空
	AvatarURL string
}

// GitHubClient 负责与 GitHub OAuth 交互。
type GitHubClient struct {
	clientID     string
	clientSecret string
	redirectURL  string
	http         *http.Client
}

func NewGitHubClient(clientID, clientSecret, redirectURL string) *GitHubClient {
	return &GitHubClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		http: &http.Client{
			// 【外部依赖必须设超时】否则对方不响应时，
			// 我们的请求会一直挂着，最终把连接池和 goroutine 耗尽。
			Timeout: 10 * time.Second,
		},
	}
}

// Enabled 表示是否配置了 GitHub OAuth 凭据。
func (c *GitHubClient) Enabled() bool {
	return c.clientID != "" && c.clientSecret != ""
}

// GenerateState 生成 OAuth state 参数。
//
// state 的作用是防 CSRF：攻击者可以伪造一个带自己 code 的回调地址，
// 诱导你点击，从而把你的账号和攻击者的 GitHub 绑定。
// 有了 state，服务端能确认「这次回调确实是我发起的那一次」。
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成 state 失败: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// AuthorizeURL 拼出让用户跳转的授权地址。
func (c *GitHubClient) AuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", c.clientID)
	q.Set("redirect_uri", c.redirectURL)
	q.Set("scope", githubScopes)
	q.Set("state", state)
	return githubAuthorizeURL + "?" + q.Encode()
}

// Exchange 用授权码换取 access_token。
func (c *GitHubClient) Exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", c.redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("构造换 token 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 GitHub 换取 token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return "", fmt.Errorf("读取 GitHub 响应失败: %w", err)
	}

	var payload struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("解析 GitHub 响应失败: %w", err)
	}

	// 【GitHub 的坑】换 token 失败时它可能返回 HTTP 200，
	// 真正的错误藏在 body 的 error 字段里。只看状态码会误判为成功。
	if payload.Error != "" {
		return "", fmt.Errorf("GitHub 拒绝授权: %s (%s)", payload.ErrorDescription, payload.Error)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("GitHub 未返回 access_token")
	}

	return payload.AccessToken, nil
}

// FetchUser 拉取用户资料，并在必要时补全已验证邮箱。
func (c *GitHubClient) FetchUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	var raw struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.getJSON(ctx, githubUserURL, accessToken, &raw); err != nil {
		return nil, err
	}

	u := &GitHubUser{
		ID:        raw.ID,
		Login:     raw.Login,
		Name:      raw.Name,
		Email:     raw.Email,
		AvatarURL: raw.AvatarURL,
	}

	// 用户把邮箱设为私密时，/user 返回的 email 是 null，
	// 必须再调 /user/emails 才能拿到。
	if u.Email == "" {
		u.Email = c.fetchPrimaryEmail(ctx, accessToken)
	}

	return u, nil
}

// fetchPrimaryEmail 取「已验证」的邮箱。
//
// 只认 verified=true 的邮箱：如果拿未验证的邮箱去关联已有账号，
// 攻击者只要在 GitHub 上填一个别人的邮箱，就能接管对方的账号。
func (c *GitHubClient) fetchPrimaryEmail(ctx context.Context, accessToken string) string {
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := c.getJSON(ctx, githubEmailsURL, accessToken, &emails); err != nil {
		return ""
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email
		}
	}
	return ""
}

func (c *GitHubClient) getJSON(ctx context.Context, endpoint, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("请求 GitHub 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return fmt.Errorf("读取 GitHub 响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub 返回 %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("解析 GitHub 响应失败: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
