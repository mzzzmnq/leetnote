package service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
	"github.com/mzzzmnq/leetnote-api/internal/service"
)

// ---------------------------------------------------------------
// 内存版 UserRepository
//
// 这就是把 UserRepository 定义成接口的价值：service 层的测试
// 完全不需要数据库，毫秒级跑完，CI 里也不会因为连不上 PG 而红。
// ---------------------------------------------------------------

type fakeUserRepo struct {
	mu     sync.Mutex
	users  map[int64]*model.User
	nextID int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: make(map[int64]*model.User), nextID: 1}
}

func (f *fakeUserRepo) Create(_ context.Context, u *model.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 模拟数据库上的大小写不敏感唯一索引（uq_users_*_lower）
	for _, existing := range f.users {
		if strings.EqualFold(existing.Username, u.Username) {
			return errs.ErrUsernameTaken
		}
		if strings.EqualFold(existing.Email, u.Email) {
			return errs.ErrEmailTaken
		}
	}

	u.ID = f.nextID
	f.nextID++
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt

	clone := *u
	f.users[u.ID] = &clone
	return nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, u := range f.users {
		if strings.EqualFold(u.Email, email) {
			clone := *u
			return &clone, nil
		}
	}
	return nil, errs.ErrNotFound.WithMessage("用户不存在")
}

func (f *fakeUserRepo) GetByID(_ context.Context, id int64) (*model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.users[id]
	if !ok {
		return nil, errs.ErrNotFound.WithMessage("用户不存在")
	}
	clone := *u
	return &clone, nil
}

func (f *fakeUserRepo) GetByLogin(_ context.Context, login string) (*model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, u := range f.users {
		if strings.EqualFold(u.Username, login) || strings.EqualFold(u.Email, login) {
			clone := *u
			return &clone, nil
		}
	}
	return nil, errs.ErrNotFound.WithMessage("用户不存在")
}

func (f *fakeUserRepo) UpdateProfile(_ context.Context, id int64, avatarURL, bio *string) (*model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.users[id]
	if !ok {
		return nil, errs.ErrNotFound.WithMessage("用户不存在")
	}
	if avatarURL != nil {
		u.AvatarURL = avatarURL
	}
	if bio != nil {
		u.Bio = bio
	}
	clone := *u
	return &clone, nil
}

func (f *fakeUserRepo) UpdatePassword(_ context.Context, id int64, passwordHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.users[id]
	if !ok {
		return errs.ErrNotFound.WithMessage("用户不存在")
	}
	u.PasswordHash = &passwordHash
	return nil
}

// ---------------------------------------------------------------
// 测试
// ---------------------------------------------------------------

func newAuthService() (*service.AuthService, *fakeUserRepo) {
	repo := newFakeUserRepo()
	tokens := jwt.NewManager("service-test-secret", 15*time.Minute, 24*time.Hour)
	return service.NewAuthService(repo, tokens), repo
}

func validRegisterInput() dto.RegisterInput {
	return dto.RegisterInput{
		Username: "CaoLingyun",
		Email:    "Cao@Example.com",
		Password: "supersecret123",
	}
}

func TestRegisterCreatesUserAndIssuesTokens(t *testing.T) {
	svc, _ := newAuthService()

	result, err := svc.Register(context.Background(), validRegisterInput())
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	if result.User.ID == 0 {
		t.Error("应回填用户 ID")
	}
	if result.User.PasswordHash == nil || *result.User.PasswordHash == "" ||
		*result.User.PasswordHash == "supersecret123" {
		t.Error("密码必须是哈希后的值，不能明文存储")
	}
	if !result.User.HasPassword() {
		t.Error("本地注册用户应当有密码")
	}
	if result.User.Email != "cao@example.com" {
		t.Errorf("邮箱应归一化为小写, 实际 %s", result.User.Email)
	}
	if result.User.Username != "CaoLingyun" {
		t.Errorf("用户名应保留原始大小写用于展示, 实际 %s", result.User.Username)
	}
	if result.Tokens.AccessToken == "" || result.Tokens.RefreshToken == "" {
		t.Error("应签发双 Token")
	}
}

func TestRegisterRejectsDuplicateUsername(t *testing.T) {
	svc, _ := newAuthService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, validRegisterInput()); err != nil {
		t.Fatalf("首次注册失败: %v", err)
	}

	dup := validRegisterInput()
	dup.Email = "another@example.com" // 只让用户名冲突
	if _, err := svc.Register(ctx, dup); !errors.Is(err, errs.ErrUsernameTaken) {
		t.Fatalf("重复用户名应返回冲突错误, 实际 %v", err)
	}
}

func TestRegisterTreatsUsernameCaseInsensitively(t *testing.T) {
	svc, _ := newAuthService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, validRegisterInput()); err != nil {
		t.Fatalf("首次注册失败: %v", err)
	}

	dup := validRegisterInput()
	dup.Username = "caolingyun" // 仅大小写不同
	dup.Email = "another@example.com"

	if _, err := svc.Register(ctx, dup); !errors.Is(err, errs.ErrUsernameTaken) {
		t.Fatalf("大小写不同的同名用户应被拒绝, 实际 %v", err)
	}
}

func TestLoginWithEmailCaseInsensitive(t *testing.T) {
	svc, _ := newAuthService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, validRegisterInput()); err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	result, err := svc.Login(ctx, dto.LoginInput{Login: "CAO@example.COM", Password: "supersecret123"})
	if err != nil {
		t.Fatalf("用邮箱登录失败: %v", err)
	}
	if result.User.Username != "CaoLingyun" {
		t.Errorf("返回了错误的用户: %s", result.User.Username)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	svc, _ := newAuthService()
	ctx := context.Background()
	_, _ = svc.Register(ctx, validRegisterInput())

	_, err := svc.Login(ctx, dto.LoginInput{Login: "CaoLingyun", Password: "wrong-password"})
	if !errors.Is(err, errs.ErrUnauthorized) {
		t.Fatalf("错误密码应返回 401, 实际 %v", err)
	}
}

// 安全底线：用户不存在与密码错误必须返回【完全一样】的错误，
// 否则攻击者能借此枚举出哪些用户名真实存在。
func TestLoginDoesNotRevealWhetherUserExists(t *testing.T) {
	svc, _ := newAuthService()
	ctx := context.Background()
	_, _ = svc.Register(ctx, validRegisterInput())

	_, errNoUser := svc.Login(ctx, dto.LoginInput{Login: "nobody-here", Password: "whatever123"})
	_, errBadPwd := svc.Login(ctx, dto.LoginInput{Login: "CaoLingyun", Password: "wrong-password"})

	if errNoUser == nil || errBadPwd == nil {
		t.Fatal("两种失败场景都应返回错误")
	}
	if errNoUser.Error() != errBadPwd.Error() {
		t.Errorf("错误信息不应有差异，否则会泄漏账号是否存在:\n  用户不存在: %v\n  密码错误:   %v",
			errNoUser, errBadPwd)
	}
}

func TestRefreshIssuesNewTokens(t *testing.T) {
	svc, _ := newAuthService()
	ctx := context.Background()

	reg, err := svc.Register(ctx, validRegisterInput())
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	result, err := svc.Refresh(ctx, reg.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("刷新失败: %v", err)
	}
	if result.Tokens.AccessToken == "" {
		t.Error("应签发新的 access token")
	}
	if result.User.ID != reg.User.ID {
		t.Error("应返回同一个用户")
	}
}

// 用 access token 去刷新必须失败（Token 类型混用攻击）。
func TestRefreshRejectsAccessToken(t *testing.T) {
	svc, _ := newAuthService()
	ctx := context.Background()

	reg, _ := svc.Register(ctx, validRegisterInput())

	if _, err := svc.Refresh(ctx, reg.Tokens.AccessToken); !errors.Is(err, errs.ErrUnauthorized) {
		t.Fatalf("用 access token 刷新应被拒绝, 实际 %v", err)
	}
}

func TestRefreshRejectsGarbage(t *testing.T) {
	svc, _ := newAuthService()

	if _, err := svc.Refresh(context.Background(), "not-a-token"); !errors.Is(err, errs.ErrUnauthorized) {
		t.Fatalf("非法 token 应被拒绝, 实际 %v", err)
	}
}
