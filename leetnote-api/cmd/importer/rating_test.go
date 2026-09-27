package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 取自真实 ratings.txt 的片段（制表符分隔，含表头）
const sampleRatings = "Rating\tID\tTitle\tTitle ZH\tTitle Slug\tContest Slug\tProblem Index\n" +
	"3124.5016688063\t3743\tMaximize Cyclic Partition Score\t循环最大值分割得分\tmaximize-cyclic-partition-score\tweekly-contest-475\tQ4\n" +
	"1710.9105378431\t15\t3Sum\t三数之和\t3sum\tbiweekly-contest-1\tQ2\n" +
	"1389.0328392117\t1\tTwo Sum\t两数之和\ttwo-sum\tweekly-contest-1\tQ1\n"

func TestParseRatings(t *testing.T) {
	ratings, err := parseRatings([]byte(sampleRatings))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(ratings) != 3 {
		t.Fatalf("应解析出 3 条, 实际 %d", len(ratings))
	}

	cases := map[string]float64{
		"maximize-cyclic-partition-score": 3124.5016688063,
		"3sum":                            1710.9105378431,
		"two-sum":                         1389.0328392117,
	}
	for slug, want := range cases {
		got, ok := ratings[slug]
		if !ok {
			t.Errorf("缺少 slug %q", slug)
			continue
		}
		// 浮点比较留一点误差
		if diff := got - want; diff > 1e-6 || diff < -1e-6 {
			t.Errorf("%s 的分数错误: 期望 %v, 实际 %v", slug, want, got)
		}
	}
}

// 表头不能被当成数据行 —— 否则会多出一条 slug 为 "Title Slug" 的脏数据
func TestParseRatingsSkipsHeader(t *testing.T) {
	ratings, err := parseRatings([]byte(sampleRatings))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if _, ok := ratings["Title Slug"]; ok {
		t.Error("表头被当成了数据行")
	}
}

// UTF-8 BOM 会让首列（分数）解析失败，整份文件全军覆没
func TestParseRatingsHandlesBOM(t *testing.T) {
	withBOM := "\xEF\xBB\xBF" + sampleRatings

	ratings, err := parseRatings([]byte(withBOM))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(ratings) != 3 {
		t.Fatalf("BOM 导致解析失败: 只得到 %d 条", len(ratings))
	}
	if _, ok := ratings["two-sum"]; !ok {
		t.Error("BOM 处理有误，two-sum 缺失")
	}
}

// 单条脏数据只跳过该条，不能让整份文件失败
func TestParseRatingsSkipsBadRows(t *testing.T) {
	input := "Rating\tID\tTitle\tTitle ZH\tTitle Slug\tContest Slug\tProblem Index\n" +
		"abc\t1\tBad\t坏行\tbad-rating\tc\tQ1\n" + // 分数不是数字
		"1500\t2\tNo Slug\t没有slug\t\tc\tQ1\n" + // slug 为空
		"1600\t3\tShort\t列数不够\n" + // 列数不足
		"1700\t4\tGood\t正常\tgood-slug\tc\tQ1\n"

	ratings, err := parseRatings([]byte(input))
	if err != nil {
		t.Fatalf("不该整体失败: %v", err)
	}
	if len(ratings) != 1 {
		t.Fatalf("应只保留 1 条有效记录, 实际 %d", len(ratings))
	}
	if _, ok := ratings["good-slug"]; !ok {
		t.Error("有效记录被误删")
	}
}

// 全部解析不出内容时必须报错，而不是静默返回空 map ——
// 否则导入会「成功」但一条难度分都没写进去，很难排查
func TestParseRatingsRejectsEmpty(t *testing.T) {
	if _, err := parseRatings([]byte("Rating\tID\tTitle\tTitle ZH\tTitle Slug\tContest Slug\tProblem Index\n")); err == nil {
		t.Error("只有表头时应该报错")
	}
}

// 数据源的格式一旦变化（比如列顺序调整），必须能被发现
func TestParseRatingsRejectsGarbage(t *testing.T) {
	if _, err := parseRatings([]byte("这不是一个 TSV 文件")); err == nil {
		t.Error("完全无关的内容应该报错")
	}
}

func TestLoadRatingsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ratings.txt")

	if err := os.WriteFile(path, []byte(sampleRatings), 0o600); err != nil {
		t.Fatalf("写临时文件失败: %v", err)
	}

	ratings, err := LoadRatingsFile(path)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if len(ratings) != 3 {
		t.Errorf("应读出 3 条, 实际 %d", len(ratings))
	}

	if _, err := LoadRatingsFile(filepath.Join(dir, "不存在.txt")); err == nil {
		t.Error("文件不存在时应该报错")
	}
}

// hostOf 用于把镜像 URL 缩减成可读的错误信息
func TestHostOf(t *testing.T) {
	cases := map[string]string{
		"https://cdn.jsdelivr.net/gh/x/y@master/ratings.txt": "cdn.jsdelivr.net",
		"http://example.com/a":                               "example.com",
		"没有协议":                                               "没有协议",
	}
	for in, want := range cases {
		if got := hostOf(in); got != want {
			t.Errorf("hostOf(%q) = %q, 期望 %q", in, got, want)
		}
	}
}

// 镜像列表里必须有多个源 —— 单点失败时整个导入就拉不到难度分
func TestRatingsMirrorsHasFallback(t *testing.T) {
	if len(ratingsMirrors) < 2 {
		t.Fatalf("应至少配置 2 个镜像, 实际 %d", len(ratingsMirrors))
	}
	for _, u := range ratingsMirrors {
		if !strings.HasPrefix(u, "https://") {
			t.Errorf("镜像地址应为 https: %q", u)
		}
		if !strings.HasSuffix(u, ratingsFile) {
			t.Errorf("镜像地址应指向 %s: %q", ratingsFile, u)
		}
	}
}
