package service

import (
	"context"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

// ProblemService 管理题目（全局共享的元数据，不区分用户）。
type ProblemService struct {
	problems repository.ProblemRepository
}

func NewProblemService(problems repository.ProblemRepository) *ProblemService {
	return &ProblemService{problems: problems}
}

func (s *ProblemService) Create(ctx context.Context, in dto.ProblemInput) (*model.Problem, error) {
	p := &model.Problem{
		LeetCodeID: in.LeetCodeID,
		Title:      in.Title,
		TitleSlug:  in.TitleSlug,
		Difficulty: in.Difficulty,
		URL:        in.URL,
	}
	if err := s.problems.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProblemService) GetByID(ctx context.Context, id int64) (*model.Problem, error) {
	return s.problems.GetByID(ctx, id)
}

func (s *ProblemService) List(ctx context.Context, q dto.ProblemQuery) ([]*model.Problem, int64, error) {
	page, size := q.PageQuery.Normalize()

	return s.problems.List(ctx, repository.ProblemFilter{
		Keyword:    q.Keyword,
		Difficulty: q.Difficulty,
		Pagination: repository.Pagination{Page: page, Size: size},
	})
}

func (s *ProblemService) Update(ctx context.Context, id int64, in dto.ProblemInput) (*model.Problem, error) {
	p := &model.Problem{
		ID:         id,
		LeetCodeID: in.LeetCodeID,
		Title:      in.Title,
		TitleSlug:  in.TitleSlug,
		Difficulty: in.Difficulty,
		URL:        in.URL,
	}
	return s.problems.Update(ctx, p)
}

func (s *ProblemService) Delete(ctx context.Context, id int64) error {
	return s.problems.Delete(ctx, id)
}
