package main

import (
	"strings"
	"testing"
)

// 取自真实题单的片段（保留了「第一列只在专题首行出现」这一关键特征）
const sampleTidan = `# 基础算法精讲·题目汇总

大家好，我是 [灵茶山艾府](https://space.bilibili.com/206214)。

|视频精讲|题目|代码|备注|
|---|---|---|---|
|[相向双指针（一）](https://www.bilibili.com/video/BV1bP411c7oJ/)|[167. 两数之和 II - 输入有序数组](https://leetcode.cn/problems/two-sum-ii-input-array-is-sorted/)|[代码](https://leetcode.cn/problems/x/solution/y/)||
||[15. 三数之和](https://leetcode.cn/problems/3sum/)|[代码](https://leetcode.cn/problems/3sum/solution/z/)|包含两个优化|
||[2824. 统计和小于目标的下标对数目](https://leetcode.cn/problems/count-pairs-whose-sum-is-less-than-target/)|[代码](https://leetcode.cn/problems/a/solution/b/)|*课后作业|
|[滑动窗口](https://www.bilibili.com/video/BV1hd4y1r7Gq/)|[209. 长度最小的子数组](https://leetcode.cn/problems/minimum-size-subarray-sum/)|[代码](https://leetcode.cn/problems/c/solution/d/)|最短|
||[3. 无重复字符的最长子串](https://leetcode.cn/problems/longest-substring-without-repeating-characters/)|[代码](https://leetcode.cn/problems/e/solution/f/)|最长|
`

func TestParseTidan(t *testing.T) {
	entries, err := ParseTidan(strings.NewReader(sampleTidan))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(entries) != 5 {
		t.Fatalf("应解析出 5 道题, 实际 %d", len(entries))
	}

	first := entries[0]
	if first.LeetCodeID != 167 {
		t.Errorf("题号错误: %d", first.LeetCodeID)
	}
	if first.Title != "两数之和 II - 输入有序数组" {
		t.Errorf("标题错误: %q", first.Title)
	}
	if first.Slug != "two-sum-ii-input-array-is-sorted" {
		t.Errorf("slug 错误: %q", first.Slug)
	}
	if first.Topic != "相向双指针（一）" {
		t.Errorf("专题错误: %q", first.Topic)
	}
	if first.Homework {
		t.Error("首行不是课后作业")
	}
}

// 题单最关键的结构特征：第一列只在专题首行出现，后续行要向下继承。
func TestParseTidanCarriesTopicForward(t *testing.T) {
	entries, err := ParseTidan(strings.NewReader(sampleTidan))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	want := map[string]string{
		"two-sum-ii-input-array-is-sorted": "相向双指针（一）",
		"3sum":                             "相向双指针（一）", // 第一列为空，继承上一行
		"count-pairs-whose-sum-is-less-than-target":      "相向双指针（一）",
		"minimum-size-subarray-sum":                      "滑动窗口", // 第一列有值，切换专题
		"longest-substring-without-repeating-characters": "滑动窗口",
	}

	for slug, topic := range want {
		found := false
		for _, e := range entries {
			if e.Slug == slug {
				found = true
				if e.Topic != topic {
					t.Errorf("%s 的专题应为 %q, 实际 %q", slug, topic, e.Topic)
				}
			}
		}
		if !found {
			t.Errorf("没有解析到 %s", slug)
		}
	}
}

func TestParseTidanDetectsHomework(t *testing.T) {
	entries, err := ParseTidan(strings.NewReader(sampleTidan))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	var homework, normal int
	for _, e := range entries {
		if e.Homework {
			homework++
			if e.Note != "*课后作业" {
				t.Errorf("课后作业的备注应保留原文, 实际 %q", e.Note)
			}
		} else {
			normal++
		}
	}

	if homework != 1 {
		t.Errorf("应有 1 道课后作业, 实际 %d", homework)
	}
	if normal != 4 {
		t.Errorf("应有 4 道非课后作业, 实际 %d", normal)
	}
}

// 同一道题在多个专题里出现时只保留首次，避免重复导入。
func TestParseTidanDeduplicates(t *testing.T) {
	dup := `|视频精讲|题目|代码|备注|
|---|---|---|---|
|[专题A](https://www.bilibili.com/video/BV1/)|[1. 两数之和](https://leetcode.cn/problems/two-sum/)|[代码](x)||
|[专题B](https://www.bilibili.com/video/BV2/)|[1. 两数之和](https://leetcode.cn/problems/two-sum/)|[代码](x)||
`
	entries, err := ParseTidan(strings.NewReader(dup))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("重复题目应只保留一条, 实际 %d", len(entries))
	}
	if entries[0].Topic != "专题A" {
		t.Errorf("应保留首次出现的专题, 实际 %q", entries[0].Topic)
	}
}

func TestParseTidanRejectsEmptyInput(t *testing.T) {
	if _, err := ParseTidan(strings.NewReader("# 只有标题\n没有表格\n")); err == nil {
		t.Error("没有题目时应返回错误，而不是静默返回空结果")
	}
}

// 题单格式如果变了（比如改成无序列表），应该明确报错而不是导入 0 条。
func TestParseTidanIgnoresNonTableLines(t *testing.T) {
	mixed := `一些说明文字
- [1. 两数之和](https://leetcode.cn/problems/two-sum/)

|视频精讲|题目|代码|备注|
|---|---|---|---|
|[专题](https://www.bilibili.com/video/BV1/)|[15. 三数之和](https://leetcode.cn/problems/3sum/)|[代码](x)||

更多说明
`
	entries, err := ParseTidan(strings.NewReader(mixed))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("只应解析表格里的题目, 实际 %d", len(entries))
	}
	if entries[0].Slug != "3sum" {
		t.Errorf("解析到了错误的题目: %s", entries[0].Slug)
	}
}

func TestTopicsPreservesOrderAndDeduplicates(t *testing.T) {
	entries := []TidanEntry{
		{Topic: "A"}, {Topic: "B"}, {Topic: "A"}, {Topic: ""}, {Topic: "C"},
	}
	got := Topics(entries)

	want := []string{"A", "B", "C"}
	if len(got) != len(want) {
		t.Fatalf("专题数应为 %d, 实际 %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 个专题应为 %q, 实际 %q（保持了首次出现顺序）", i, want[i], got[i])
		}
	}
}
