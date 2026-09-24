package dto

import (
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/model"
)

// UserResponse 是用户信息对外的表示。
//
// 【安全要点】这里刻意没有 PasswordHash 字段。
// 用专门的 DTO 而不是直接把 model.User 序列化返回，
// 就是为了从结构上杜绝「不小心把密码哈希发给前端」这类事故。
type UserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AvatarURL *string   `json:"avatar_url"`
	Bio       *string   `json:"bio"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponse(u *model.User) UserResponse {
	if u == nil {
		return UserResponse{}
	}
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		Bio:       u.Bio,
		CreatedAt: u.CreatedAt,
	}
}

// UpdateProfileInput 是修改个人资料的请求体。
//
// 用指针是为了区分两种语义：
//   - 字段不传（nil）    → 不修改
//   - 传了空字符串（""） → 清空
type UpdateProfileInput struct {
	AvatarURL *string `json:"avatar_url" binding:"omitempty,url,max=500"`
	Bio       *string `json:"bio"        binding:"omitempty,max=200"`
}

// ChangePasswordInput 是修改密码的请求体。
//
// 必须提供旧密码：即使 access_token 被盗用，
// 攻击者也无法直接改密码把账号锁死（纵深防御）。
type ChangePasswordInput struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}
