package model

import "time"

// User 对应 users 表。
//
// 字段用指针表示数据库里可为 NULL 的列（avatar_url / bio），
// 这样能区分「没设置」（nil）和「设置为空字符串」两种语义。
type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	AvatarURL    *string
	Bio          *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
