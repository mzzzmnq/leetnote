package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
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

// CardScheduler 只暴露「确保复习卡存在」这一个能力。
//
// 定义成窄接口而不是直接依赖 *ReviewService：
// NoteService 只需要「建卡」这一件事，依赖整个复习服务会让两者的
// 变更互相牵连，也不好做测试替身。
type CardScheduler interface {
	ScheduleEnsureCard(noteID int64)
}

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
	reviews   CardScheduler
}

func NewNoteService(
	pool *pgxpool.Pool,
	notes repository.NoteRepository,
	solutions repository.SolutionRepository,
	tags repository.TagRepository,
	problems repository.ProblemRepository,
	aiClient *ai.Client,
	reviews CardScheduler,
) *NoteService {
	return &NoteService{
		pool:      pool,
		notes:     notes,
		solutions: solutions,
		tags:      tags,
		problems:  problems,
		ai:        aiClient,
		reviews:   reviews,
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

		solutions, err := buildSolutions(in.Solutions)
		if err != nil {
			return err
		}
		if err := solutionRepo.ReplaceForNote(ctx, noteID, solutions); err != nil {
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
	// 异步建复习卡（幂等）
	s.ensureReviewCard(noteID)
	return created, nil
}

// ImportResult 是导入的结果。
type ImportResult struct {
	Note           *model.Note
	ProblemCreated bool // 题目是否为本次新建
	NoteCreated    bool // 笔记是否为本次新建（false = 更新了同一题的已有笔记）
}

// ImportNote 把外部工具（如 openleet 浏览器助手）提交的一道题 + 代码导成一篇笔记。
//
// 【为什么复用 Create/Update】导入与站内写入在落库上完全一致（事务、解法、
// 标签、向量、复习卡），差别只在「题目从哪来」和「字段怎么翻译」。把这两件事
// 单独处理、其余照走既有路径，就避免了第二条写入路径 —— 那正是 AGENTS.md
// 提醒过的「两条路径只改一条」的坑。
//
// 【幂等】同一用户对同一题（uq_notes_user_problem 保证一题一篇）已有笔记时，
// 导入会【更新】那篇而不是报 409 —— 插件重复点「保存」应当更新，而不是失败。
func (s *NoteService) ImportNote(
	ctx context.Context, userID int64, in dto.ImportNoteInput,
) (*ImportResult, error) {
	problemID, problemCreated, err := s.ensureProblem(ctx, in.Problem)
	if err != nil {
		return nil, err
	}

	solutions := make([]dto.SolutionInput, 0, len(in.Solutions))
	for _, sol := range in.Solutions {
		title := strings.TrimSpace(sol.Title)
		if title == "" {
			// NoteInput 要求解法标题必填；导入方没有这个概念，给个中性默认值。
			title = "导入的解法"
		}
		solutions = append(solutions, dto.SolutionInput{
			Title:    title,
			Language: sol.Language,
			Code:     sol.Code,
		})
	}

	content := in.ContentMD
	if strings.TrimSpace(content) == "" {
		content = defaultImportContent(in.Problem)
	}

	noteInput := dto.NoteInput{
		ProblemID: &problemID,
		Title:     importNoteTitle(in.Problem),
		ContentMD: content,
		Summary:   in.Summary,
		Status:    model.NoteStatusPublished,
		Solutions: solutions,
	}

	// 已经为这道题写过笔记 → 更新它，保证重复导入幂等
	existing, err := s.notes.GetByProblemID(ctx, userID, problemID)
	switch {
	case err == nil:
		updated, uerr := s.Update(ctx, userID, existing.ID, noteInput)
		if uerr != nil {
			return nil, uerr
		}
		return &ImportResult{Note: updated, ProblemCreated: problemCreated, NoteCreated: false}, nil
	case !errs.IsNotFound(err):
		return nil, err
	}

	created, err := s.Create(ctx, userID, noteInput)
	if err != nil {
		return nil, err
	}
	return &ImportResult{Note: created, ProblemCreated: problemCreated, NoteCreated: true}, nil
}

// ensureProblem 按 slug 复用已有题目；没有则新建。
//
// 题单里的 172 道题大多已存在，导入时会命中 GetBySlug 直接复用，
// 于是「导入」的笔记天然挂到题单题目上 —— 这就是「按 slug 关联」。
func (s *NoteService) ensureProblem(ctx context.Context, p dto.ImportProblem) (int64, bool, error) {
	existing, err := s.problems.GetBySlug(ctx, p.TitleSlug)
	switch {
	case err == nil:
		return existing.ID, false, nil
	case !errs.IsNotFound(err):
		// 只有「确实没找到」才继续新建；连不上库之类的错误要如实上报
		return 0, false, err
	}

	difficulty := p.Difficulty
	if difficulty == "" {
		difficulty = model.DifficultyMedium
	}
	url := p.URL
	if url == nil || *url == "" {
		u := "https://leetcode.cn/problems/" + p.TitleSlug + "/"
		url = &u
	}

	problem := &model.Problem{
		LeetCodeID: p.LeetCodeID,
		Title:      p.Title,
		TitleSlug:  p.TitleSlug,
		Difficulty: difficulty,
		URL:        url,
	}
	if err := s.problems.Create(ctx, problem); err != nil {
		// 并发下可能刚被另一个请求建出来 —— 撞唯一键时回读一次即可
		if again, e := s.problems.GetBySlug(ctx, p.TitleSlug); e == nil {
			return again.ID, false, nil
		}
		return 0, false, err
	}
	return problem.ID, true, nil
}

// importNoteTitle 生成笔记标题，优先带题号（便于在列表里一眼定位）。
func importNoteTitle(p dto.ImportProblem) string {
	if p.LeetCodeID != nil {
		return fmt.Sprintf("%d. %s", *p.LeetCodeID, p.Title)
	}
	return p.Title
}

// defaultImportContent 在导入方没给正文时，生成一段带题目链接的占位正文。
func defaultImportContent(p dto.ImportProblem) string {
	url := "https://leetcode.cn/problems/" + p.TitleSlug + "/"
	if p.URL != nil && *p.URL != "" {
		url = *p.URL
	}
	return fmt.Sprintf("> 由浏览器助手导入\n\n题目：[%s](%s)\n", importNoteTitle(p), url)
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

		solutions, err := buildSolutions(in.Solutions)
		if err != nil {
			return err
		}
		if err := solutionRepo.ReplaceForNote(ctx, id, solutions); err != nil {
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
	// 复习卡是幂等的；对「上线复习功能之前建的老笔记」也顺手补卡
	s.ensureReviewCard(id)
	return updated, nil
}

// ensureReviewCard 在笔记创建/更新后确保复习卡存在。
//
// 复习功能未接时 reviews 为 nil，静默跳过，不影响笔记功能。
func (s *NoteService) ensureReviewCard(noteID int64) {
	if s.reviews == nil {
		return
	}
	s.reviews.ScheduleEnsureCard(noteID)
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

	sol, err := solutionFromInput(in)
	if err != nil {
		return nil, err
	}
	sol.NoteID = noteID

	if err := s.solutions.Create(ctx, sol); err != nil {
		return nil, err
	}
	return sol, nil
}

func (s *NoteService) UpdateSolution(ctx context.Context, userID, solutionID int64, in dto.SolutionInput) (*model.Solution, error) {
	sol, err := solutionFromInput(in)
	if err != nil {
		return nil, err
	}
	sol.ID = solutionID

	// 仓储层用子查询在 UPDATE 内部校验归属，一条 SQL 完成鉴权
	return s.solutions.Update(ctx, userID, sol)
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

// ExplainNote 让 LLM 点评这篇笔记，返回固定结构的讲解。
func (s *NoteService) ExplainNote(ctx context.Context, userID, noteID int64) (*ai.ExplainResult, error) {
	if !s.ai.Enabled() {
		return nil, errs.ErrBadRequest.WithMessage("AI 服务未启用（未配置 AI_SERVICE_URL）")
	}

	// 先确认笔记归属，越权直接 404，连 AI 服务都不会被打到
	if _, err := s.notes.GetByID(ctx, userID, noteID); err != nil {
		return nil, err
	}

	result, err := s.ai.ExplainNote(ctx, userID, noteID)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return result, nil
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

// solutionFromInput 把 DTO 转成领域模型，顺便把语言归一化成规范形式。
//
// 【为什么放在这里而不是 DTO 层】解法有两条写入路径：
//  1. 单独的解法接口（POST/PUT /notes/:id/solutions）
//  2. 「保存笔记」时整批替换（笔记编辑器把解法一起提交）
//
// 两条都走这一个函数，就不会出现「从编辑器存的没归一化」这种漏网之鱼 ——
// 这类"两条路径只改了一条"的问题在数据清洗时最头疼。
func solutionFromInput(in dto.SolutionInput) (*model.Solution, error) {
	lang, ok := model.NormalizeLanguage(in.Language)
	if !ok {
		return nil, errs.ErrBadRequest.WithMessage(
			"不支持的语言 " + strconv.Quote(in.Language) + "，可选：" +
				strings.Join(model.SupportedLanguageNames(), " / "))
	}

	return &model.Solution{
		Title:           in.Title,
		Language:        string(lang),
		Code:            in.Code,
		TimeComplexity:  in.TimeComplexity,
		SpaceComplexity: in.SpaceComplexity,
	}, nil
}

func buildSolutions(inputs []dto.SolutionInput) ([]*model.Solution, error) {
	out := make([]*model.Solution, 0, len(inputs))
	for _, in := range inputs {
		sol, err := solutionFromInput(in)
		if err != nil {
			return nil, err
		}
		out = append(out, sol)
	}
	return out, nil
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
