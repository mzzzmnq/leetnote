package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// newTestClient 起一个 miniredis（纯内存、无外部依赖）并返回启用中的客户端。
func newTestClient(t *testing.T) *Client {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("启动 miniredis 失败: %v", err)
	}
	t.Cleanup(mr.Close)

	c := New(context.Background(), mr.Addr())
	if !c.Enabled() {
		t.Fatal("客户端应处于启用状态")
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestAllowN_AllowsUpToLimitThenDenies(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	for i := range 3 {
		allowed, remaining, err := c.AllowN(ctx, "rl:test", 3, time.Minute)
		if err != nil {
			t.Fatalf("第 %d 次出错: %v", i+1, err)
		}
		if !allowed {
			t.Fatalf("第 %d 次不应被拒绝", i+1)
		}
		if want := 3 - i - 1; remaining != want {
			t.Errorf("第 %d 次剩余额度 = %d，期望 %d", i+1, remaining, want)
		}
	}

	allowed, remaining, err := c.AllowN(ctx, "rl:test", 3, time.Minute)
	if err != nil {
		t.Fatalf("超限时出错: %v", err)
	}
	if allowed {
		t.Error("超过上限后应被拒绝")
	}
	if remaining != 0 {
		t.Errorf("被拒时剩余额度 = %d，期望 0", remaining)
	}
}

func TestAllowN_KeyIsIndependent(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	if ok, _, _ := c.AllowN(ctx, "rl:a", 1, time.Minute); !ok {
		t.Fatal("a 首次应放行")
	}
	if ok, _, _ := c.AllowN(ctx, "rl:a", 1, time.Minute); ok {
		t.Fatal("a 第二次应被拒")
	}
	// 另一个 key 的额度不受影响
	if ok, _, _ := c.AllowN(ctx, "rl:b", 1, time.Minute); !ok {
		t.Fatal("b 应独立计数并放行")
	}
}

func TestAllowN_WindowSlides(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	// 窗口故意取得很小，用真实时间等待来验证「滑过去就恢复」。
	// （不能用 miniredis.FastForward：脚本用的是 Go 侧传入的真实时间戳，
	//   而 FastForward 只推进 miniredis 自己的时钟。）
	window := 100 * time.Millisecond

	for i := range 2 {
		if ok, _, _ := c.AllowN(ctx, "rl:win", 2, window); !ok {
			t.Fatalf("窗口内第 %d 次应放行", i+1)
		}
	}
	if ok, _, _ := c.AllowN(ctx, "rl:win", 2, window); ok {
		t.Fatal("窗口内第 3 次应被拒")
	}

	time.Sleep(150 * time.Millisecond)

	if ok, _, _ := c.AllowN(ctx, "rl:win", 2, window); !ok {
		t.Fatal("窗口滑过后应重新放行")
	}
}

func TestAllowN_DisabledClientFailsOpen(t *testing.T) {
	c := New(context.Background(), "")
	if c.Enabled() {
		t.Fatal("空地址应得到未启用的客户端")
	}

	allowed, _, err := c.AllowN(context.Background(), "rl:x", 1, time.Minute)
	if err != nil {
		t.Fatalf("未启用时不应报错: %v", err)
	}
	if !allowed {
		t.Error("未启用时应放行（fail-open）")
	}
}

func TestPing_DisabledReturnsErrDisabled(t *testing.T) {
	c := New(context.Background(), "")
	if err := c.Ping(context.Background()); err == nil {
		t.Error("未启用客户端的 Ping 应返回错误")
	}
}
