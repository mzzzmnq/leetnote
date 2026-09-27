package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
)

type ReviewRepository interface {
	// CreateForNote 为笔记建一张复习卡（幂等：已存在则什么都不做）
	CreateForNote(ctx context.Context, noteID int64) error
	// GetDue 取到期待复习的卡片
	GetDue(ctx context.Context, userID int64, limit int) ([]*model.ReviewCard, error)
	// GetOwnedCard 取卡片并校验归属（通过 JOIN notes 一条 SQL 完成鉴权）
	GetOwnedCard(ctx context.Context, userID, cardID int64) (*model.ReviewCard, error)
	// UpdateCard 写回 SM-2 算出的新状态
	UpdateCard(ctx context.Context, card *model.ReviewCard) error
	// InsertLog 追加一条复习流水
	InsertLog(ctx context.Context, log *model.ReviewLog) error
	Stats(ctx context.Context, userID int64) (*model.ReviewStats, error)
}

type reviewRepo struct {
	db Querier
}

func NewReviewRepository(db Querier) ReviewRepository {
	return &reviewRepo{db: db}
}

// 卡片查询统一带上笔记标题 —— 复习界面要展示「在复习哪道题」
//
// 注意：review_cards 表【没有】created_at / updated_at
// （见 migrations/000001_init.up.sql）。这里不要顺手加，
// 否则会得到 "column c.created_at does not exist"。
const reviewCardColumns = `
	c.id, c.note_id, n.title, n.summary,
	c.ease_factor, c.interval_days, c.repetitions,
	c.due_at, c.last_reviewed_at`

func scanReviewCard(row pgx.Row) (*model.ReviewCard, error) {
	var card model.ReviewCard
	err := row.Scan(
		&card.ID, &card.NoteID, &card.NoteTitle, &card.NoteSummary,
		&card.EaseFactor, &card.IntervalDays, &card.Repetitions,
		&card.DueAt, &card.LastReviewedAt,
	)
	if err != nil {
		return nil, err
	}
	return &card, nil
}

func (r *reviewRepo) CreateForNote(ctx context.Context, noteID int64) error {
	// ON CONFLICT DO NOTHING 让这个操作幂等：
	// 编辑笔记时也会调用，不能因为卡片已存在就报错。
	const q = `
		INSERT INTO review_cards (note_id)
		VALUES ($1)
		ON CONFLICT (note_id) DO NOTHING`

	if _, err := r.db.Exec(ctx, q, noteID); err != nil {
		if isForeignKeyViolation(err) {
			return errs.ErrBadRequest.WithMessage("关联的笔记不存在").Wrap(err)
		}
		return fmt.Errorf("创建复习卡失败: %w", err)
	}
	return nil
}

func (r *reviewRepo) GetDue(ctx context.Context, userID int64, limit int) ([]*model.ReviewCard, error) {
	const q = `
		SELECT ` + reviewCardColumns + `
		FROM review_cards c
		JOIN notes n ON n.id = c.note_id
		WHERE n.user_id = $1 AND c.due_at <= now()
		ORDER BY c.due_at
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("查询待复习卡片失败: %w", err)
	}
	defer rows.Close()

	out := make([]*model.ReviewCard, 0, limit)
	for rows.Next() {
		card, err := scanReviewCard(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描复习卡失败: %w", err)
		}
		out = append(out, card)
	}
	return out, rows.Err()
}

func (r *reviewRepo) GetOwnedCard(ctx context.Context, userID, cardID int64) (*model.ReviewCard, error) {
	const q = `
		SELECT ` + reviewCardColumns + `
		FROM review_cards c
		JOIN notes n ON n.id = c.note_id
		WHERE c.id = $1 AND n.user_id = $2`

	card, err := scanReviewCard(r.db.QueryRow(ctx, q, cardID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("复习卡不存在").Wrap(err)
		}
		return nil, fmt.Errorf("查询复习卡失败: %w", err)
	}
	return card, nil
}

func (r *reviewRepo) UpdateCard(ctx context.Context, card *model.ReviewCard) error {
	const q = `
		UPDATE review_cards
		SET ease_factor      = $2,
		    interval_days    = $3,
		    repetitions      = $4,
		    due_at           = $5,
		    last_reviewed_at = $6
		WHERE id = $1`

	tag, err := r.db.Exec(ctx, q,
		card.ID, card.EaseFactor, card.IntervalDays,
		card.Repetitions, card.DueAt, card.LastReviewedAt)

	if err != nil {
		return fmt.Errorf("更新复习卡失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound.WithMessage("复习卡不存在")
	}
	return nil
}

func (r *reviewRepo) InsertLog(ctx context.Context, log *model.ReviewLog) error {
	const q = `
		INSERT INTO review_logs (card_id, rating, prev_interval, next_interval)
		VALUES ($1, $2, $3, $4)
		RETURNING id, reviewed_at`

	err := r.db.QueryRow(ctx, q,
		log.CardID, log.Rating, log.PrevInterval, log.NextInterval,
	).Scan(&log.ID, &log.ReviewedAt)

	if err != nil {
		return fmt.Errorf("写入复习流水失败: %w", err)
	}
	return nil
}

func (r *reviewRepo) Stats(ctx context.Context, userID int64) (*model.ReviewStats, error) {
	out := &model.ReviewStats{}

	// ---------- 基础计数 ----------
	const baseQ = `
		SELECT
			count(*) FILTER (WHERE c.due_at <= now()),
			count(*),
			count(*) FILTER (WHERE c.last_reviewed_at IS NOT NULL),
			(SELECT count(*) FROM review_logs l
			   JOIN review_cards rc ON rc.id = l.card_id
			   JOIN notes n2 ON n2.id = rc.note_id
			 WHERE n2.user_id = $1)
		FROM review_cards c
		JOIN notes n ON n.id = c.note_id
		WHERE n.user_id = $1`

	if err := r.db.QueryRow(ctx, baseQ, userID).Scan(
		&out.DueCount, &out.TotalCards, &out.LearnedCards, &out.TotalReviews,
	); err != nil {
		return nil, fmt.Errorf("统计复习数据失败: %w", err)
	}

	// ---------- 未来 7 天到期待distribution ----------
	//
	// 先把「该用户的卡片」做成子查询再 LEFT JOIN，而不是直接 JOIN notes 加条件：
	// 后者会让左连接退化成内连接，导致没有卡片的日子被整行丢掉，
	// 前端柱状图就会缺柱子。
	const forecastQ = `
		WITH days AS (
			SELECT generate_series(current_date, current_date + 6, '1 day')::date AS day
		),
		user_cards AS (
			SELECT c.id, c.due_at
			FROM review_cards c
			JOIN notes n ON n.id = c.note_id
			WHERE n.user_id = $1
		)
		SELECT d.day, count(uc.id)
		FROM days d
		LEFT JOIN user_cards uc ON uc.due_at::date = d.day
		GROUP BY d.day
		ORDER BY d.day`

	rows, err := r.db.Query(ctx, forecastQ, userID)
	if err != nil {
		return nil, fmt.Errorf("统计复习计划失败: %w", err)
	}
	defer rows.Close()

	out.Forecast = make([]model.ForecastPoint, 0, 7)
	for rows.Next() {
		var p model.ForecastPoint
		if err := rows.Scan(&p.Date, &p.Count); err != nil {
			return nil, fmt.Errorf("扫描复习计划失败: %w", err)
		}
		out.Forecast = append(out.Forecast, p)
	}

	return out, rows.Err()
}
