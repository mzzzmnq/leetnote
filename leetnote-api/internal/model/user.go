package model

import "time"

// User 对应 users 表。
//
// 字段用指针表示数据库里可为 NULL 的列，这样能区分
// 「没有值」（nil）和「值为空字符串」（""）两种语义。
type User struct {
	ID       int64
	Username string
	Email    string

	// PasswordHash 为 nil 表示该账号【没有密码】——即通过 OAuth 注册的用户。
	// 这类账号不能用密码登录，修改密码时也不需要提供旧密码。
	PasswordHash *string

	AvatarURL *string
	Bio       *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// HasPassword 判断该账号是否设置了密码。
func (u *User) HasPassword() bool {
	return u.PasswordHash != nil && *u.PasswordHash != ""
}
