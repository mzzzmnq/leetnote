package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/mzzzmnq/leetnote-api/internal/model"
)

type SearchRepository interface {
	SearchNotes(ctx context.Context, userID int64, keyword string, limit int) ([]*model.Note, error)
	SearchProblems(ctx context.Context, keyword string, limit int) ([]*model.Problem, error)
}

type searchRepo struct {
	db Querier
}

func NewSearchRepository(db Querier) SearchRepository {
	return &searchRepo{db: db}
}

const defaultSearchLimit = 20

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	if limit > 50 {
		return 50
	}
	return limit
}

// SearchNotes 搜索当前用户的笔记。
//
// 两个要点：
//
//  1. 【过滤】用 ILIKE '%kw%'。pg_trgm 的 GIN 索引（gin_trgm_ops）
//     能加速这种前后都带通配符的匹配——普通 B-tree 索引做不到。
//
//  2. 【排序】不用 similarity() 对正文打分：正文很长时三元组相似度会被
//     稀释到接近 0，排序几乎随机。改成「标题命中优先 → 标题相似度 → 最近更新」，
//     结果更符合直觉。
func (r *searchRepo) SearchNotes(ctx context.Context, userID int64, keyword string, limit int) ([]*model.Note, error) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return []*model.Note{}, nil
	}

	const q = `
		SELECT ` + noteColumns + `
		FROM notes n
		WHERE n.user_id = $1
		  AND (n.title ILIKE $2 OR n.content_md ILIKE $2)
		ORDER BY
			(n.title ILIKE $2) DESC,
			similarity(n.title, $3) DESC,
			n.updated_at DESC
		LIMIT $4`

	rows, err := r.db.Query(ctx, q, userID, likeContains(kw), kw, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("搜索笔记失败: %w", err)
	}
	defer rows.Close()

	out := make([]*model.Note, 0, 8)
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描搜索结果失败: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// SearchProblems 搜索题目（全局共享，不区分用户）。
func (r *searchRepo) SearchProblems(ctx context.Context, keyword string, limit int) ([]*model.Problem, error) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return []*model.Problem{}, nil
	}

	const q = `
		SELECT ` + problemColumns + `
		FROM problems
		WHERE title ILIKE $1 OR title_slug ILIKE $1
		ORDER BY
			(title ILIKE $1) DESC,
			similarity(title, $2) DESC,
			leetcode_id NULLS LAST
		LIMIT $3`

	rows, err := r.db.Query(ctx, q, likeContains(kw), kw, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("搜索题目失败: %w", err)
	}

	return scanProblems(rows)
}
