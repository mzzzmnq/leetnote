package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/oauth"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
	"github.com/mzzzmnq/leetnote-api/internal/service"
)

// ---------- 内存版 OAuthRepository ----------

type fakeOAuthRepo struct {
	mu       sync.Mutex
	accounts map[string]*model.OAuthAccount // key: provider|providerUID
	nextID   int64
}

func newFakeOAuthRepo() *fakeOAuthRepo {
	return &fakeOAuthRepo{accounts: make(map[string]*model.OAuthAccount), nextID: 1}
}

func oauthKey(provider, uid string) string { return provider + "|" + uid }

func (f *fakeOAuthRepo) Create(_ context.Context, a *model.OAuthAccount) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	k := oauthKey(a.Provider, a.ProviderUID)
	if _, exists := f.accounts[k]; exists {
		return errs.ErrConflict.WithMessage("该 GitHub 账号已被其他用户绑定")
	}

	a.ID = f.nextID
	f.nextID++
	a.CreatedAt = time.Now()
	a.UpdatedAt = a.CreatedAt

	clone := *a
	f.accounts[k] = &clone
	return nil
}

func (f *fakeOAuthRepo) GetByProviderUID(_ context.Context, provider, uid string) (*model.OAuthAccount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	a, ok := f.accounts[oauthKey(provider, uid)]
	if !ok {
		return nil, errs.ErrNotFound.WithMessage("未找到关联的第三方账号")
	}
	clone := *a
	return &clone, nil
}

func (f *fakeOAuthRepo) ListByUserID(_ context.Context, userID int64) ([]*model.OAuthAccount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]*model.OAuthAccount, 0, 1)
	for _, a := range f.accounts {
		if a.UserID == userID {
			clone := *a
			out = append(out, &clone)
		}
	}
	return out, nil
}

func (f *fakeOAuthRepo) Delete(_ context.Context, userID int64, provider string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for k, a := range f.accounts {
		if a.UserID == userID && a.Provider == provider {
			delete(f.accounts, k)
			return nil
		}
	}
	return errs.ErrNotFound.WithMessage("该账号未绑定此第三方登录")
}

// ---------- 测试脚手架 ----------

func newOAuthService() (*service.OAuthService, *fakeUserRepo, *fakeOAuthRepo) {
	users := newFakeUserRepo()
	accounts := newFakeOAuthRepo()
	tokens := jwt.NewManager("oauth-test-secret", 15*time.Minute, 24*time.Hour)
	// LoginWithGitHub 只用 gu（已归一化的用户信息），不会真的调 GitHub
	github := oauth.NewGitHubClient("", "", "")

	return service.NewOAuthService(users, accounts, github, tokens), users, accounts
}

func githubUser(id int64, login, email string) *oauth.GitHubUser {
	return &oauth.GitHubUser{
		ID:        id,
		Login:     login,
		Name:      "Test " + login,
		Email:     email,
		AvatarURL: "https://avatars.githubusercontent.com/u/" + fmt.Sprint(id),
	}
}

// ---------- 测试 ----------

func TestGitHubLoginCreatesNewUser(t *testing.T) {
	svc, _, _ := newOAuthService()

	result, err := svc.LoginWithGitHub(context.Background(), githubUser(1001, "cao-dev", "cao@example.com"))
	if err != nil {
		t.Fatalf("GitHub 登录失败: %v", err)
	}

	// GitHub login 是 cao-dev，本站规则不允许连字符，会被转成下划线
	if result.User.Username != "cao_dev" {
		t.Errorf("用户名应由 GitHub login 转换而来, 实际 %s", result.User.Username)
	}
	// 【关键】OAuth 用户不应有密码
	if result.User.HasPassword() {
		t.Error("GitHub 注册的用户不应有密码")
	}
	if result.User.AvatarURL == nil || *result.User.AvatarURL == "" {
		t.Error("应同步 GitHub 头像")
	}
	if result.Tokens.AccessToken == "" {
		t.Error("应签发 access token")
	}
}

// GitHub 登录名允许连字符，本站用户名规则不允许，必须转换。
func TestGitHubLoginSanitizesUsername(t *testing.T) {
	svc, _, _ := newOAuthService()

	result, err := svc.LoginWithGitHub(context.Background(), githubUser(1002, "cao-ling-yun", "c@example.com"))
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	if strings.Contains(result.User.Username, "-") {
		t.Errorf("用户名不应包含连字符, 实际 %s", result.User.Username)
	}
	if result.User.Username != "cao_ling_yun" {
		t.Errorf("连字符应转成下划线, 实际 %s", result.User.Username)
	}
}

// 第二次用同一个 GitHub 账号登录，应当复用已有用户而不是重复创建。
func TestGitHubLoginReusesExistingLink(t *testing.T) {
	svc, users, _ := newOAuthService()
	ctx := context.Background()

	first, err := svc.LoginWithGitHub(ctx, githubUser(1003, "repeat-user", "repeat@example.com"))
	if err != nil {
		t.Fatalf("首次登录失败: %v", err)
	}

	second, err := svc.LoginWithGitHub(ctx, githubUser(1003, "repeat-user", "repeat@example.com"))
	if err != nil {
		t.Fatalf("二次登录失败: %v", err)
	}

	if first.User.ID != second.User.ID {
		t.Errorf("同一 GitHub 账号应映射到同一用户: %d vs %d", first.User.ID, second.User.ID)
	}
	if len(users.users) != 1 {
		t.Errorf("不应重复创建用户, 实际有 %d 个", len(users.users))
	}
}

// 邮箱命中已有本地账号时，应把 GitHub 关联过去，而不是新建用户。
func TestGitHubLoginLinksByVerifiedEmail(t *testing.T) {
	svc, users, _ := newOAuthService()
	ctx := context.Background()

	// 先有一个本地注册的账号
	tokens := jwt.NewManager("t", time.Minute, time.Hour)
	authSvc := service.NewAuthService(users, tokens)
	local, err := authSvc.Register(ctx, dto.RegisterInput{
		Username: "LocalUser",
		Email:    "same@example.com",
		Password: "local-password-123",
	})
	if err != nil {
		t.Fatalf("准备本地账号失败: %v", err)
	}

	// 用同一邮箱的 GitHub 账号登录
	gh, err := svc.LoginWithGitHub(ctx, githubUser(1004, "gh-user", "SAME@example.com"))
	if err != nil {
		t.Fatalf("GitHub 登录失败: %v", err)
	}

	if gh.User.ID != local.User.ID {
		t.Errorf("应关联到已有账号 %d, 实际 %d", local.User.ID, gh.User.ID)
	}
	if len(users.users) != 1 {
		t.Errorf("不应新建用户, 实际有 %d 个", len(users.users))
	}
	// 原有密码不应被抹掉——用户仍然可以用密码登录
	if !gh.User.HasPassword() {
		t.Error("关联后不应丢失原有密码")
	}
}

// 用户名与已有本地用户冲突时，应加随机后缀而不是报错。
func TestGitHubLoginHandlesUsernameCollision(t *testing.T) {
	svc, users, _ := newOAuthService()
	ctx := context.Background()

	tokens := jwt.NewManager("t", time.Minute, time.Hour)
	authSvc := service.NewAuthService(users, tokens)
	if _, err := authSvc.Register(ctx, dto.RegisterInput{
		Username: "occupied",
		Email:    "occupied@example.com",
		Password: "some-password-123",
	}); err != nil {
		t.Fatalf("准备账号失败: %v", err)
	}

	// GitHub login 恰好也是 occupied，但邮箱不同
	gh, err := svc.LoginWithGitHub(ctx, githubUser(1005, "occupied", "different@example.com"))
	if err != nil {
		t.Fatalf("GitHub 登录应成功（自动改名）, 实际 %v", err)
	}

	if gh.User.Username == "occupied" {
		t.Error("用户名冲突时应自动加后缀")
	}
	if !strings.HasPrefix(gh.User.Username, "occupied_") {
		t.Errorf("后缀命名不符预期: %s", gh.User.Username)
	}
}

// 拿不到已验证邮箱时必须拒绝——否则无法安全地关联/创建账号。
func TestGitHubLoginRequiresVerifiedEmail(t *testing.T) {
	svc, _, _ := newOAuthService()

	_, err := svc.LoginWithGitHub(context.Background(), githubUser(1006, "no-email", ""))
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("无邮箱应返回 400, 实际 %v", err)
	}
}

// 没有密码的账号不允许解绑第三方登录，否则会把自己锁在门外。
func TestUnlinkRequiresPassword(t *testing.T) {
	svc, _, _ := newOAuthService()
	ctx := context.Background()

	gh, err := svc.LoginWithGitHub(ctx, githubUser(1007, "unlink-me", "unlink@example.com"))
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	if err := svc.UnlinkGitHub(ctx, gh.User.ID); !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("无密码账号解绑应被拒绝, 实际 %v", err)
	}
}

func TestListLinkedAccounts(t *testing.T) {
	svc, _, _ := newOAuthService()
	ctx := context.Background()

	gh, _ := svc.LoginWithGitHub(ctx, githubUser(1008, "list-me", "list@example.com"))

	accounts, err := svc.ListLinkedAccounts(ctx, gh.User.ID)
	if err != nil {
		t.Fatalf("查询绑定失败: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("应有 1 个绑定, 实际 %d", len(accounts))
	}
	if accounts[0].Provider != model.ProviderGitHub {
		t.Errorf("provider 不符: %s", accounts[0].Provider)
	}
	if accounts[0].ProviderUID != "1008" {
		t.Errorf("provider_uid 不符: %s", accounts[0].ProviderUID)
	}
}

// GitHub 注册的用户用密码登录，应给出可操作的提示。
func TestPasswordLoginOnOAuthAccount(t *testing.T) {
	svc, users, _ := newOAuthService()
	ctx := context.Background()

	gh, _ := svc.LoginWithGitHub(ctx, githubUser(1009, "oauth-only", "oauth@example.com"))

	tokens := jwt.NewManager("t", time.Minute, time.Hour)
	authSvc := service.NewAuthService(users, tokens)

	_, err := authSvc.Login(ctx, dto.LoginInput{Login: "oauth_only", Password: "whatever123"})
	if !errors.Is(err, errs.ErrUnauthorized) {
		t.Fatalf("应返回 401, 实际 %v", err)
	}
	if !strings.Contains(err.Error(), "GitHub") {
		t.Errorf("提示应引导用户使用 GitHub 登录, 实际: %v", err)
	}

	// 用户设置密码后，就应当能用密码登录了
	if err := service.NewUserService(users).ChangePassword(ctx, gh.User.ID, dto.ChangePasswordInput{
		NewPassword: "brand-new-password-1",
	}); err != nil {
		t.Fatalf("OAuth 用户首次设置密码失败: %v", err)
	}

	if _, err := authSvc.Login(ctx, dto.LoginInput{Login: "oauth_only", Password: "brand-new-password-1"}); err != nil {
		t.Fatalf("设置密码后应能用密码登录, 实际 %v", err)
	}
}
