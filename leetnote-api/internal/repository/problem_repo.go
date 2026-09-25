package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
)

// ProblemFilter 是题目列表的筛选条件。
type ProblemFilter struct {
	Keyword    string // 模糊匹配 title / title_slug
	Difficulty string // Easy / Medium / Hard
	Pagination
}

type ProblemRepository interface {
	Create(ctx context.Context, p *model.Problem) error
	GetByID(ctx context.Context, id int64) (*model.Problem, error)
	List(ctx context.Context, f ProblemFilter) ([]*model.Problem, int64, error)
	Update(ctx context.Context, p *model.Problem) (*model.Problem, error)
	Delete(ctx context.Context, id int64) error
	// GetByIDs 批量取，用于笔记详情一次性带出题目，避免 N+1
	GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.Problem, error)
}

type problemRepo struct {
	db Querier
}

func NewProblemRepository(db Querier) ProblemRepository {
	return &problemRepo{db: db}
}

const problemColumns = `id, leetcode_id, title, title_slug, difficulty, url, created_at`

func scanProblem(row pgx.Row) (*model.Problem, error) {
	var p model.Problem
	err := row.Scan(&p.ID, &p.LeetCodeID, &p.Title, &p.TitleSlug, &p.Difficulty, &p.URL, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanProblems(rows pgx.Rows) ([]*model.Problem, error) {
	defer rows.Close()

	out := make([]*model.Problem, 0, 16)
	for rows.Next() {
		p, err := scanProblem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *problemRepo) Create(ctx context.Context, p *model.Problem) error {
	const q = `
		INSERT INTO problems (leetcode_id, title, title_slug, difficulty, url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, q,
		p.LeetCodeID, p.Title, p.TitleSlug, p.Difficulty, p.URL,
	).Scan(&p.ID, &p.CreatedAt)

	if err != nil {
		if isUniqueViolation(err, "problems_title_slug_key") {
			return errs.ErrConflict.WithMessage("该题目的 slug 已存在").Wrap(err)
		}
		return fmt.Errorf("创建题目失败: %w", err)
	}
	return nil
}

func (r *problemRepo) GetByID(ctx context.Context, id int64) (*model.Problem, error) {
	const q = `SELECT ` + problemColumns + ` FROM problems WHERE id = $1`

	p, err := scanProblem(r.db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("题目不存在").Wrap(err)
		}
		return nil, fmt.Errorf("查询题目失败: %w", err)
	}
	return p, nil
}

func (r *problemRepo) GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.Problem, error) {
	if len(ids) == 0 {
		return map[int64]*model.Problem{}, nil
	}

	const q = `SELECT ` + problemColumns + ` FROM problems WHERE id = ANY($1)`

	rows, err := r.db.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("批量查询题目失败: %w", err)
	}

	list, err := scanProblems(rows)
	if err != nil {
		return nil, fmt.Errorf("扫描题目失败: %w", err)
	}

	out := make(map[int64]*model.Problem, len(list))
	for _, p := range list {
		out[p.ID] = p
	}
	return out, nil
}

// List 支持条件筛选 + 分页。
//
// SQL 是【动态拼接】的，但值一律走占位符 $N，绝不做字符串拼接——
// 这是防 SQL 注入的基本功。拼的只是 WHERE 子句的结构。
func (r *problemRepo) List(ctx context.Context, f ProblemFilter) ([]*model.Problem, int64, error) {
	conditions := []string{"TRUE"}
	args := make([]any, 0, 4)

	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		args = append(args, "%"+kw+"%")
		// 同一个占位符用两次（同一个值），所以 n 只自增一次
		n := len(args)
		conditions = append(conditions,
			fmt.Sprintf("(title ILIKE $%d OR title_slug ILIKE $%d)", n, n))
	}
	if f.Difficulty != "" {
		args = append(args, f.Difficulty)
		conditions = append(conditions, fmt.Sprintf("difficulty = $%d", len(args)))
	}

	where := strings.Join(conditions, " AND ")

	var total int64
	countSQL := "SELECT count(*) FROM problems WHERE " + where
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计题目数量失败: %w", err)
	}

	args = append(args, f.Limit(), f.Offset())
	listSQL := fmt.Sprintf(
		`SELECT %s FROM problems WHERE %s
		 ORDER BY leetcode_id NULLS LAST, id
		 LIMIT $%d OFFSET $%d`,
		problemColumns, where, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询题目列表失败: %w", err)
	}

	list, err := scanProblems(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("扫描题目列表失败: %w", err)
	}
	return list, total, nil
}

func (r *problemRepo) Update(ctx context.Context, p *model.Problem) (*model.Problem, error) {
	const q = `
		UPDATE problems
		SET leetcode_id = $2, title = $3, title_slug = $4, difficulty = $5, url = $6
		WHERE id = $1
		RETURNING ` + problemColumns

	updated, err := scanProblem(r.db.QueryRow(ctx, q,
		p.ID, p.LeetCodeID, p.Title, p.TitleSlug, p.Difficulty, p.URL))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("题目不存在").Wrap(err)
		}
		if isUniqueViolation(err, "problems_title_slug_key") {
			return nil, errs.ErrConflict.WithMessage("该题目的 slug 已存在").Wrap(err)
		}
		return nil, fmt.Errorf("更新题目失败: %w", err)
	}
	return updated, nil
}

func (r *problemRepo) Delete(ctx context.Context, id int64) error {
	// problems 被 notes.problem_id 以 ON DELETE SET NULL 引用：
	// 删题目不会连带删笔记，笔记本身有价值。
	tag, err := r.db.Exec(ctx, `DELETE FROM problems WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除题目失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound.WithMessage("题目不存在")
	}
	return nil
}
