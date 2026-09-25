package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
)

// OAuthRepository 管理第三方账号与本地用户的关联关系。
type OAuthRepository interface {
	Create(ctx context.Context, a *model.OAuthAccount) error
	// GetByProviderUID 按「provider + provider 侧用户 ID」查找关联记录
	GetByProviderUID(ctx context.Context, provider, providerUID string) (*model.OAuthAccount, error)
	ListByUserID(ctx context.Context, userID int64) ([]*model.OAuthAccount, error)
	Delete(ctx context.Context, userID int64, provider string) error
}

type oauthRepo struct {
	db Querier
}

func NewOAuthRepository(db Querier) OAuthRepository {
	return &oauthRepo{db: db}
}

const oauthColumns = `id, user_id, provider, provider_uid, provider_login, avatar_url, created_at, updated_at`

func scanOAuthAccount(row pgx.Row) (*model.OAuthAccount, error) {
	var a model.OAuthAccount
	err := row.Scan(
		&a.ID, &a.UserID, &a.Provider, &a.ProviderUID,
		&a.ProviderLogin, &a.AvatarURL, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *oauthRepo) Create(ctx context.Context, a *model.OAuthAccount) error {
	const q = `
		INSERT INTO oauth_accounts (user_id, provider, provider_uid, provider_login, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q,
		a.UserID, a.Provider, a.ProviderUID, a.ProviderLogin, a.AvatarURL,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return translateOAuthError(err)
	}
	return nil
}

func (r *oauthRepo) GetByProviderUID(ctx context.Context, provider, providerUID string) (*model.OAuthAccount, error) {
	const q = `SELECT ` + oauthColumns + `
		FROM oauth_accounts WHERE provider = $1 AND provider_uid = $2`

	a, err := scanOAuthAccount(r.db.QueryRow(ctx, q, provider, providerUID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("未找到关联的第三方账号").Wrap(err)
		}
		return nil, fmt.Errorf("查询第三方账号失败: %w", err)
	}
	return a, nil
}

func (r *oauthRepo) ListByUserID(ctx context.Context, userID int64) ([]*model.OAuthAccount, error) {
	const q = `SELECT ` + oauthColumns + `
		FROM oauth_accounts WHERE user_id = $1 ORDER BY created_at`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("查询第三方账号列表失败: %w", err)
	}
	defer rows.Close()

	accounts := make([]*model.OAuthAccount, 0, 2)
	for rows.Next() {
		a, err := scanOAuthAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描第三方账号失败: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (r *oauthRepo) Delete(ctx context.Context, userID int64, provider string) error {
	const q = `DELETE FROM oauth_accounts WHERE user_id = $1 AND provider = $2`

	tag, err := r.db.Exec(ctx, q, userID, provider)
	if err != nil {
		return fmt.Errorf("解绑第三方账号失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound.WithMessage("该账号未绑定此第三方登录")
	}
	return nil
}

func translateOAuthError(err error) error {
	if isUniqueViolation(err, "uq_oauth_provider_uid") {
		return errs.ErrConflict.WithMessage("该 GitHub 账号已被其他用户绑定").Wrap(err)
	}
	return fmt.Errorf("数据库操作失败: %w", err)
}
