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

// NoteFilter 是笔记列表的筛选条件。
//
// UserID 是【必填】且不可省略：笔记是用户私有数据，
// 任何查询都必须限定在当前用户名下，否则就是越权漏洞（IDOR）。
type NoteFilter struct {
	UserID     int64
	Keyword    string
	Difficulty string
	TagID      int64
	Status     string
	Starred    *bool
	ProblemID  int64
	Sort       string
	Pagination
}

type NoteRepository interface {
	Create(ctx context.Context, n *model.Note) error

	// GetByID 必须同时传 userID —— 只按 id 查会让 A 用户读到 B 用户的笔记
	GetByID(ctx context.Context, userID, id int64) (*model.Note, error)

	List(ctx context.Context, f NoteFilter) ([]*model.Note, int64, error)
	Update(ctx context.Context, n *model.Note) (*model.Note, error)
	Delete(ctx context.Context, userID, id int64) error
	ToggleStar(ctx context.Context, userID, id int64) (*model.Note, error)
}

type noteRepo struct {
	db Querier
}

func NewNoteRepository(db Querier) NoteRepository {
	return &noteRepo{db: db}
}

// 列表里带上解法数量，避免前端为每条笔记再查一次 solutions（N+1）。
// 这里用相关子查询而不是 LEFT JOIN + GROUP BY：可读性更好，
// 且每页只有 20 行，性能差异可忽略。
const noteColumns = `
	n.id, n.user_id, n.problem_id, n.title, n.content_md, n.summary,
	n.status, n.is_starred, n.view_count, n.created_at, n.updated_at,
	(SELECT count(*) FROM solutions s WHERE s.note_id = n.id) AS solution_count`

func scanNote(row pgx.Row) (*model.Note, error) {
	var n model.Note
	err := row.Scan(
		&n.ID, &n.UserID, &n.ProblemID, &n.Title, &n.ContentMD, &n.Summary,
		&n.Status, &n.IsStarred, &n.ViewCount, &n.CreatedAt, &n.UpdatedAt,
		&n.SolutionCount,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *noteRepo) Create(ctx context.Context, n *model.Note) error {
	const q = `
		INSERT INTO notes (user_id, problem_id, title, content_md, summary, status, is_starred)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, view_count, created_at, updated_at`

	err := r.db.QueryRow(ctx, q,
		n.UserID, n.ProblemID, n.Title, n.ContentMD, n.Summary, n.Status, n.IsStarred,
	).Scan(&n.ID, &n.ViewCount, &n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		// 部分唯一索引 uq_notes_user_problem：同一用户对同一题目只能有一篇笔记
		if isUniqueViolation(err, "uq_notes_user_problem") {
			return errs.ErrConflict.WithMessage("你已经为该题目创建过笔记了").Wrap(err)
		}
		if isForeignKeyViolation(err) {
			return errs.ErrBadRequest.WithMessage("关联的题目不存在").Wrap(err)
		}
		return fmt.Errorf("创建笔记失败: %w", err)
	}
	return nil
}

func (r *noteRepo) GetByID(ctx context.Context, userID, id int64) (*model.Note, error) {
	// WHERE 里同时带上 user_id —— 这是防越权的核心一行。
	// 传别人的 note_id 只会得到 404，而不是别人的数据。
	const q = `SELECT ` + noteColumns + ` FROM notes n WHERE n.id = $1 AND n.user_id = $2`

	n, err := scanNote(r.db.QueryRow(ctx, q, id, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("笔记不存在").Wrap(err)
		}
		return nil, fmt.Errorf("查询笔记失败: %w", err)
	}
	return n, nil
}

func (r *noteRepo) List(ctx context.Context, f NoteFilter) ([]*model.Note, int64, error) {
	conditions := []string{"n.user_id = $1"}
	args := []any{f.UserID}
	joins := ""

	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		args = append(args, likeContains(kw))
		n := len(args)
		conditions = append(conditions,
			fmt.Sprintf("(n.title ILIKE $%d OR n.content_md ILIKE $%d)", n, n))
	}
	if f.Difficulty != "" {
		args = append(args, f.Difficulty)
		conditions = append(conditions, fmt.Sprintf("p.difficulty = $%d", len(args)))
		joins += " LEFT JOIN problems p ON p.id = n.problem_id"
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conditions = append(conditions, fmt.Sprintf("n.status = $%d", len(args)))
	}
	if f.Starred != nil {
		args = append(args, *f.Starred)
		conditions = append(conditions, fmt.Sprintf("n.is_starred = $%d", len(args)))
	}
	if f.ProblemID > 0 {
		args = append(args, f.ProblemID)
		conditions = append(conditions, fmt.Sprintf("n.problem_id = $%d", len(args)))
	}
	if f.TagID > 0 {
		args = append(args, f.TagID)
		// EXISTS 而不是 JOIN：同一篇笔记命中多个标签时不会产生重复行
		n := len(args)
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM note_tags nt WHERE nt.note_id = n.id AND nt.tag_id = $%d)", n))
	}

	where := strings.Join(conditions, " AND ")

	var total int64
	countSQL := "SELECT count(*) FROM notes n " + joins + " WHERE " + where
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计笔记数量失败: %w", err)
	}

	args = append(args, f.Limit(), f.Offset())
	listSQL := fmt.Sprintf(
		`SELECT %s FROM notes n %s WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		noteColumns, joins, where, noteOrderBy(f.Sort), len(args)-1, len(args))

	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询笔记列表失败: %w", err)
	}
	defer rows.Close()

	out := make([]*model.Note, 0, f.Limit())
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描笔记失败: %w", err)
		}
		out = append(out, n)
	}
	return out, total, rows.Err()
}

// noteOrderBy 把排序参数【白名单化】。
//
// 绝不能把用户输入直接拼进 ORDER BY —— 那是 SQL 注入的经典入口。
// 只允许映射到写死的几种子句。
func noteOrderBy(sort string) string {
	switch sort {
	case "updated":
		return "n.updated_at DESC, n.id DESC"
	case "title":
		return "n.title ASC, n.id DESC"
	case "created_asc":
		return "n.created_at ASC, n.id ASC"
	default:
		return "n.created_at DESC, n.id DESC"
	}
}

func (r *noteRepo) Update(ctx context.Context, n *model.Note) (*model.Note, error) {
	const q = `
		UPDATE notes AS n
		SET problem_id = $3, title = $4, content_md = $5,
		    summary = $6, status = $7, is_starred = $8
		WHERE n.id = $1 AND n.user_id = $2
		RETURNING ` + noteColumns

	updated, err := scanNote(r.db.QueryRow(ctx, q,
		n.ID, n.UserID, n.ProblemID, n.Title, n.ContentMD, n.Summary, n.Status, n.IsStarred))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("笔记不存在").Wrap(err)
		}
		if isUniqueViolation(err, "uq_notes_user_problem") {
			return nil, errs.ErrConflict.WithMessage("你已经为该题目创建过笔记了").Wrap(err)
		}
		return nil, fmt.Errorf("更新笔记失败: %w", err)
	}
	return updated, nil
}

func (r *noteRepo) Delete(ctx context.Context, userID, id int64) error {
	// solutions / note_tags 上是 ON DELETE CASCADE，会一并清理
	tag, err := r.db.Exec(ctx, `DELETE FROM notes WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("删除笔记失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound.WithMessage("笔记不存在")
	}
	return nil
}

// ToggleStar 原子地翻转收藏状态。
//
// 用一条 UPDATE ... SET is_starred = NOT is_starred 完成，
// 而不是「先读出来再写回去」——后者在并发下会丢更新。
func (r *noteRepo) ToggleStar(ctx context.Context, userID, id int64) (*model.Note, error) {
	const q = `
		UPDATE notes AS n SET is_starred = NOT n.is_starred
		WHERE n.id = $1 AND n.user_id = $2
		RETURNING ` + noteColumns

	updated, err := scanNote(r.db.QueryRow(ctx, q, id, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("笔记不存在").Wrap(err)
		}
		return nil, fmt.Errorf("切换收藏状态失败: %w", err)
	}
	return updated, nil
}
