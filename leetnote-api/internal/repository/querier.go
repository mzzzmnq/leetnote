package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier 抽象出 *pgxpool.Pool 与 pgx.Tx 的共同能力。
//
// 为什么需要它：同一个仓储方法，既要在「直接用连接池」时可用，
// 也要能在「事务里」复用。如果不抽象，就得为事务再写一套重复实现。
//
// pgxpool.Pool 和 pgx.Tx 的方法签名完全一致，所以都满足这个接口。
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
