package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

// 导入灵茶山艾府（灵神）的题单。
//
// 用法：
//
//	go run ./cmd/importer -file D:\dev\_downloads\lingshen-tidan.md
//	go run ./cmd/importer -file xxx.md -dry-run
//	go run ./cmd/importer -file xxx.md -max 20        # 试跑前 20 道
//
// 工具是【幂等】的：重复执行不会产生重复数据，只会更新已有记录。
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ 导入失败: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		file      string
		dryRun    bool
		skipFetch bool
		fallback  string
		maxN      int
	)
	flag.StringVar(&file, "file", "", "题单 markdown 文件路径（必填）")
	flag.BoolVar(&dryRun, "dry-run", false, "只解析并打印统计，不写数据库")
	flag.BoolVar(&skipFetch, "skip-fetch", false, "跳过 LeetCode 接口调用（难度用 fallback 值填充）")
	flag.StringVar(&fallback, "fallback-difficulty", "Medium", "LeetCode 上找不到对应题目时使用的难度")
	flag.IntVar(&maxN, "max", 0, "最多导入多少道题（0 表示不限，用于试跑）")
	flag.Parse()

	if file == "" {
		flag.Usage()
		return fmt.Errorf("-file 是必填参数")
	}

	// ---------- 1. 解析题单 ----------
	fmt.Println("📖 解析题单...")
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("打开题单文件失败: %w", err)
	}
	defer f.Close()

	entries, err := ParseTidan(f)
	if err != nil {
		return err
	}

	homework := 0
	for _, e := range entries {
		if e.Homework {
			homework++
		}
	}
	fmt.Printf("   题目 %d 道（课后作业 %d 道），专题 %d 个\n",
		len(entries), homework, len(Topics(entries)))

	if maxN > 0 && len(entries) > maxN {
		entries = entries[:maxN]
		fmt.Printf("   （已截断到前 %d 道）\n", maxN)
	}

	// ---------- 2. 补齐难度 ----------
	difficulties := map[string]string{}
	if skipFetch {
		fmt.Printf("⏭  跳过 LeetCode 接口，全部使用 fallback 难度 %s\n", fallback)
	} else {
		fmt.Println("🌐 从 LeetCode 拉取题目难度（题单里没有这个字段）...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		difficulties, err = FetchDifficulties(ctx)
		if err != nil {
			return fmt.Errorf("拉取失败（可用 -skip-fetch 跳过）: %w", err)
		}
		fmt.Printf("   共获取 %d 道题的难度\n", len(difficulties))
	}

	missing := make([]TidanEntry, 0, 16)
	for i := range entries {
		if d := difficulties[entries[i].Slug]; d != "" {
			entries[i].Difficulty = d
		} else {
			entries[i].Difficulty = fallback
			missing = append(missing, entries[i])
		}
	}

	if len(missing) > 0 {
		fmt.Printf("   ⚠️  %d 道题未在 LeetCode 上匹配到，已用 %s 填充：\n", len(missing), fallback)
		for i, e := range missing {
			if i >= 8 {
				fmt.Printf("      ...（其余 %d 道略）\n", len(missing)-8)
				break
			}
			fmt.Printf("      - %d. %s (%s)\n", e.LeetCodeID, e.Title, e.Slug)
		}
	}

	// ---------- 3. 干跑 ----------
	if dryRun {
		fmt.Println("\n🧪 dry-run 模式，不写数据库。各专题题量：")
		printTopicStats(entries)
		return nil
	}

	// ---------- 4. 写库 ----------
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer pool.Close()

	fmt.Println("💾 写入数据库...")
	stats, err := importToDB(ctx, pool, entries)
	if err != nil {
		return err
	}

	fmt.Printf("\n✅ 导入完成\n")
	fmt.Printf("   题目：新增 %d 道，更新 %d 道\n", stats.problemsCreated, stats.problemsUpdated)
	fmt.Printf("   专题标签：新建 %d 个，复用已有 %d 个\n", stats.tagsCreated, stats.tagsReused)
	fmt.Printf("   题目↔专题 关联 %d 条\n", stats.links)
	return nil
}

type importStats struct {
	problemsCreated int
	problemsUpdated int
	tagsCreated     int
	tagsReused      int
	links           int
}

func importToDB(ctx context.Context, pool *pgxpool.Pool, entries []TidanEntry) (*importStats, error) {
	stats := &importStats{}

	// 整个导入放在一个事务里：中途失败不会留下半截数据
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		tagRepo := repository.NewTagRepository(tx)

		topicIDs, tagStats, err := ensureTopicTags(ctx, tagRepo, entries)
		if err != nil {
			return err
		}
		stats.tagsCreated = tagStats.created
		stats.tagsReused = tagStats.reused

		problemIDs, created, updated, err := upsertProblems(ctx, tx, entries)
		if err != nil {
			return err
		}
		stats.problemsCreated = created
		stats.problemsUpdated = updated

		links, err := relinkProblemTopics(ctx, tx, entries, problemIDs, topicIDs)
		if err != nil {
			return err
		}
		stats.links = links

		return nil
	})

	return stats, err
}

type tagStats struct{ created, reused int }

// ensureTopicTags 为每个专题准备一个标签。
//
// 优先【按名字复用】已有标签：题单里的「滑动窗口」「二分查找」等，
// 用户很可能已经手工建过同名标签，重复建会出现两个「滑动窗口」。
func ensureTopicTags(ctx context.Context, tagRepo repository.TagRepository, entries []TidanEntry) (map[string]int64, tagStats, error) {
	var stats tagStats

	existing, err := tagRepo.List(ctx, "")
	if err != nil {
		return nil, stats, err
	}

	byName := make(map[string]int64, len(existing))
	for _, t := range existing {
		byName[t.Name] = t.ID
	}

	topicIDs := make(map[string]int64, 32)
	for i, topic := range Topics(entries) {
		if id, ok := byName[topic]; ok {
			topicIDs[topic] = id
			stats.reused++
			continue
		}

		created, err := tagRepo.FindOrCreate(ctx, &model.Tag{
			Name: topic,
			// slug 用序号保证唯一且 ASCII 安全；展示一律用中文 name
			Slug: fmt.Sprintf("tidan-%02d", i+1),
			Kind: model.TagKindTopic,
		})
		if err != nil {
			return nil, stats, fmt.Errorf("创建专题标签 %q 失败: %w", topic, err)
		}
		topicIDs[topic] = created.ID
		stats.created++
	}

	return topicIDs, stats, nil
}

// upsertProblems 批量写入题目，返回 slug -> 题目 ID。
//
// 用一条 SQL 配合 unnest 完成几百条记录的插入，
// 比循环执行 N 次 INSERT 快两个数量级。
func upsertProblems(ctx context.Context, tx pgx.Tx, entries []TidanEntry) (map[string]int64, int, int, error) {
	lcIDs := make([]int, 0, len(entries))
	titles := make([]string, 0, len(entries))
	slugs := make([]string, 0, len(entries))
	diffs := make([]string, 0, len(entries))

	for _, e := range entries {
		lcIDs = append(lcIDs, e.LeetCodeID)
		titles = append(titles, e.Title)
		slugs = append(slugs, e.Slug)
		diffs = append(diffs, e.Difficulty)
	}

	// 先统计已有多少道，用于区分「新增」与「更新」
	var existed int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM problems WHERE title_slug = ANY($1)`, slugs).Scan(&existed); err != nil {
		return nil, 0, 0, fmt.Errorf("统计已有题目失败: %w", err)
	}

	const q = `
		INSERT INTO problems (leetcode_id, title, title_slug, difficulty)
		SELECT unnest($1::int[]), unnest($2::text[]), unnest($3::text[]), unnest($4::text[])
		ON CONFLICT (title_slug) DO UPDATE
		SET leetcode_id = EXCLUDED.leetcode_id,
		    title      = EXCLUDED.title,
		    difficulty = EXCLUDED.difficulty
		RETURNING id, title_slug`

	rows, err := tx.Query(ctx, q, lcIDs, titles, slugs, diffs)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("写入题目失败: %w", err)
	}
	defer rows.Close()

	ids := make(map[string]int64, len(entries))
	for rows.Next() {
		var id int64
		var slug string
		if err := rows.Scan(&id, &slug); err != nil {
			return nil, 0, 0, fmt.Errorf("读取题目 ID 失败: %w", err)
		}
		ids[slug] = id
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, err
	}

	created := len(entries) - existed
	if created < 0 {
		created = 0
	}
	return ids, created, existed, nil
}

// relinkProblemTopics 重建题目与专题的关联（先删后插，保证幂等）。
func relinkProblemTopics(
	ctx context.Context,
	tx pgx.Tx,
	entries []TidanEntry,
	problemIDs, topicIDs map[string]int64,
) (int, error) {
	allProblemIDs := make([]int64, 0, len(problemIDs))
	for _, id := range problemIDs {
		allProblemIDs = append(allProblemIDs, id)
	}
	if len(allProblemIDs) == 0 {
		return 0, nil
	}

	// 只清理本次涉及的题目，避免影响用户手工维护的其他关联
	if _, err := tx.Exec(ctx,
		`DELETE FROM problem_tags WHERE problem_id = ANY($1)`, allProblemIDs); err != nil {
		return 0, fmt.Errorf("清理旧关联失败: %w", err)
	}

	pids := make([]int64, 0, len(entries))
	tids := make([]int64, 0, len(entries))
	for _, e := range entries {
		pid, okP := problemIDs[e.Slug]
		tid, okT := topicIDs[e.Topic]
		if !okP || !okT {
			continue
		}
		pids = append(pids, pid)
		tids = append(tids, tid)
	}
	if len(pids) == 0 {
		return 0, nil
	}

	const q = `
		INSERT INTO problem_tags (problem_id, tag_id)
		SELECT unnest($1::bigint[]), unnest($2::bigint[])
		ON CONFLICT DO NOTHING`

	tag, err := tx.Exec(ctx, q, pids, tids)
	if err != nil {
		return 0, fmt.Errorf("建立题目标签关联失败: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func printTopicStats(entries []TidanEntry) {
	counts := make(map[string]int, 32)
	for _, e := range entries {
		counts[e.Topic]++
	}

	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		label := name
		if label == "" {
			label = "(未分类)"
		}
		fmt.Printf("   %-26s %d 道\n", label, counts[name])
	}
}
