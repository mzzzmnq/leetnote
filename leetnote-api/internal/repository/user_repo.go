package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
)

// UserRepository 定义用户数据的存取契约。
//
// 定义成接口而不是直接用结构体，是为了让 service 层可以注入假实现做单元测试
// （不需要真实数据库）。这是 Go 里最常用的解耦手法。
type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id int64) (*model.User, error)
	// GetByLogin 支持用「用户名」或「邮箱」登录
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	// GetByEmail 按邮箱精确查找（OAuth 首次登录时用于关联已有账号）
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateProfile(ctx context.Context, id int64, avatarURL, bio *string) (*model.User, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

// 显式列出字段，绝不用 SELECT *。
// 原因：① 表加字段时会静默改变查询结果；② 避免误把 password_hash 带到不该去的地方。
const userColumns = `id, username, email, password_hash, avatar_url, bio, created_at, updated_at`

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.AvatarURL, &u.Bio, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) Create(ctx context.Context, u *model.User) error {
	const q = `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q, u.Username, u.Email, u.PasswordHash).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return translateUserError(err)
	}
	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	u, err := scanUser(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("用户不存在").Wrap(err)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return u, nil
}

func (r *userRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	// 用 lower() 比较，配合 uq_users_*_lower 表达式索引，
	// 实现「输入 Alice / alice / ALICE 都能登录」。
	const q = `
		SELECT ` + userColumns + `
		FROM users
		WHERE lower(username) = lower($1) OR lower(email) = lower($1)
		LIMIT 1`

	u, err := scanUser(r.pool.QueryRow(ctx, q, login))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 注意：这里返回的是「不存在」，但 service 层会统一转成 401「用户名或密码错误」，
			// 绝不能把 404 直接透给登录接口——那等于告诉攻击者哪些用户名真实存在。
			return nil, errs.ErrNotFound.WithMessage("用户不存在").Wrap(err)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return u, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE lower(email) = lower($1) LIMIT 1`

	u, err := scanUser(r.pool.QueryRow(ctx, q, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("用户不存在").Wrap(err)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return u, nil
}

func (r *userRepo) UpdateProfile(ctx context.Context, id int64, avatarURL, bio *string) (*model.User, error) {
	// COALESCE：传 nil 表示「不修改该字段」，传空字符串才是「清空」
	const q = `
		UPDATE users
		SET avatar_url = COALESCE($2, avatar_url),
		    bio        = COALESCE($3, bio)
		WHERE id = $1
		RETURNING ` + userColumns

	u, err := scanUser(r.pool.QueryRow(ctx, q, id, avatarURL, bio))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound.WithMessage("用户不存在").Wrap(err)
		}
		return nil, translateUserError(err)
	}
	return u, nil
}

func (r *userRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	const q = `UPDATE users SET password_hash = $2 WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id, passwordHash)
	if err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound.WithMessage("用户不存在")
	}
	return nil
}

// isUniqueViolation 判断错误是否为唯一约束冲突。
// constraint 传空字符串表示「任意唯一约束」。
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && (constraint == "" || pgErr.ConstraintName == constraint)
}

// translateUserError 把 PostgreSQL 的底层错误翻译成业务错误。
//
// 关键点：唯一约束冲突【不靠预先查询判断】，而是直接插入后捕获 23505。
// 因为「先查再插」存在竞态：两个并发请求可能同时查到「不存在」然后都去插入。
// 交给数据库的唯一索引兜底，才是并发安全的做法。
func translateUserError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			switch pgErr.ConstraintName {
			case "users_username_key", "uq_users_username_lower":
				return errs.ErrUsernameTaken.Wrap(err)
			case "users_email_key", "uq_users_email_lower":
				return errs.ErrEmailTaken.Wrap(err)
			default:
				return errs.ErrConflict.Wrap(err)
			}
		case "23514": // check_violation
			return errs.ErrValidation.Wrap(err)
		}
	}
	return fmt.Errorf("数据库操作失败: %w", err)
}
