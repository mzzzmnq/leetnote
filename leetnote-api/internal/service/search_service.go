package service

import (
	"context"
	"strings"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

// SearchResult 是一次搜索的完整结果。
type SearchResult struct {
	Notes    []*model.Note
	Problems []*model.Problem
}

// SearchService 提供跨笔记与题目的检索。
type SearchService struct {
	search   repository.SearchRepository
	tags     repository.TagRepository
	problems repository.ProblemRepository
}

func NewSearchService(
	search repository.SearchRepository,
	tags repository.TagRepository,
	problems repository.ProblemRepository,
) *SearchService {
	return &SearchService{search: search, tags: tags, problems: problems}
}

// Search 按关键词检索。
//
// 笔记的标签与题目关联在这里补齐，复用与列表页相同的 enrichNotes，
// 保证搜索结果和列表页的响应结构完全一致（前端能用同一个卡片组件渲染）。
func (s *SearchService) Search(ctx context.Context, userID int64, q dto.SearchQuery) (*SearchResult, error) {
	keyword := strings.TrimSpace(q.Q)
	if keyword == "" {
		return &SearchResult{Notes: []*model.Note{}, Problems: []*model.Problem{}}, nil
	}

	result := &SearchResult{
		Notes:    []*model.Note{},
		Problems: []*model.Problem{},
	}

	// Type 为空表示「都搜」
	if q.Type == "" || q.Type == "note" {
		notes, err := s.search.SearchNotes(ctx, userID, keyword, q.Limit)
		if err != nil {
			return nil, err
		}
		enrichNotes(ctx, notes, s.tags, s.problems)
		result.Notes = notes
	}

	if q.Type == "" || q.Type == "problem" {
		problems, err := s.search.SearchProblems(ctx, keyword, q.Limit)
		if err != nil {
			return nil, err
		}
		result.Problems = problems
	}

	return result, nil
}
