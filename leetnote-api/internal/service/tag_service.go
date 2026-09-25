package service

import (
	"context"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

// TagService 管理标签。
//
// 标签目前是全局共享的（不做用户隔离），这样「动态规划」这类通用标签
// 只需维护一份。将来若需要用户私有标签，加个 owner_id 即可。
type TagService struct {
	tags repository.TagRepository
}

func NewTagService(tags repository.TagRepository) *TagService {
	return &TagService{tags: tags}
}

func (s *TagService) Create(ctx context.Context, in dto.CreateTagInput) (*model.Tag, error) {
	t := &model.Tag{
		Name: in.Name,
		Slug: in.Slug,
		Kind: in.Kind,
	}
	if err := s.tags.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TagService) List(ctx context.Context, kind string) ([]*model.Tag, error) {
	return s.tags.List(ctx, kind)
}

func (s *TagService) Delete(ctx context.Context, id int64) error {
	return s.tags.Delete(ctx, id)
}
