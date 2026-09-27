package model

import "time"

// 标签分类。
const (
	TagKindAlgorithm     = "algorithm"      // 算法思想：动态规划、双指针
	TagKindDataStructure = "data_structure" // 数据结构：栈、并查集
	TagKindTopic         = "topic"          // 主题：面试高频、错题
)

var TagKindValues = []string{TagKindAlgorithm, TagKindDataStructure, TagKindTopic}

// Tag 对应 tags 表。
//
// Name 是展示名（中文），Slug 是机器名（英文）。
// 之所以两个都存：URL、筛选参数用 slug 更稳（不用 encode 中文），
// 展示用 name 更友好。
type Tag struct {
	ID        int64
	Name      string
	Slug      string
	Kind      string
	CreatedAt time.Time

	// SortOrder 是该标签在题单里的出现顺序（0 表示非题单标签）。
	// 用于还原「相向双指针 → 滑动窗口 → 二分查找」这样的专题顺序。
	SortOrder int
}
