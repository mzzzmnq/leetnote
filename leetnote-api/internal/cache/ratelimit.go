package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// slidingWindowScript 是「滑动窗口限流」的原子实现。
//
// 数据结构：有序集合（ZSET），score 与 member 都带请求时间戳。
//
//	ZREMRANGEBYSCORE  清掉窗口外的旧记录
//	ZCARD             统计窗口内还剩多少次请求
//	未超限 → ZADD 记录本次，并 PEXPIRE 让 key 自动过期（不占内存）
//
// 【为什么必须用 Lua】「读计数 → 判断 → 写入」三步必须原子完成。
// 拆成多条命令时，高并发下会出现竞态：两个请求都读到 count < limit，双双放行，
// 实际放行数超过上限。Lua 在 Redis 里单线程执行，天然没有这个问题。
//
// 返回 {allowed(1/0), remaining}。
var slidingWindowScript = redis.NewScript(`
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit  = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local used = redis.call('ZCARD', key)

if used < limit then
    redis.call('ZADD', key, now, member)
    redis.call('PEXPIRE', key, window)
    return {1, limit - used - 1}
end

redis.call('PEXPIRE', key, window)
return {0, 0}
`)

// AllowN 判断 key 在 window 内是否还可以再发生一次请求。
//
// 返回 (allowed, remaining, error)。Redis 未启用或 limit<=0 时一律放行
// （fail-open）——调用方据此把限流做成「有则更好，没有也不坏事」。
func (c *Client) AllowN(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	if !c.Enabled() || limit <= 0 || window <= 0 {
		return true, limit, nil
	}

	now := time.Now().UnixMilli()
	// member 必须唯一，否则同一毫秒内的多次请求会互相覆盖，计数偏低。
	member := fmt.Sprintf("%d-%s", now, randomToken())

	res, err := slidingWindowScript.Run(ctx, c.rdb,
		[]string{key},
		now, window.Milliseconds(), limit, member,
	).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	if len(res) != 2 {
		return false, 0, fmt.Errorf("限流脚本返回 %d 个值，预期 2 个", len(res))
	}
	return res[0] == 1, int(res[1]), nil
}

// randomToken 生成 8 字节随机十六进制串，仅用于让 ZSET member 唯一。
func randomToken() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
