package model

import "time"

// APIToken 是长期访问令牌（PAT）的元数据。
//
// 【刻意不含明文】：明文只在生成时返回一次，库里只有哈希。
// 所以这张表的对象天然是「不可逆」的，别试图加一个 Raw 字段。
type APIToken struct {
	ID         int64
	UserID     int64
	Name       string
	TokenHash  string
	LastUsedAt *time.Time
	CreatedAt  time.Time
}
