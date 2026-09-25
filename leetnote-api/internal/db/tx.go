package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx 在一个事务里执行 fn，出错自动回滚，成功自动提交。
//
// 为什么需要事务：创建一篇笔记要同时写 notes、solutions、note_tags 三张表。
// 如果中途失败（比如某个 tag_id 不存在），必须整体回滚——
// 否则会留下「有笔记但没解法」这种半截数据，用户看到的就是脏数据。
//
// 用法：
//
//	err := db.WithTx(ctx, pool, func(tx pgx.Tx) error {
//	    noteRepo := repository.NewNoteRepository(tx)   // 事务内复用同一套仓储
//	    ...
//	})
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	// 已提交后再 Rollback 是安全的 no-op；用 defer 保证任何 panic 或提前返回都会回滚
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}
