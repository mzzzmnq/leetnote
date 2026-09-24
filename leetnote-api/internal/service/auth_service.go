package service

import (
	"context"
	"errors"
	"strings"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/hash"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

// AuthResult 是认证成功后的内部结果。
//
// 单独定义而不是直接返回 dto：service 层不应该关心
// 「ExpiresIn 用秒还是毫秒」这类 HTTP 表现层细节。
type AuthResult struct {
	User   *model.User
	Tokens *jwt.TokenPair
}

type AuthService struct {
	users  repository.UserRepository
	tokens *jwt.Manager
}

func NewAuthService(users repository.UserRepository, tokens *jwt.Manager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

// Register 创建账号并直接签发 Token（注册即登录）。
func (s *AuthService) Register(ctx context.Context, in dto.RegisterInput) (*AuthResult, error) {
	passwordHash, err := hash.HashPassword(in.Password)
	if err != nil {
		if errors.Is(err, hash.ErrPasswordTooLong) {
			return nil, errs.ErrValidation.WithDetails(map[string]string{
				"password": "密码长度不能超过 72 字节",
			})
		}
		return nil, errs.ErrInternal.Wrap(err)
	}

	u := &model.User{
		Username:     in.Username, // 保留原始大小写用于展示
		Email:        strings.ToLower(in.Email),
		PasswordHash: passwordHash,
	}

	// 【不预先查询用户名是否已存在】——那会产生竞态：
	// 两个并发请求可能同时查到「不存在」然后都去插入。
	// 直接插入，让数据库的唯一索引兜底，冲突会返回 409。
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}

	return s.issue(u)
}

// Login 校验凭证并签发 Token。
func (s *AuthService) Login(ctx context.Context, in dto.LoginInput) (*AuthResult, error) {
	u, err := s.users.GetByLogin(ctx, in.Login)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			// 用户不存在时也跑一次 bcrypt，让耗时与「密码错误」一致。
			// 否则攻击者能通过响应时间差异枚举出哪些用户名真实存在。
			hash.DummyVerify()
			return nil, errInvalidCredentials()
		}
		return nil, err
	}

	if !hash.VerifyPassword(u.PasswordHash, in.Password) {
		return nil, errInvalidCredentials()
	}

	return s.issue(u)
}

// Refresh 用 refresh token 换一对新 Token（同时轮换 refresh token）。
//
// 为什么轮换：refresh token 有效期长达 7 天，一旦泄漏危害很大。
// 每次刷新都作废旧的那个，可以把泄漏窗口压缩到「上一次刷新为止」。
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := s.tokens.Parse(refreshToken, jwt.TypeRefresh)
	if err != nil {
		return nil, errs.ErrUnauthorized.
			WithMessage("登录状态已过期，请重新登录").
			Wrap(err)
	}

	// 重新查库：用户可能已被删除，或需要拿到最新的用户信息
	u, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	return s.issue(u)
}

func (s *AuthService) issue(u *model.User) (*AuthResult, error) {
	pair, err := s.tokens.GeneratePair(u.ID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return &AuthResult{User: u, Tokens: pair}, nil
}

// errInvalidCredentials 统一「用户不存在」与「密码错误」的对外表现，
// 防止账号枚举（user enumeration）。
func errInvalidCredentials() *errs.AppError {
	return errs.ErrUnauthorized.WithMessage("用户名或密码错误")
}
