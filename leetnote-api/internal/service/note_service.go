package service

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/ai"
	"github.com/mzzzmnq/leetnote-api/internal/db"
	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

// NoteService 是项目的核心业务。
//
// 它比其它 service 多持有两个东西：
//   - pool：因为「创建/更新笔记」要在事务里同时写三张表
//   - ai  ：用于触发向量生成与相似题检索
type NoteService struct {
	pool      *pgxpool.Pool
	notes     repository.NoteRepository
	solutions repository.SolutionRepository
	tags      repository.TagRepository
	problems  repository.ProblemRepository
	ai        *ai.Client
}

func NewNoteService(
	pool *pgxpool.Pool,
	notes repository.NoteRepository,
	solutions repository.SolutionRepository,
	tags repository.TagRepository,
	problems repository.ProblemRepository,
	aiClient *ai.Client,
) *NoteService {
	return &NoteService{
		pool:      pool,
		notes:     notes,
		solutions: solutions,
		tags:      tags,
		problems:  problems,
		ai:        aiClient,
	}
}

// Create 创建笔记，连同解法与标签一起写入。
//
// 整个过程在一个事务里：任何一步失败都整体回滚，
// 避免出现「笔记建好了但解法只写了一半」这类脏数据。
func (s *NoteService) Create(ctx context.Context, userID int64, in dto.NoteInput) (*model.Note, error) {
	var noteID int64

	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		// 事务内用同一套仓储实现，只是换成 tx 作为 Querier
		noteRepo := repository.NewNoteRepository(tx)
		solutionRepo := repository.NewSolutionRepository(tx)
		tagRepo := repository.NewTagRepository(tx)

		tagIDs, err := normalizeTagIDs(in.TagIDs)
		if err != nil {
			return err
		}
		if err := ensureTagsExist(ctx, tagRepo, tagIDs); err != nil {
			return err
		}

		note := &model.Note{
			UserID:    userID,
			ProblemID: in.ProblemID,
			Title:     in.Title,
			ContentMD: in.ContentMD,
			Summary:   in.Summary,
			Status:    in.Status,
			IsStarred: in.IsStarred,
		}
		if err := noteRepo.Create(ctx, note); err != nil {
			return err
		}
		noteID = note.ID

		if err := solutionRepo.ReplaceForNote(ctx, noteID, buildSolutions(in.Solutions)); err != nil {
			return err
		}
		return tagRepo.SetNoteTags(ctx, noteID, tagIDs)
	})

	if err != nil {
		return nil, err
	}

	// 事务提交后再读一次，返回带关联数据的完整对象
	created, err := s.GetByID(ctx, userID, noteID)
	if err != nil {
		return nil, err
	}

	// 异步生成向量（失败不影响保存）
	s.scheduleEmbed(noteID)
	return created, nil
}

// GetByID 返回笔记详情（含题目、标签、解法）。
func (s *NoteService) GetByID(ctx context.Context, userID, id int64) (*model.Note, error) {
	note, err := s.notes.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	enrichNotes(ctx, []*model.Note{note}, s.tags, s.problems)

	if sols, err := s.solutions.ListByNoteID(ctx, note.ID); err == nil {
		note.Solutions = sols
		note.SolutionCount = len(sols)
	}

	return note, nil
}

// List 返回笔记列表。
func (s *NoteService) List(ctx context.Context, userID int64, q dto.NoteQuery) ([]*model.Note, int64, error) {
	page, size := q.PageQuery.Normalize()

	notes, total, err := s.notes.List(ctx, repository.NoteFilter{
		UserID:     userID,
		Keyword:    q.Keyword,
		Difficulty: q.Difficulty,
		TagID:      q.TagID,
		Status:     q.Status,
		Starred:    q.Starred,
		ProblemID:  q.ProblemID,
		Sort:       q.Sort,
		Pagination: repository.Pagination{Page: page, Size: size},
	})
	if err != nil {
		return nil, 0, err
	}

	enrichNotes(ctx, notes, s.tags, s.problems)
	return notes, total, nil
}

// Update 全量更新笔记及其解法、标签（同样在事务里）。
func (s *NoteService) Update(ctx context.Context, userID, id int64, in dto.NoteInput) (*model.Note, error) {
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		noteRepo := repository.NewNoteRepository(tx)
		solutionRepo := repository.NewSolutionRepository(tx)
		tagRepo := repository.NewTagRepository(tx)

		tagIDs, err := normalizeTagIDs(in.TagIDs)
		if err != nil {
			return err
		}
		if err := ensureTagsExist(ctx, tagRepo, tagIDs); err != nil {
			return err
		}

		_, err = noteRepo.Update(ctx, &model.Note{
			ID:        id,
			UserID:    userID,
			ProblemID: in.ProblemID,
			Title:     in.Title,
			ContentMD: in.ContentMD,
			Summary:   in.Summary,
			Status:    in.Status,
			IsStarred: in.IsStarred,
		})
		if err != nil {
			return err
		}

		if err := solutionRepo.ReplaceForNote(ctx, id, buildSolutions(in.Solutions)); err != nil {
			return err
		}
		return tagRepo.SetNoteTags(ctx, id, tagIDs)
	})

	if err != nil {
		return nil, err
	}

	updated, err := s.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	// 正文变了，向量要重建，否则相似题结果会一直停留在旧内容上
	s.scheduleEmbed(id)
	return updated, nil
}

func (s *NoteService) Delete(ctx context.Context, userID, id int64) error {
	return s.notes.Delete(ctx, userID, id)
}

func (s *NoteService) ToggleStar(ctx context.Context, userID, id int64) (*model.Note, error) {
	return s.notes.ToggleStar(ctx, userID, id)
}

// ---------------------------------------------------------------
// 解法的独立编辑接口
//
// 笔记的 PUT 已经能整体替换解法，这几个接口是给「只想改一个解法」
// 的场景用的，避免为了改一行代码而提交整篇笔记。
// ---------------------------------------------------------------

func (s *NoteService) ListSolutions(ctx context.Context, userID, noteID int64) ([]*model.Solution, error) {
	// 先确认这篇笔记属于当前用户，否则就是越权读别人的解法
	if _, err := s.notes.GetByID(ctx, userID, noteID); err != nil {
		return nil, err
	}
	return s.solutions.ListByNoteID(ctx, noteID)
}

func (s *NoteService) CreateSolution(ctx context.Context, userID, noteID int64, in dto.SolutionInput) (*model.Solution, error) {
	if _, err := s.notes.GetByID(ctx, userID, noteID); err != nil {
		return nil, err
	}

	sol := &model.Solution{
		NoteID:          noteID,
		Title:           in.Title,
		Language:        in.Language,
		Code:            in.Code,
		TimeComplexity:  in.TimeComplexity,
		SpaceComplexity: in.SpaceComplexity,
	}
	if err := s.solutions.Create(ctx, sol); err != nil {
		return nil, err
	}
	return sol, nil
}

func (s *NoteService) UpdateSolution(ctx context.Context, userID, solutionID int64, in dto.SolutionInput) (*model.Solution, error) {
	// 仓储层用子查询在 UPDATE 内部校验归属，一条 SQL 完成鉴权
	return s.solutions.Update(ctx, userID, &model.Solution{
		ID:              solutionID,
		Title:           in.Title,
		Language:        in.Language,
		Code:            in.Code,
		TimeComplexity:  in.TimeComplexity,
		SpaceComplexity: in.SpaceComplexity,
	})
}

func (s *NoteService) DeleteSolution(ctx context.Context, userID, solutionID int64) error {
	return s.solutions.DeleteOwned(ctx, userID, solutionID)
}

// ---------------------------------------------------------------

// enrichNotes 批量补齐笔记的题目与标签。
//
// 【关键】用两次「按 ID 批量查」而不是循环里逐条查——
// 后者就是典型的 N+1：20 条笔记会产生 1 + 20 + 20 = 41 次查询。
// 现在固定是 3 次（列表 + 批量题目 + 批量标签），与条数无关。
//
// 抽成包级函数是为了让搜索模块也能复用同一套补齐逻辑。
func enrichNotes(
	ctx context.Context,
	notes []*model.Note,
	tags repository.TagRepository,
	problems repository.ProblemRepository,
) {
	if len(notes) == 0 {
		return
	}

	noteIDs := make([]int64, 0, len(notes))
	problemIDs := make([]int64, 0, len(notes))
	for _, n := range notes {
		noteIDs = append(noteIDs, n.ID)
		if n.ProblemID != nil {
			problemIDs = append(problemIDs, *n.ProblemID)
		}
	}

	if tagMap, err := tags.ListByNoteIDs(ctx, noteIDs); err == nil {
		for _, n := range notes {
			n.Tags = tagMap[n.ID]
		}
	}

	if len(problemIDs) > 0 {
		if problemMap, err := problems.GetByIDs(ctx, problemIDs); err == nil {
			for _, n := range notes {
				if n.ProblemID != nil {
					n.Problem = problemMap[*n.ProblemID]
				}
			}
		}
	}
}

// ---------------------------------------------------------------
// 相似题推荐
// ---------------------------------------------------------------

// FindSimilar 返回与指定笔记相似的笔记。
func (s *NoteService) FindSimilar(
	ctx context.Context,
	userID, noteID int64,
	limit int,
) ([]ai.SimilarNote, string, error) {
	if !s.ai.Enabled() {
		return nil, "", errs.ErrBadRequest.WithMessage("AI 服务未启用（未配置 AI_SERVICE_URL）")
	}

	// 先确认这篇笔记属于当前用户。
	// AI 服务内部也会用 user_id 过滤，这里是更早的一道防线：
	// 拿别人的 note_id 会直接 404，连 AI 服务都不会被打到。
	if _, err := s.notes.GetByID(ctx, userID, noteID); err != nil {
		return nil, "", err
	}

	items, model, err := s.ai.FindSimilar(ctx, userID, noteID, limit)
	if err != nil {
		return nil, "", errs.ErrInternal.Wrap(err)
	}
	return items, model, nil
}

// scheduleEmbed 异步触发向量生成。
//
// 【为什么异步】embedding 要调外部模型，可能耗时几百毫秒到几秒。
// 用户保存笔记时不该干等 —— 向量只是「相似题推荐」的附加数据。
//
// 【为什么新开 context】不能用请求的 context：请求返回后它就被取消了，
// 向量就生成不出来。这里用一个独立的、带超时的 context。
//
// 【生产环境的做法】应该换成消息队列（失败可重试、可观测、不丢任务）。
// 这里用 goroutine 是权衡后的简化，失败只记日志。
func (s *NoteService) scheduleEmbed(noteID int64) {
	if !s.ai.Enabled() {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := s.ai.EmbedNote(ctx, noteID); err != nil {
			slog.Warn("生成笔记向量失败，相似题推荐将不包含这篇",
				"note_id", noteID, "error", err)
		}
	}()
}

func buildSolutions(inputs []dto.SolutionInput) []*model.Solution {
	out := make([]*model.Solution, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, &model.Solution{
			Title:           in.Title,
			Language:        in.Language,
			Code:            in.Code,
			TimeComplexity:  in.TimeComplexity,
			SpaceComplexity: in.SpaceComplexity,
		})
	}
	return out
}

// normalizeTagIDs 去重并排序。
//
// 必须去重：同一个 tag_id 传两次会让「存在性校验的数量比对」误判失败，
// 也会让 note_tags 插入产生重复（虽然有 ON CONFLICT 兜底）。
func normalizeTagIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errs.ErrBadRequest.WithMessage("tag_ids 含非法值")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func ensureTagsExist(ctx context.Context, repo repository.TagRepository, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	n, err := repo.CountExisting(ctx, ids)
	if err != nil {
		return err
	}
	if int(n) != len(ids) {
		return errs.ErrBadRequest.WithMessage("包含不存在的标签")
	}
	return nil
}
