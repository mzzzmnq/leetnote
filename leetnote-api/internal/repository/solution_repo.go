package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
)

type SolutionRepository interface {
	Create(ctx context.Context, s *model.Solution) error
	ListByNoteID(ctx context.Context, noteID int64) ([]*model.Solution, error)

	// GetOwnedByID 取出解法并【同时校验归属】——
	// 通过 JOIN notes 确认它所属的笔记属于该用户，一条 SQL 完成鉴权。
	GetOwnedByID(ctx context.Context, userID, solutionID int64) (*model.Solution, error)

	Update(ctx context.Context, userID int64, s *model.Solution) (*model.Solution, error)
	DeleteOwned(ctx context.Context, userID, solutionID int64) error

	// ReplaceForNote 全量替换某篇笔记的解法（先删后插），调用方需在事务内执行
	ReplaceForNote(ctx context.Context, noteID int64, solutions []*model.Solution) error
}

type solutionRepo struct {
	db Querier
}

func NewSolutionRepository(db Querier) SolutionRepository {
	return &solutionRepo{db: db}
}

const solutionColumns = `
	s.id, s.note_id, s.title, s.language, s.code,
	s.time_complexity, s.space_complexity, s.sort_order, s.created_at`

func scanSolution(row pgx.Row) (*model.Solution, error) {
	var s model.Solution
	err := row.Scan(
		&s.ID, &s.NoteID, &s.Title, &s.Language, &s.Code,
		&s.TimeComplexity, &s.SpaceComplexity, &s.SortOrder, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *solutionRepo) Create(ctx context.Context, s *model.Solution) error {
	const q = `
		INSERT INTO solutions (note_id, title, language, code,
		                       time_complexity, space_complexity, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, q,
		s.NoteID, s.Title, s.Language, s.Code,
		s.TimeComplexity, s.SpaceComplexity, s.SortOrder,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		if isForeignKeyViolation(err) {
			return errs.ErrBadRequest.WithMessage("关联的笔记不存在").Wrap(err)
		}
		return fmt.Errorf("创建解法失败: %w", err)
	}
	return nil
}

func (r *solutionRepo) ListByNoteID(ctx context.Context, noteID int64) ([]*model.Solution, error) {
	const q = `SELECT ` + solutionColumns + `
		FROM solutions s WHERE s.note_id = $1 ORDER BY s.sort_order, s.id`

	rows, err := r.db.Query(ctx, q, noteID)
	if err != nil {
		return nil, fmt.Errorf("查询解法列表失败: %w", err)
	}
	defer rows.Close()

	out := make([]*model.Solution, 0, 4)
	for rows.Next() {
		s, err := scanSolution(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描解法失败: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *solutionRepo) GetOwnedByID(ctx context.Context, userID, solutionID int64) (*model.Solution, error) {
	const q = `SELECT ` + solutionColumns + `
		FROM solutions s
		JOIN notes n ON n.id = s.note_id
		WHERE s.id = $1 AND n.user_id = $2`

	s, err := scanSolution(r.db.QueryRow(ctx, q, solutionID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("解法不存在").Wrap(err)
		}
		return nil, fmt.Errorf("查询解法失败: %w", err)
	}
	return s, nil
}

func (r *solutionRepo) Update(ctx context.Context, userID int64, s *model.Solution) (*model.Solution, error) {
	// 用子查询在 UPDATE 内部校验归属，避免「先查后改」之间的竞态
	const q = `
		UPDATE solutions AS s
		SET title = $3, language = $4, code = $5,
		    time_complexity = $6, space_complexity = $7, sort_order = $8
		WHERE s.id = $1
		  AND s.note_id IN (SELECT id FROM notes WHERE user_id = $2)
		RETURNING ` + solutionColumns

	updated, err := scanSolution(r.db.QueryRow(ctx, q,
		s.ID, userID, s.Title, s.Language, s.Code,
		s.TimeComplexity, s.SpaceComplexity, s.SortOrder))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("解法不存在").Wrap(err)
		}
		return nil, fmt.Errorf("更新解法失败: %w", err)
	}
	return updated, nil
}

func (r *solutionRepo) DeleteOwned(ctx context.Context, userID, solutionID int64) error {
	const q = `
		DELETE FROM solutions
		WHERE id = $1
		  AND note_id IN (SELECT id FROM notes WHERE user_id = $2)`

	tag, err := r.db.Exec(ctx, q, solutionID, userID)
	if err != nil {
		return fmt.Errorf("删除解法失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound.WithMessage("解法不存在")
	}
	return nil
}

// ReplaceForNote 全量替换笔记的解法。
//
// 一篇笔记通常只有 1-5 个解法，所以在事务内循环 INSERT 完全够用，
// 不值得为了批量插入引入 unnest/COPY 的复杂度。
func (r *solutionRepo) ReplaceForNote(ctx context.Context, noteID int64, solutions []*model.Solution) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM solutions WHERE note_id = $1`, noteID); err != nil {
		return fmt.Errorf("清除原有解法失败: %w", err)
	}

	for i, s := range solutions {
		s.NoteID = noteID
		s.SortOrder = i
		if err := r.Create(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
