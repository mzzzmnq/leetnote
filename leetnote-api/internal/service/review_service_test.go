package service_test

import (
	"math"
	"testing"

	"github.com/mzzzmnq/leetnote-api/internal/service"
)

// 新卡片的初始状态（与数据库默认值一致）
func newCard() service.SM2State {
	return service.SM2State{EaseFactor: 2.5, IntervalDays: 0, Repetitions: 0}
}

func approx(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// 连续答对时间隔应按 1 → 6 → interval×EF 增长。
func TestApplySM2IntervalProgression(t *testing.T) {
	state := newCard()

	// 第 1 次答对
	first := service.ApplySM2(state, 4)
	if first.IntervalDays != 1 {
		t.Errorf("第 1 次答对间隔应为 1 天, 实际 %d", first.IntervalDays)
	}
	if first.Repetitions != 1 {
		t.Errorf("连续次数应为 1, 实际 %d", first.Repetitions)
	}

	// 第 2 次答对
	second := service.ApplySM2(first, 4)
	if second.IntervalDays != 6 {
		t.Errorf("第 2 次答对间隔应为 6 天, 实际 %d", second.IntervalDays)
	}
	if second.Repetitions != 2 {
		t.Errorf("连续次数应为 2, 实际 %d", second.Repetitions)
	}

	// 第 3 次：6 * EF
	third := service.ApplySM2(second, 4)
	want := int(math.Round(6 * second.EaseFactor))
	if third.IntervalDays != want {
		t.Errorf("第 3 次间隔应为 6 × EF = %d, 实际 %d", want, third.IntervalDays)
	}
	if third.IntervalDays <= second.IntervalDays {
		t.Error("间隔应单调递增")
	}
}

// 答错：间隔重置为 1、连续次数归零，但【难度系数要保留】。
//
// 这是很多简化实现会写错的地方：如果把 EF 也重置成 2.5，
// 一张反复答错的难题就永远不会因为「难」而缩短间隔。
func TestApplySM2WrongAnswerResetsIntervalButKeepsEase(t *testing.T) {
	// 先答对几次把 EF 抬高
	state := newCard()
	for range 3 {
		state = service.ApplySM2(state, 5)
	}
	if state.EaseFactor <= 2.5 {
		t.Fatalf("连续满分后 EF 应大于 2.5, 实际 %.3f", state.EaseFactor)
	}
	easeBefore := state.EaseFactor

	after := service.ApplySM2(state, 1)

	if after.IntervalDays != 1 {
		t.Errorf("答错后间隔应重置为 1 天, 实际 %d", after.IntervalDays)
	}
	if after.Repetitions != 0 {
		t.Errorf("答错后连续次数应归零, 实际 %d", after.Repetitions)
	}
	// EF 会被本次低分拉低，但绝不能被重置回 2.5
	if after.EaseFactor <= 1.3 {
		t.Errorf("EF 不应低于下限 1.3, 实际 %.3f", after.EaseFactor)
	}
	if after.EaseFactor == 2.5 {
		t.Error("答错后 EF 被重置成了初始值 2.5 —— 难度信息丢失了")
	}
	if after.EaseFactor >= easeBefore {
		t.Errorf("答错应该拉低 EF: 之前 %.3f, 之后 %.3f", easeBefore, after.EaseFactor)
	}
}

// 难度系数下限是 1.3，反复答错也不能击穿。
func TestApplySM2EaseFactorFloor(t *testing.T) {
	state := newCard()

	for range 20 {
		state = service.ApplySM2(state, 0)
	}

	if state.EaseFactor < 1.3 {
		t.Errorf("EF 下限应为 1.3, 实际 %.3f", state.EaseFactor)
	}
	if !approx(state.EaseFactor, 1.3) {
		t.Errorf("反复答 0 分后 EF 应贴着下限 1.3, 实际 %.3f", state.EaseFactor)
	}
}

// 评分越高，难度系数增长越多 —— 这是 EF 公式的方向性验证。
func TestApplySM2EaseFactorDirection(t *testing.T) {
	best := service.ApplySM2(newCard(), 5)
	good := service.ApplySM2(newCard(), 4)
	barely := service.ApplySM2(newCard(), 3)

	if !(best.EaseFactor > good.EaseFactor && good.EaseFactor > barely.EaseFactor) {
		t.Errorf("评分为 5/4/3 时 EF 应递减, 实际 %.4f / %.4f / %.4f",
			best.EaseFactor, good.EaseFactor, barely.EaseFactor)
	}

	// q=5 时 delta = 0.1 - 0*(...) = +0.1
	if !approx(best.EaseFactor, 2.6) {
		t.Errorf("满分后 EF 应为 2.6, 实际 %.4f", best.EaseFactor)
	}
}

// 边界：评分为 3 恰好是「答对」的下界。
func TestApplySM2PassThreshold(t *testing.T) {
	passed := service.ApplySM2(newCard(), 3)
	if passed.Repetitions != 1 || passed.IntervalDays != 1 {
		t.Errorf("评分 3 应算答对（连续 1 次、间隔 1 天）, 实际 reps=%d interval=%d",
			passed.Repetitions, passed.IntervalDays)
	}

	failed := service.ApplySM2(newCard(), 2)
	if failed.Repetitions != 0 || failed.IntervalDays != 1 {
		t.Errorf("评分 2 应算答错（连续归零、间隔 1 天）, 实际 reps=%d interval=%d",
			failed.Repetitions, failed.IntervalDays)
	}
}

// EF 为 0（脏数据或未初始化）时应回退到初始值而不是算出负数间隔。
func TestApplySM2ZeroEaseFactorFallsBack(t *testing.T) {
	wonky := service.SM2State{EaseFactor: 0, IntervalDays: 10, Repetitions: 3}

	got := service.ApplySM2(wonky, 4)

	if got.EaseFactor < 1.3 {
		t.Errorf("EF 异常时应回退到 >=1.3, 实际 %.3f", got.EaseFactor)
	}
	if got.IntervalDays <= 0 {
		t.Errorf("间隔必须为正数, 实际 %d", got.IntervalDays)
	}
}

// 间隔永远不为零 —— 否则卡片会立刻再次到期，陷入死循环。
func TestApplySM2IntervalAlwaysPositive(t *testing.T) {
	state := service.SM2State{EaseFactor: 1.3, IntervalDays: 1, Repetitions: 5}

	for _, rating := range []int{0, 1, 2, 3, 4, 5} {
		got := service.ApplySM2(state, rating)
		if got.IntervalDays < 1 {
			t.Errorf("rating=%d 时算出了非正间隔 %d", rating, got.IntervalDays)
		}
	}
}
