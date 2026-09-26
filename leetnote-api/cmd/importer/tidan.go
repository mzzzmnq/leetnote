// Package main 提供题单导入工具。
//
// 用途：把灵茶山艾府（灵神）的题单 markdown 导入到 problems / tags 表。
//
// 数据来源：https://github.com/EndlessCheng/codeforces-go/blob/master/leetcode/README.md
//
// 为什么写成一个独立的命令行工具而不是接口：
// 导入是运维行为（一次性、需要审阅结果），不该暴露成 HTTP 接口。
package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// TidanEntry 是题单里的一道题。
type TidanEntry struct {
	LeetCodeID int
	Title      string
	Slug       string
	Topic      string // 所属专题（对应「视频精讲」那一列，按分组向下继承）
	Note       string // 备注，如「*课后作业」「最短」
	Homework   bool   // 备注以 * 开头即为课后作业

	// Difficulty 不在题单里，由 import 流程从 LeetCode 接口补齐
	Difficulty string
}

var (
	// 匹配「视频精讲」列： [相向双指针（一）](https://www.bilibili.com/video/BVxxx/)
	reTopic = regexp.MustCompile(`\[([^\]]+)\]\(https://www\.bilibili\.com/video/[^)]+\)`)

	// 匹配「题目」列： [167. 两数之和 II - 输入有序数组](https://leetcode.cn/problems/two-sum-ii-input-array-is-sorted/)
	reProblem = regexp.MustCompile(`\[(\d+)\.\s*([^\]]+)\]\(https://leetcode\.cn/problems/([^/)]+?)/?\)`)
)

// ParseTidan 解析题单 markdown，返回题目列表。
//
// 表格形态（4 列）：
//
//	|视频精讲|题目|代码|备注|
//	|---|---|---|---|
//	|[相向双指针（一）](url)|[167. 两数之和 II](url)|[代码](url)||
//	||[15. 三数之和](url)|[代码](url)|包含两个优化|
//
// 注意第一列【只在每个专题的第一行出现】，后续行为空，
// 所以要「向下继承」当前专题——这是解析这段 markdown 的关键。
func ParseTidan(r io.Reader) ([]TidanEntry, error) {
	scanner := bufio.NewScanner(r)
	// 单行可能很长（题解链接），把缓冲上限提到 1MB
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	entries := make([]TidanEntry, 0, 512)
	currentTopic := ""
	lineNo := 0
	seen := make(map[string]struct{}, 512)

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		// 只处理表格行
		if !strings.HasPrefix(line, "|") {
			continue
		}

		// |a|b|c|d|  → ["", "a", "b", "c", "d", ""]
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		// 跳过表头与分隔行
		if strings.Contains(parts[1], "视频精讲") || strings.HasPrefix(strings.TrimSpace(parts[1]), "---") {
			continue
		}

		// 第一列非空则切换专题
		if topicCell := strings.TrimSpace(parts[1]); topicCell != "" {
			if m := reTopic.FindStringSubmatch(topicCell); m != nil {
				currentTopic = m[1]
			}
		}

		problemCell := ""
		noteCell := ""
		if len(parts) > 2 {
			problemCell = parts[2]
		}
		if len(parts) > 4 {
			noteCell = strings.TrimSpace(parts[4])
		}

		m := reProblem.FindStringSubmatch(problemCell)
		if m == nil {
			continue // 不是题目行（可能是纯文字说明）
		}

		id, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		slug := m[3]

		// 同一道题可能在多个专题里出现，保留首次出现的专题
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}

		entries = append(entries, TidanEntry{
			LeetCodeID: id,
			Title:      strings.TrimSpace(m[2]),
			Slug:       slug,
			Topic:      currentTopic,
			Note:       noteCell,
			Homework:   strings.HasPrefix(noteCell, "*"),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取题单失败: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("没有解析到任何题目，请检查题单文件格式是否变化")
	}
	return entries, nil
}

// Topics 返回去重后的专题列表（保持首次出现的顺序，便于按原顺序建标签）。
func Topics(entries []TidanEntry) []string {
	seen := make(map[string]struct{}, 32)
	out := make([]string, 0, 32)

	for _, e := range entries {
		if e.Topic == "" {
			continue
		}
		if _, ok := seen[e.Topic]; ok {
			continue
		}
		seen[e.Topic] = struct{}{}
		out = append(out, e.Topic)
	}
	return out
}
