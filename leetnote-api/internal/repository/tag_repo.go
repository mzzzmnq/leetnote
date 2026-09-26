package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
)

type TagRepository interface {
	Create(ctx context.Context, t *model.Tag) error
	GetByID(ctx context.Context, id int64) (*model.Tag, error)
	List(ctx context.Context, kind string) ([]*model.Tag, error)
	Delete(ctx context.Context, id int64) error

	// SetNoteTags 全量替换某篇笔记的标签关联（先删后插）
	SetNoteTags(ctx context.Context, noteID int64, tagIDs []int64) error
	// ListByNoteIDs 批量取多篇笔记的标签，用于列表页避免 N+1
	ListByNoteIDs(ctx context.Context, noteIDs []int64) (map[int64][]*model.Tag, error)

	// SetProblemTags 全量替换某道题的标签关联
	SetProblemTags(ctx context.Context, problemID int64, tagIDs []int64) error
	// ListByProblemIDs 批量取多道题的标签
	ListByProblemIDs(ctx context.Context, problemIDs []int64) (map[int64][]*model.Tag, error)

	// FindOrCreate 按 slug 查找标签，不存在则创建（导入时用，保证可重复执行）
	FindOrCreate(ctx context.Context, t *model.Tag) (*model.Tag, error)

	// CountExisting 返回给定 ID 中真实存在的个数，用于校验用户传的 tag_ids
	CountExisting(ctx context.Context, ids []int64) (int64, error)
}

type tagRepo struct {
	db Querier
}

func NewTagRepository(db Querier) TagRepository {
	return &tagRepo{db: db}
}

const tagColumns = `id, name, slug, kind, created_at`

func scanTag(row pgx.Row) (*model.Tag, error) {
	var t model.Tag
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Kind, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *tagRepo) Create(ctx context.Context, t *model.Tag) error {
	const q = `
		INSERT INTO tags (name, slug, kind)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, q, t.Name, t.Slug, t.Kind).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		switch {
		case isUniqueViolation(err, "tags_name_key"):
			return errs.ErrConflict.WithMessage("该标签名已存在").Wrap(err)
		case isUniqueViolation(err, "tags_slug_key"):
			return errs.ErrConflict.WithMessage("该标签 slug 已存在").Wrap(err)
		}
		return fmt.Errorf("创建标签失败: %w", err)
	}
	return nil
}

func (r *tagRepo) GetByID(ctx context.Context, id int64) (*model.Tag, error) {
	const q = `SELECT ` + tagColumns + ` FROM tags WHERE id = $1`

	t, err := scanTag(r.db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("标签不存在").Wrap(err)
		}
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	return t, nil
}

func (r *tagRepo) List(ctx context.Context, kind string) ([]*model.Tag, error) {
	conditions := "TRUE"
	args := []any{}

	if kind != "" {
		args = append(args, kind)
		conditions = "kind = $1"
	}

	q := `SELECT ` + tagColumns + ` FROM tags WHERE ` + conditions + ` ORDER BY kind, id`

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	defer rows.Close()

	out := make([]*model.Tag, 0, 32)
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描标签失败: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *tagRepo) Delete(ctx context.Context, id int64) error {
	// note_tags 上是 ON DELETE CASCADE，删标签会自动清理关联，不动笔记本身
	tag, err := r.db.Exec(ctx, `DELETE FROM tags WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除标签失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound.WithMessage("标签不存在")
	}
	return nil
}

// SetNoteTags 全量替换笔记的标签。
//
// 「先删后插」而不是「算出差异再增删」：逻辑简单、不易出错，
// 标签数量很小（十几条），性能差异可以忽略。
func (r *tagRepo) SetNoteTags(ctx context.Context, noteID int64, tagIDs []int64) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM note_tags WHERE note_id = $1`, noteID); err != nil {
		return fmt.Errorf("清除笔记标签失败: %w", err)
	}
	if len(tagIDs) == 0 {
		return nil
	}

	// 用 unnest 一条 SQL 批量插入，避免循环执行 N 次 INSERT
	const q = `
		INSERT INTO note_tags (note_id, tag_id)
		SELECT $1, unnest($2::bigint[])
		ON CONFLICT DO NOTHING`

	if _, err := r.db.Exec(ctx, q, noteID, tagIDs); err != nil {
		return fmt.Errorf("写入笔记标签失败: %w", err)
	}
	return nil
}

func (r *tagRepo) ListByNoteIDs(ctx context.Context, noteIDs []int64) (map[int64][]*model.Tag, error) {
	if len(noteIDs) == 0 {
		return map[int64][]*model.Tag{}, nil
	}

	const q = `
		SELECT nt.note_id, t.id, t.name, t.slug, t.kind, t.created_at
		FROM note_tags nt
		JOIN tags t ON t.id = nt.tag_id
		WHERE nt.note_id = ANY($1)
		ORDER BY nt.note_id, t.id`

	rows, err := r.db.Query(ctx, q, noteIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询笔记标签失败: %w", err)
	}
	defer rows.Close()

	out := make(map[int64][]*model.Tag, len(noteIDs))
	for rows.Next() {
		var noteID int64
		var t model.Tag
		if err := rows.Scan(&noteID, &t.ID, &t.Name, &t.Slug, &t.Kind, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描笔记标签失败: %w", err)
		}
		out[noteID] = append(out[noteID], &t)
	}
	return out, rows.Err()
}

func (r *tagRepo) CountExisting(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	var n int64
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM tags WHERE id = ANY($1)`, ids).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("校验标签失败: %w", err)
	}
	return n, nil
}

// FindOrCreate 按 slug 查找标签，不存在则创建。
//
// 用 ON CONFLICT ... DO UPDATE 而不是 DO NOTHING：
// DO NOTHING 在冲突时不返回任何行，拿不到已存在记录的 id，
// 就得再查一次。DO UPDATE 把 slug 设成它原本的值（无副作用的空操作），
// 这样无论插入还是冲突都能 RETURNING 回完整的行，一次往返搞定。
//
// 这让导入脚本可以【重复执行】而不产生重复标签。
func (r *tagRepo) FindOrCreate(ctx context.Context, t *model.Tag) (*model.Tag, error) {
	const q = `
		INSERT INTO tags (name, slug, kind)
		VALUES ($1, $2, $3)
		ON CONFLICT (slug) DO UPDATE SET slug = EXCLUDED.slug
		RETURNING ` + tagColumns

	out, err := scanTag(r.db.QueryRow(ctx, q, t.Name, t.Slug, t.Kind))
	if err != nil {
		return nil, fmt.Errorf("创建或查询标签失败: %w", err)
	}
	return out, nil
}

// SetProblemTags 全量替换题目的标签，逻辑与 SetNoteTags 一致。
func (r *tagRepo) SetProblemTags(ctx context.Context, problemID int64, tagIDs []int64) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM problem_tags WHERE problem_id = $1`, problemID); err != nil {
		return fmt.Errorf("清除题目标签失败: %w", err)
	}
	if len(tagIDs) == 0 {
		return nil
	}

	const q = `
		INSERT INTO problem_tags (problem_id, tag_id)
		SELECT $1, unnest($2::bigint[])
		ON CONFLICT DO NOTHING`

	if _, err := r.db.Exec(ctx, q, problemID, tagIDs); err != nil {
		return fmt.Errorf("写入题目标签失败: %w", err)
	}
	return nil
}

func (r *tagRepo) ListByProblemIDs(ctx context.Context, problemIDs []int64) (map[int64][]*model.Tag, error) {
	if len(problemIDs) == 0 {
		return map[int64][]*model.Tag{}, nil
	}

	const q = `
		SELECT pt.problem_id, t.id, t.name, t.slug, t.kind, t.created_at
		FROM problem_tags pt
		JOIN tags t ON t.id = pt.tag_id
		WHERE pt.problem_id = ANY($1)
		ORDER BY pt.problem_id, t.id`

	rows, err := r.db.Query(ctx, q, problemIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询题目标签失败: %w", err)
	}
	defer rows.Close()

	out := make(map[int64][]*model.Tag, len(problemIDs))
	for rows.Next() {
		var problemID int64
		var t model.Tag
		if err := rows.Scan(&problemID, &t.ID, &t.Name, &t.Slug, &t.Kind, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描题目标签失败: %w", err)
		}
		out[problemID] = append(out[problemID], &t)
	}
	return out, rows.Err()
}
