// Package cache 封装 Redis 客户端。
//
// 【设计原则：Redis 不可用不能拖垮主服务】
// Redis 在这里只承载「接口限流」这类非核心能力。连不上时把客户端标记为
// 未启用，限流中间件据此降级为放行（fail-open），服务照常启动。
// 这样就避免了又一次「换个环境服务起不来」的坑。
package cache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrDisabled 表示 Redis 未启用（地址为空，或启动时连接失败）。
var ErrDisabled = errors.New("redis 未启用")

// Client 是 Redis 客户端的包装。请用 New 构造，零值不可用。
type Client struct {
	rdb     *redis.Client
	enabled bool
}

// New 创建并探活一个 Redis 客户端。
//
// addr 为空时直接返回「未启用」的客户端（测试、或本机没跑 Redis 时用）。
//
// 连接/PING 失败【不返回 error】——只记一条日志并降级。
// 调用方（main）不该因为 Redis 挂了就起不来：限流是保护性能力，
// 它的可用性不该高于主服务本身。
func New(ctx context.Context, addr string) *Client {
	if addr == "" {
		slog.Info("未配置 REDIS_ADDR，Redis 相关能力（限流）将关闭")
		return &Client{enabled: false}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		PoolSize:     10,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		slog.Warn("Redis 连接失败，限流将降级为放行（fail-open）",
			"addr", addr, "error", err)
		_ = rdb.Close()
		return &Client{enabled: false}
	}

	slog.Info("Redis 就绪", "addr", addr)
	return &Client{rdb: rdb, enabled: true}
}

// Enabled 表示 Redis 是否可用。
func (c *Client) Enabled() bool {
	return c != nil && c.enabled
}

// Ping 探测连通性，供 /health 使用。
func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return ErrDisabled
	}
	return c.rdb.Ping(ctx).Err()
}

// Close 关闭底层连接。
func (c *Client) Close() error {
	if !c.Enabled() {
		return nil
	}
	return c.rdb.Close()
}
