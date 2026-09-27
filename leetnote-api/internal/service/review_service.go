package service

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/db"
	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

// ---------- SM-2 算法 ----------

const (
	// 难度系数下限。低于它间隔会越缩越短，卡片永远出不来。
	minEaseFactor = 1.3
	// 新卡片的初始难度系数
	initialEaseFactor = 2.5
	// rating >= 这个值算「答对」
	passThreshold = 3
)

// SM2State 是 SM-2 算法需要的卡片状态，同时也是它的输出。
//
// 输入与输出结构完全一致，没必要定义两个类型 —— 那只会让调用方
// 到处写类型转换。
type SM2State struct {
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
}

// ApplySM2 是 SM-2 算法的纯函数实现。
//
// 刻意不碰数据库、不碰时间 —— 纯函数才好在测试里把边界情况一次性覆盖
// （比如「答错后难度系数是否保留」「连续答对时间隔怎么涨」）。
//
// 规则：
//   - rating < 3（没答上来）：连续次数归零，间隔重置为 1 天
//   - rating >= 3（答对）：间隔按 1 → 6 → 上次间隔 × 难度系数 增长
//   - 难度系数每次都要调整，下限 1.3
func ApplySM2(state SM2State, rating int) SM2State {
	ease := state.EaseFactor
	if ease <= 0 {
		ease = initialEaseFactor
	}

	// 难度系数调整公式（SM-2 原版）：
	//   EF' = EF + (0.1 - (5-q) * (0.08 + (5-q) * 0.02))
	// q=5 时 +0.1（越来越容易），q=3 时 -0.14，q=0 时 -0.8
	delta := float64(5-rating) * (0.08 + float64(5-rating)*0.02)
	ease += 0.1 - delta
	if ease < minEaseFactor {
		ease = minEaseFactor
	}

	// 没答上来：重来。
	//
	// 注意这里【保留】了调整后的难度系数 —— 这是很多简化实现会写错的地方。
	// 如果把 EF 也重置成 2.5，一张反复答错的难题就永远不会因为「难」
	// 而缩短间隔，复习负担会越来越重。
	if rating < passThreshold {
		return SM2State{EaseFactor: ease, IntervalDays: 1, Repetitions: 0}
	}

	var nextInterval int
	switch state.Repetitions {
	case 0:
		nextInterval = 1
	case 1:
		nextInterval = 6
	default:
		nextInterval = int(math.Round(float64(state.IntervalDays) * ease))
		if nextInterval < 1 {
			nextInterval = 1
		}
	}

	return SM2State{
		EaseFactor:   ease,
		IntervalDays: nextInterval,
		Repetitions:  state.Repetitions + 1,
	}
}

// ---------- 服务 ----------

// ReviewService 负责间隔重复复习。
type ReviewService struct {
	pool    *pgxpool.Pool
	reviews repository.ReviewRepository
	notes   repository.NoteRepository
}

func NewReviewService(
	pool *pgxpool.Pool,
	reviews repository.ReviewRepository,
	notes repository.NoteRepository,
) *ReviewService {
	return &ReviewService{pool: pool, reviews: reviews, notes: notes}
}

// EnsureCardForNote 确保笔记有对应的复习卡。
//
// 笔记创建时调用；更新时也调用一次（幂等），这样在复习功能上线之前
// 建的老笔记也能逐步补上卡片。
func (s *ReviewService) EnsureCardForNote(ctx context.Context, noteID int64) error {
	return s.reviews.CreateForNote(ctx, noteID)
}

// Due 返回到期待复习的卡片。
func (s *ReviewService) Due(ctx context.Context, userID int64, limit int) ([]*model.ReviewCard, error) {
	return s.reviews.GetDue(ctx, userID, limit)
}

// Submit 提交一次复习评分，按 SM-2 更新卡片状态。
//
// 更新卡片与写流水必须原子：否则会出现「间隔变了但没有历史记录」
// 或者反过来，统计就再也不准了。
func (s *ReviewService) Submit(
	ctx context.Context,
	userID, cardID int64,
	in dto.SubmitReviewInput,
) (*model.ReviewCard, error) {
	if in.Rating == nil {
		return nil, errs.ErrValidation.WithDetails(map[string]string{
			"rating": "必须提供评分",
		})
	}
	rating := *in.Rating

	if rating < 0 || rating > 5 {
		return nil, errs.ErrValidation.WithDetails(map[string]string{
			"rating": "评分必须在 0-5 之间",
		})
	}

	card, err := s.reviews.GetOwnedCard(ctx, userID, cardID)
	if err != nil {
		return nil, err
	}

	next := ApplySM2(SM2State{
		EaseFactor:   card.EaseFactor,
		IntervalDays: card.IntervalDays,
		Repetitions:  card.Repetitions,
	}, rating)

	now := time.Now()
	prevInterval := card.IntervalDays

	card.EaseFactor = next.EaseFactor
	card.IntervalDays = next.IntervalDays
	card.Repetitions = next.Repetitions
	card.DueAt = now.AddDate(0, 0, next.IntervalDays)
	card.LastReviewedAt = &now

	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		// 事务内用同一套仓储，只是换成 tx 作为 Querier
		repo := repository.NewReviewRepository(tx)

		if err := repo.UpdateCard(ctx, card); err != nil {
			return err
		}
		return repo.InsertLog(ctx, &model.ReviewLog{
			CardID:       card.ID,
			Rating:       rating,
			PrevInterval: prevInterval,
			NextInterval: next.IntervalDays,
		})
	})
	if err != nil {
		return nil, err
	}

	return card, nil
}

// Stats 返回复习统计。
func (s *ReviewService) Stats(ctx context.Context, userID int64) (*model.ReviewStats, error) {
	return s.reviews.Stats(ctx, userID)
}

// ScheduleEnsureCard 在笔记创建/更新后确保复习卡存在（异步，失败只记日志）。
//
// 用 goroutine 是为了不让笔记保存被额外的一次写操作拖慢。
// 复习卡没建上不影响笔记本身，所以失败不阻断主流程。
func (s *ReviewService) ScheduleEnsureCard(noteID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.reviews.CreateForNote(ctx, noteID); err != nil {
			slog.Warn("创建复习卡失败", "note_id", noteID, "error", err)
		}
	}()
}
