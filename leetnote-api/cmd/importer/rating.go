package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// 社区统计的题目难度分。
//
// 来源：https://github.com/zerotrac/leetcode_problem_rating
// 它用竞赛的通过率与参赛者表现反推每题难度，是目前公认最准的一档数据。
// 灵神题单里说的「先完成难度分 ≤ 1700 的题目」用的就是这个分。
// 灵神的「难度练习」插件（huxulm/lc-rating）数据也完全来自这里。
//
// 文件是制表符分隔，首行是表头：
//
//	Rating  ID  Title  Title ZH  Title Slug  Contest Slug  Problem Index
//	3124.50 3743 Maximize... 循环... maximize-... weekly-contest-475 Q4
//
// 【已知局限】这份数据是从【竞赛】表现反推的，而 LeetCode 竞赛从 2018 年
// （约 700 多题）才开始，所以早期经典题（如 1. 两数之和、15. 三数之和、
// 42. 接雨水）没有分数。这不是 bug，是数据源本身的边界，无解。
// 实测题单里的覆盖约 42%，缺的全是竞赛时代之前的老题。
const ratingsFile = "ratings.txt"

// 多个镜像按顺序尝试。
//
// 中国大陆直连 raw.githubusercontent.com 很不稳定（TLS 握手超时是常态），
// jsDelivr 是 GitHub 的 CDN，国内可达性好得多，所以放在第一位。
var ratingsMirrors = []string{
	"https://cdn.jsdelivr.net/gh/zerotrac/leetcode_problem_rating@master/" + ratingsFile,
	"https://raw.githubusercontent.com/zerotrac/leetcode_problem_rating/master/" + ratingsFile,
}

// FetchRatings 依次尝试各镜像，返回 slug -> 分数。
func FetchRatings(ctx context.Context) (map[string]float64, error) {
	client := &http.Client{Timeout: 60 * time.Second}

	var lastErr error
	for _, url := range ratingsMirrors {
		for attempt := range 2 {
			if attempt > 0 {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}

			body, err := download(ctx, client, url)
			if err != nil {
				lastErr = fmt.Errorf("%s: %w", hostOf(url), err)
				continue
			}

			ratings, err := parseRatings(body)
			if err != nil {
				lastErr = fmt.Errorf("%s: %w", hostOf(url), err)
				continue
			}
			return ratings, nil
		}
	}

	return nil, lastErr
}

func download(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "leetnote-importer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("返回 %d", resp.StatusCode)
	}

	// 16MB 上限，正常文件约 370KB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	return body, nil
}

func hostOf(url string) string {
	if i := strings.Index(url, "//"); i >= 0 {
		rest := url[i+2:]
		if j := strings.IndexByte(rest, '/'); j >= 0 {
			return rest[:j]
		}
	}
	return url
}

// LoadRatingsFile 从本地文件读取难度分。
//
// 网络确实不通时（公司代理、离线环境），可以自己下载一份再传进来：
//
//	go run ./cmd/importer -file xxx.md -ratings-file D:\dev\_downloads\ratings.txt
func LoadRatingsFile(path string) (map[string]float64, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取难度分文件失败: %w", err)
	}
	return parseRatings(body)
}

// parseRatings 解析制表符分隔的难度分文件。
//
// 单条记录格式错误只跳过该条、不影响整体 —— 外部数据不该让整个导入失败。
func parseRatings(body []byte) (map[string]float64, error) {
	// 去掉可能存在的 UTF-8 BOM，否则首列解析会带上不可见字符
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})

	out := make(map[string]float64, 3000)

	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		// 跳过表头
		if lineNo == 1 || strings.HasPrefix(line, "Rating") {
			continue
		}

		cols := strings.Split(line, "\t")
		if len(cols) < 5 {
			continue
		}

		rating, err := strconv.ParseFloat(strings.TrimSpace(cols[0]), 64)
		if err != nil {
			continue
		}

		slug := strings.TrimSpace(cols[4])
		if slug == "" {
			continue
		}

		out[slug] = rating
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("解析难度分文件失败: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("难度分文件解析出 0 条记录，格式可能变了")
	}
	return out, nil
}
