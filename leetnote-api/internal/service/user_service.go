package service

import (
	"context"
	"errors"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/hash"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

type UserService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) *UserService {
	return &UserService{users: users}
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return s.users.GetByID(ctx, id)
}

// UpdateProfile 修改昵称之外的资料字段（头像、简介）。
func (s *UserService) UpdateProfile(ctx context.Context, id int64, in dto.UpdateProfileInput) (*model.User, error) {
	return s.users.UpdateProfile(ctx, id, in.AvatarURL, in.Bio)
}

// ChangePassword 修改密码。
//
// 要求提供旧密码是纵深防御：即使 access_token 被盗，
// 攻击者也无法直接改密码把账号锁死。
func (s *UserService) ChangePassword(ctx context.Context, id int64, in dto.ChangePasswordInput) error {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !hash.VerifyPassword(u.PasswordHash, in.OldPassword) {
		return errs.ErrBadRequest.WithMessage("原密码不正确")
	}

	if in.OldPassword == in.NewPassword {
		return errs.ErrBadRequest.WithMessage("新密码不能与原密码相同")
	}

	newHash, err := hash.HashPassword(in.NewPassword)
	if err != nil {
		if errors.Is(err, hash.ErrPasswordTooLong) {
			return errs.ErrValidation.WithDetails(map[string]string{
				"new_password": "密码长度不能超过 72 字节",
			})
		}
		return errs.ErrInternal.Wrap(err)
	}

	return s.users.UpdatePassword(ctx, id, newHash)
}
