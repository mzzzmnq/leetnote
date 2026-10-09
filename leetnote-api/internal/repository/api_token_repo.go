package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/apitoken"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
)

// APITokenRepository 管理长期令牌（PAT）。
type APITokenRepository interface {
	Create(ctx context.Context, t *model.APIToken) error
	// ResolveUserID 用【明文】令牌换 user_id（内部做哈希），并顺带刷新 last_used_at。
	// 令牌无效 / 已被吊销时返回 401。
	ResolveUserID(ctx context.Context, raw string) (int64, error)
	ListByUser(ctx context.Context, userID int64) ([]*model.APIToken, error)
	Revoke(ctx context.Context, userID, id int64) error
}

type apiTokenRepo struct {
	db Querier
}

func NewAPITokenRepository(db Querier) APITokenRepository {
	return &apiTokenRepo{db: db}
}

func (r *apiTokenRepo) Create(ctx context.Context, t *model.APIToken) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO api_tokens (user_id, name, token_hash) VALUES ($1, $2, $3)`,
		t.UserID, t.Name, t.TokenHash)
	return err
}

// ResolveUserID 用一条 UPDATE ... RETURNING 同时完成「校验 + 记录使用时间」。
//
// 【为什么不用先 SELECT 再 UPDATE】那是两次往返，高并发下还有竞态；
// 一条语句既能原子命中唯一索引，也能顺手刷新 last_used_at。
func (r *apiTokenRepo) ResolveUserID(ctx context.Context, raw string) (int64, error) {
	var (
		id     int64
		userID int64
	)
	err := r.db.QueryRow(ctx,
		`UPDATE api_tokens SET last_used_at = now()
		  WHERE token_hash = $1
		  RETURNING id, user_id`, apitoken.Hash(raw)).Scan(&id, &userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errs.ErrUnauthorized.WithMessage("令牌无效或已被吊销")
		}
		return 0, errs.ErrInternal.Wrap(err)
	}
	return userID, nil
}

func (r *apiTokenRepo) ListByUser(ctx context.Context, userID int64) ([]*model.APIToken, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, token_hash, last_used_at, created_at
		   FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	defer rows.Close()

	out := make([]*model.APIToken, 0, 4)
	for rows.Next() {
		var t model.APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.LastUsedAt, &t.CreatedAt); err != nil {
			return nil, errs.ErrInternal.Wrap(err)
		}
		out = append(out, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return out, nil
}

// Revoke 删除令牌。带上 user_id 条件，避免越权删别人的令牌。
func (r *apiTokenRepo) Revoke(ctx context.Context, userID, id int64) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return errs.ErrInternal.Wrap(err)
	}
	if tag.RowsAffected() == 0 {
		return errs.NotFound("TOKEN", "令牌")
	}
	return nil
}
