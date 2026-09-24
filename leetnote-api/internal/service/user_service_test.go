package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/hash"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
	"github.com/mzzzmnq/leetnote-api/internal/service"
)

const (
	originalPassword = "original-password-123"
	newPassword      = "brand-new-password-456"
)

// setupUserService 先通过 AuthService 造一个真实用户（密码是真哈希过的），
// 再返回 UserService 供测试。
func setupUserService(t *testing.T) (*service.UserService, *fakeUserRepo, int64) {
	t.Helper()

	repo := newFakeUserRepo()
	tokens := jwt.NewManager("user-service-test-secret", 15*time.Minute, 24*time.Hour)
	authSvc := service.NewAuthService(repo, tokens)

	result, err := authSvc.Register(context.Background(), dto.RegisterInput{
		Username: "TestUser",
		Email:    "test@example.com",
		Password: originalPassword,
	})
	if err != nil {
		t.Fatalf("准备测试用户失败: %v", err)
	}

	return service.NewUserService(repo), repo, result.User.ID
}

func TestChangePasswordSucceeds(t *testing.T) {
	svc, repo, id := setupUserService(t)
	ctx := context.Background()

	err := svc.ChangePassword(ctx, id, dto.ChangePasswordInput{
		OldPassword: originalPassword,
		NewPassword: newPassword,
	})
	if err != nil {
		t.Fatalf("修改密码失败: %v", err)
	}

	u, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("读取用户失败: %v", err)
	}
	if u.PasswordHash == nil || !hash.VerifyPassword(*u.PasswordHash, newPassword) {
		t.Error("新密码应能校验通过")
	}
	if hash.VerifyPassword(*u.PasswordHash, originalPassword) {
		t.Error("旧密码应已失效")
	}
}

func TestChangePasswordRejectsWrongOldPassword(t *testing.T) {
	svc, repo, id := setupUserService(t)
	ctx := context.Background()

	err := svc.ChangePassword(ctx, id, dto.ChangePasswordInput{
		OldPassword: "definitely-not-the-old-password",
		NewPassword: newPassword,
	})
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("旧密码错误应返回 400, 实际 %v", err)
	}

	// 校验失败时绝不能改动密码
	u, _ := repo.GetByID(ctx, id)
	if !hash.VerifyPassword(*u.PasswordHash, originalPassword) {
		t.Error("校验失败时不应修改密码")
	}
}

func TestChangePasswordRejectsSameAsOld(t *testing.T) {
	svc, _, id := setupUserService(t)

	err := svc.ChangePassword(context.Background(), id, dto.ChangePasswordInput{
		OldPassword: originalPassword,
		NewPassword: originalPassword,
	})
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("新旧密码相同应返回 400, 实际 %v", err)
	}
}

func TestChangePasswordUserNotFound(t *testing.T) {
	svc, _, _ := setupUserService(t)

	err := svc.ChangePassword(context.Background(), 99999, dto.ChangePasswordInput{
		OldPassword: originalPassword,
		NewPassword: newPassword,
	})
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("用户不存在应返回 404, 实际 %v", err)
	}
}

// UpdateProfile 的指针语义：nil 表示「不修改」，空串表示「清空」。
func TestUpdateProfilePointerSemantics(t *testing.T) {
	svc, _, id := setupUserService(t)
	ctx := context.Background()

	// 先设置两个字段
	bio := "刷题中"
	avatar := "https://example.com/a.png"
	if _, err := svc.UpdateProfile(ctx, id, dto.UpdateProfileInput{AvatarURL: &avatar, Bio: &bio}); err != nil {
		t.Fatalf("设置资料失败: %v", err)
	}

	// 只改 bio，avatar 传 nil 应保持不变
	newBio := "专注动态规划"
	u, err := svc.UpdateProfile(ctx, id, dto.UpdateProfileInput{Bio: &newBio})
	if err != nil {
		t.Fatalf("更新资料失败: %v", err)
	}
	if u.Bio == nil || *u.Bio != "专注动态规划" {
		t.Errorf("bio 未更新: %v", u.Bio)
	}
	if u.AvatarURL == nil || *u.AvatarURL != avatar {
		t.Errorf("avatar 传 nil 时不应被修改, 实际 %v", u.AvatarURL)
	}

	// 传空串应清空
	empty := ""
	u, err = svc.UpdateProfile(ctx, id, dto.UpdateProfileInput{Bio: &empty})
	if err != nil {
		t.Fatalf("清空 bio 失败: %v", err)
	}
	if u.Bio == nil || *u.Bio != "" {
		t.Errorf("传空串应清空 bio, 实际 %v", u.Bio)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	svc, _, _ := setupUserService(t)

	if _, err := svc.GetByID(context.Background(), 99999); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("应返回 404, 实际 %v", err)
	}
}
