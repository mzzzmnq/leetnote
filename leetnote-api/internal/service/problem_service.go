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
	// 需要标签仓储才能把「专题」一起返回给前端
	tags repository.TagRepository
}

func NewProblemService(problems repository.ProblemRepository, tags repository.TagRepository) *ProblemService {
	return &ProblemService{problems: problems, tags: tags}
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

// GetByID 返回题目详情，附带所属专题。
func (s *ProblemService) GetByID(ctx context.Context, id int64) (*model.Problem, error) {
	p, err := s.problems.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.attachTags(ctx, []*model.Problem{p})
	return p, nil
}

func (s *ProblemService) List(ctx context.Context, q dto.ProblemQuery) ([]*model.Problem, int64, error) {
	page, size := q.PageQuery.Normalize()

	problems, total, err := s.problems.List(ctx, repository.ProblemFilter{
		Keyword:    q.Keyword,
		Difficulty: q.Difficulty,
		TagID:      q.TagID,
		Pagination: repository.Pagination{Page: page, Size: size},
	})
	if err != nil {
		return nil, 0, err
	}

	s.attachTags(ctx, problems)
	return problems, total, nil
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

	updated, err := s.problems.Update(ctx, p)
	if err != nil {
		return nil, err
	}
	s.attachTags(ctx, []*model.Problem{updated})
	return updated, nil
}

func (s *ProblemService) Delete(ctx context.Context, id int64) error {
	return s.problems.Delete(ctx, id)
}

// attachTags 批量补齐题目的专题标签。
//
// 与 enrichNotes 同样的思路：一次批量查，避免 N+1。
func (s *ProblemService) attachTags(ctx context.Context, problems []*model.Problem) {
	if len(problems) == 0 {
		return
	}

	ids := make([]int64, 0, len(problems))
	for _, p := range problems {
		ids = append(ids, p.ID)
	}

	tagMap, err := s.tags.ListByProblemIDs(ctx, ids)
	if err != nil {
		return // 标签只是附加信息，查不到不影响题目本身的返回
	}
	for _, p := range problems {
		p.Tags = tagMap[p.ID]
	}
}
