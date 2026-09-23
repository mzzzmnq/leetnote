package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool 创建 PostgreSQL 连接池并做一次连通性探测。
//
// 连接池参数的意义（面试常问）：
//   - MaxConns：最大连接数，超过会阻塞等待。设为 10 是因为 PG 默认 max_connections=100，
//     且要留给其他服务；盲目调大会导致数据库侧连接耗尽。
//   - MinConns：预热连接数，避免冷启动时并发建连的开销。
//   - MaxConnLifetime：强制连接回收，防止内存泄漏累积与 DNS 变更后连到旧实例。
//   - MaxConnIdleTime：空闲连接回收，释放数据库资源。
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析 DATABASE_URL 失败: %w", err)
	}

	poolCfg.MaxConns = 10
	poolCfg.MinConns = 1
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("创建数据库连接池失败: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("数据库连通性检查失败: %w", err)
	}

	return pool, nil
}
