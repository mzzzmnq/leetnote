package service

import (
	"context"

	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

// StatsService 提供刷题数据统计。
type StatsService struct {
	stats repository.StatsRepository
}

func NewStatsService(stats repository.StatsRepository) *StatsService {
	return &StatsService{stats: stats}
}

func (s *StatsService) Overview(ctx context.Context, userID int64) (*model.StatsOverview, error) {
	return s.stats.Overview(ctx, userID)
}

func (s *StatsService) Trend(ctx context.Context, userID int64, days int) ([]*model.TrendPoint, error) {
	return s.stats.Trend(ctx, userID, days)
}
