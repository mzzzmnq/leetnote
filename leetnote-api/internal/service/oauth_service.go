package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/oauth"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

type OAuthService struct {
	users    repository.UserRepository
	accounts repository.OAuthRepository
	github   *oauth.GitHubClient
	tokens   *jwt.Manager
}

func NewOAuthService(
	users repository.UserRepository,
	accounts repository.OAuthRepository,
	github *oauth.GitHubClient,
	tokens *jwt.Manager,
) *OAuthService {
	return &OAuthService{users: users, accounts: accounts, github: github, tokens: tokens}
}

// GitHubEnabled 表示是否配置了 GitHub OAuth。
func (s *OAuthService) GitHubEnabled() bool {
	return s.github.Enabled()
}

// GitHubAuthorizeURL 生成让前端跳转的授权地址。
func (s *OAuthService) GitHubAuthorizeURL(state string) string {
	return s.github.AuthorizeURL(state)
}

// GenerateState 生成防 CSRF 的 state。
func (s *OAuthService) GenerateState() (string, error) {
	return oauth.GenerateState()
}

// ExchangeCode 用授权码换取 GitHub 用户信息。
func (s *OAuthService) ExchangeCode(ctx context.Context, code string) (*oauth.GitHubUser, error) {
	token, err := s.github.Exchange(ctx, code)
	if err != nil {
		return nil, errs.ErrUnauthorized.WithMessage("GitHub 授权失败，请重试").Wrap(err)
	}

	gu, err := s.github.FetchUser(ctx, token)
	if err != nil {
		return nil, errs.ErrUnauthorized.WithMessage("获取 GitHub 用户信息失败").Wrap(err)
	}
	return gu, nil
}

// LoginWithGitHub 用 GitHub 身份完成登录。
//
// 三种情况，按优先级处理：
//  1. 该 GitHub 账号已关联过 → 直接登录
//  2. 未关联，但邮箱命中已有本地账号 → 关联后登录（账号合并）
//  3. 都没有 → 创建新用户
func (s *OAuthService) LoginWithGitHub(ctx context.Context, gu *oauth.GitHubUser) (*AuthResult, error) {
	uid := strconv.FormatInt(gu.ID, 10)

	// ---------- 1. 已关联 ----------
	account, err := s.accounts.GetByProviderUID(ctx, model.ProviderGitHub, uid)
	if err == nil {
		u, err := s.users.GetByID(ctx, account.UserID)
		if err != nil {
			return nil, err
		}
		return issueTokens(s.tokens, u)
	}
	if !errors.Is(err, errs.ErrNotFound) {
		return nil, err
	}

	// ---------- 2. 按邮箱关联已有账号 ----------
	// 只认 GitHub 已验证的邮箱（由 oauth 客户端保证），
	// 否则攻击者在 GitHub 填上别人的邮箱就能接管对方账号。
	var u *model.User
	if gu.Email != "" {
		existing, err := s.users.GetByEmail(ctx, gu.Email)
		switch {
		case err == nil:
			u = existing
		case errors.Is(err, errs.ErrNotFound):
			// 正常情况：没有同名邮箱的账号
		default:
			return nil, err
		}
	}

	// ---------- 3. 创建新用户 ----------
	if u == nil {
		u, err = s.createUserFromGitHub(ctx, gu)
		if err != nil {
			return nil, err
		}
	}

	// ---------- 4. 建立关联 ----------
	if err := s.linkAccount(ctx, u.ID, gu); err != nil {
		return nil, err
	}

	return issueTokens(s.tokens, u)
}

// UnlinkGitHub 解除 GitHub 绑定。
//
// 安全约束：如果该账号【没有密码】，解绑后就再也无法登录了，
// 因此必须拒绝，并提示用户先设置密码。
func (s *OAuthService) UnlinkGitHub(ctx context.Context, userID int64) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !u.HasPassword() {
		return errs.ErrBadRequest.WithMessage(
			"解绑前请先设置密码，否则你将无法再登录")
	}

	return s.accounts.Delete(ctx, userID, model.ProviderGitHub)
}

// ListLinkedAccounts 返回当前用户已绑定的第三方账号。
func (s *OAuthService) ListLinkedAccounts(ctx context.Context, userID int64) ([]*model.OAuthAccount, error) {
	return s.accounts.ListByUserID(ctx, userID)
}

// ---------------------------------------------------------------

func (s *OAuthService) linkAccount(ctx context.Context, userID int64, gu *oauth.GitHubUser) error {
	avatar := gu.AvatarURL
	var avatarPtr *string
	if avatar != "" {
		avatarPtr = &avatar
	}

	account := &model.OAuthAccount{
		UserID:        userID,
		Provider:      model.ProviderGitHub,
		ProviderUID:   strconv.FormatInt(gu.ID, 10),
		ProviderLogin: gu.Login,
		AvatarURL:     avatarPtr,
	}

	err := s.accounts.Create(ctx, account)
	if err == nil {
		return nil
	}

	// 并发登录时可能两个请求同时走到这里，唯一约束会挡住其中一个。
	// 这不是错误，重新读取已存在的关联即可。
	if errors.Is(err, errs.ErrConflict) {
		if _, e := s.accounts.GetByProviderUID(ctx, model.ProviderGitHub, account.ProviderUID); e == nil {
			return nil
		}
	}
	return err
}

// createUserFromGitHub 创建纯 OAuth 用户（没有密码）。
func (s *OAuthService) createUserFromGitHub(ctx context.Context, gu *oauth.GitHubUser) (*model.User, error) {
	// 没有可用邮箱就没法建账号——users.email 是 NOT NULL UNIQUE。
	// 这里不做「造一个假邮箱」这种糊弄事，直接告诉用户去 GitHub 处理。
	if strings.TrimSpace(gu.Email) == "" {
		return nil, errs.ErrBadRequest.WithMessage(
			"你的 GitHub 账号没有已验证的邮箱，请先在 GitHub 设置中验证邮箱后重试")
	}

	base := sanitizeUsername(gu.Login)
	if base == "" {
		base = fmt.Sprintf("gh_%d", gu.ID)
	}

	var avatarPtr *string
	if gu.AvatarURL != "" {
		avatarPtr = &gu.AvatarURL
	}

	// 用户名可能与已有用户冲突（GitHub 登录名和本站用户名规则不同），
	// 冲突就加随机后缀重试。
	for attempt := 0; attempt < 5; attempt++ {
		username := base
		if attempt > 0 {
			suffix, err := randomSuffix(4)
			if err != nil {
				return nil, errs.ErrInternal.Wrap(err)
			}
			username = truncateString(base, 45) + "_" + suffix
		}

		u := &model.User{
			Username:     username,
			Email:        strings.ToLower(gu.Email),
			PasswordHash: nil, // OAuth 用户没有密码
			AvatarURL:    avatarPtr,
		}

		err := s.users.Create(ctx, u)
		if err == nil {
			return u, nil
		}

		if errors.Is(err, errs.ErrUsernameTaken) {
			continue // 换个后缀再试
		}
		// 邮箱冲突说明刚才的按邮箱查询与实际写入之间存在竞态，
		// 或者是并发注册。交给调用方重试整条流程。
		return nil, err
	}

	return nil, errs.ErrConflict.WithMessage("用户名冲突次数过多，请稍后重试")
}

// sanitizeUsername 把 GitHub 登录名转成本站允许的用户名。
//
// GitHub 登录名只含字母数字和连字符；本站规则是 [a-zA-Z0-9_]{3,50}。
func sanitizeUsername(login string) string {
	var b strings.Builder
	for _, r := range login {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}

	s := b.String()
	if len(s) < 3 {
		s += "_gh"
	}
	return truncateString(s, 50)
}

func randomSuffix(n int) (string, error) {
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成随机后缀失败: %w", err)
	}
	return hex.EncodeToString(b)[:n], nil
}

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
