package repository

import (
	"context"
	"fmt"

	"github.com/mzzzmnq/leetnote-api/internal/model"
)

type StatsRepository interface {
	Overview(ctx context.Context, userID int64) (*model.StatsOverview, error)
	// Trend 返回最近 days 天的新增笔记数，包含没有记录的日期（补 0）
	Trend(ctx context.Context, userID int64, days int) ([]*model.TrendPoint, error)
}

type statsRepo struct {
	db Querier
}

func NewStatsRepository(db Querier) StatsRepository {
	return &statsRepo{db: db}
}

func (r *statsRepo) Overview(ctx context.Context, userID int64) (*model.StatsOverview, error) {
	out := &model.StatsOverview{}

	// ---------- 1. 基础计数 ----------
	// FILTER (WHERE ...) 是 PostgreSQL 的聚合过滤语法，
	// 比 count(CASE WHEN ... THEN 1 END) 更直观，也能走索引。
	const baseQ = `
		SELECT
			count(*),
			count(*) FILTER (WHERE n.is_starred),
			count(*) FILTER (WHERE n.status = 'draft'),
			count(DISTINCT n.problem_id)
		FROM notes n
		WHERE n.user_id = $1`

	if err := r.db.QueryRow(ctx, baseQ, userID).Scan(
		&out.TotalNotes, &out.Starred, &out.Drafts, &out.TotalProblems,
	); err != nil {
		return nil, fmt.Errorf("统计笔记概览失败: %w", err)
	}

	// ---------- 2. 按难度分布 ----------
	const diffQ = `
		SELECT p.difficulty, count(*)
		FROM notes n
		JOIN problems p ON p.id = n.problem_id
		WHERE n.user_id = $1
		GROUP BY p.difficulty`

	rows, err := r.db.Query(ctx, diffQ, userID)
	if err != nil {
		return nil, fmt.Errorf("统计难度分布失败: %w", err)
	}
	for rows.Next() {
		var difficulty string
		var count int64
		if err := rows.Scan(&difficulty, &count); err != nil {
			rows.Close()
			return nil, fmt.Errorf("扫描难度分布失败: %w", err)
		}
		switch difficulty {
		case model.DifficultyEasy:
			out.Easy = count
		case model.DifficultyMedium:
			out.Medium = count
		case model.DifficultyHard:
			out.Hard = count
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取难度分布失败: %w", err)
	}

	// ---------- 3. 解法数与标签数 ----------
	const miscQ = `
		SELECT
			(SELECT count(*) FROM solutions s
			   JOIN notes n ON n.id = s.note_id WHERE n.user_id = $1),
			(SELECT count(DISTINCT nt.tag_id) FROM note_tags nt
			   JOIN notes n ON n.id = nt.note_id WHERE n.user_id = $1)`

	if err := r.db.QueryRow(ctx, miscQ, userID).Scan(&out.TotalSolutions, &out.TotalTags); err != nil {
		return nil, fmt.Errorf("统计解法与标签数失败: %w", err)
	}

	// ---------- 4. 连续打卡（gaps and islands）----------
	if err := r.db.QueryRow(ctx, streakQuery, userID).Scan(
		&out.CurrentStreak, &out.LongestStreak, &out.ActiveDays,
	); err != nil {
		return nil, fmt.Errorf("统计连续打卡失败: %w", err)
	}

	return out, nil
}

// streakQuery 用「gaps and islands」技巧计算连续天数。
//
// 核心思路：把日期排序后减去行号。
// 如果日期是连续的（如 1/1, 1/2, 1/3），减去行号 1,2,3 后都得到 12/31，
// 于是相同的差值就成了「同一段连续区间」的分组标识。
//
// 这是 SQL 面试的经典题，比在应用层循环判断要高效得多。
const streakQuery = `
	WITH days AS (
		SELECT DISTINCT created_at::date AS day
		FROM notes
		WHERE user_id = $1
	),
	islands AS (
		SELECT day, day - (row_number() OVER (ORDER BY day))::int AS grp
		FROM days
	),
	runs AS (
		SELECT grp, count(*) AS len, max(day) AS last_day
		FROM islands
		GROUP BY grp
	)
	SELECT
		-- 当前连续：最后一段必须延伸到今天或昨天，否则视为已断
		COALESCE((
			SELECT len FROM runs
			WHERE last_day >= current_date - 1
			ORDER BY last_day DESC LIMIT 1
		), 0),
		COALESCE((SELECT max(len) FROM runs), 0),
		(SELECT count(*) FROM days)`

func (r *statsRepo) Trend(ctx context.Context, userID int64, days int) ([]*model.TrendPoint, error) {
	if days <= 0 || days > 365 {
		days = 30
	}

	// generate_series 先生成一串完整日期，再 LEFT JOIN 笔记。
	// 这样没有记录的日期也会出现在结果里（count = 0），
	// 前端画折线图时不会出现断点。
	const q = `
		WITH date_series AS (
			SELECT generate_series(
				current_date - ($2::int - 1), current_date, '1 day'
			)::date AS day
		)
		SELECT ds.day, count(n.id)
		FROM date_series ds
		LEFT JOIN notes n
			ON n.user_id = $1 AND n.created_at::date = ds.day
		GROUP BY ds.day
		ORDER BY ds.day`

	rows, err := r.db.Query(ctx, q, userID, days)
	if err != nil {
		return nil, fmt.Errorf("统计趋势失败: %w", err)
	}
	defer rows.Close()

	out := make([]*model.TrendPoint, 0, days)
	for rows.Next() {
		var p model.TrendPoint
		if err := rows.Scan(&p.Day, &p.Count); err != nil {
			return nil, fmt.Errorf("扫描趋势数据失败: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}
