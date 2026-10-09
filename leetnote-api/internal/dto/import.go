package dto

import "github.com/mzzzmnq/leetnote-api/internal/model"

// ImportNoteInput 是「外部工具导入一篇笔记」的请求体。
//
// 【为什么单独一套 DTO 而不是复用 NoteInput】
// 导入方（浏览器脚本）手里的信息形态与站内编辑器不同：
//   - 它只有题目的 slug / title / difficulty，没有 LeetNote 内部的 problem_id
//   - 解法标题它给不出来，而 NoteInput 里 Solutions[].title 是必填
//   - 它可能附带一段 AI 讲解，希望写进正文
//
// 硬套 NoteInput 会让两边互相将就；显式建模「导入」这个场景，
// 服务层再把它翻译成标准 NoteInput 复用既有创建逻辑。
type ImportNoteInput struct {
	Problem   ImportProblem    `json:"problem" binding:"required"`
	ContentMD string           `json:"content_md" binding:"max=100000"`
	Summary   *string          `json:"summary" binding:"omitempty,max=500"`
	Solutions []ImportSolution `json:"solutions" binding:"max=10,dive"`

	// Source 只是来源标签（如 "openleet"），用于日志排查，不入库。
	Source string `json:"source" binding:"max=50"`
}

// ImportProblem 是导入时携带的题目信息。slug 是关联键。
type ImportProblem struct {
	TitleSlug  string  `json:"title_slug" binding:"required,max=200"`
	Title      string  `json:"title" binding:"required,max=200"`
	LeetCodeID *int    `json:"leetcode_id" binding:"omitempty,min=1,max=9999"`
	Difficulty string  `json:"difficulty" binding:"omitempty,oneof=Easy Medium Hard"`
	URL        *string `json:"url" binding:"omitempty,url,max=500"`
}

// ImportSolution 是导入的解法。Title 可省略，服务层会补一个默认标题。
type ImportSolution struct {
	Language string `json:"language" binding:"required,max=20"`
	Code     string `json:"code" binding:"required,max=50000"`
	Title    string `json:"title" binding:"omitempty,max=100"`
}

// ImportNoteResponse 返回导入结果。
//
// ProblemCreated 让导入方知道题目是新建成还是复用了题单里已有的；
// NoteCreated 区分「新建笔记」与「更新了同一题的已有笔记」——
// 插件重复点保存时是更新，界面可以据此提示「已更新」。
type ImportNoteResponse struct {
	Note           NoteResponse `json:"note"`
	ProblemCreated bool         `json:"problem_created"`
	NoteCreated    bool         `json:"note_created"`
}

func NewImportNoteResponse(n *model.Note, problemCreated, noteCreated bool) ImportNoteResponse {
	return ImportNoteResponse{
		Note:           NewNoteResponse(n),
		ProblemCreated: problemCreated,
		NoteCreated:    noteCreated,
	}
}
