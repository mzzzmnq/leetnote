package dto

// SearchQuery 是搜索请求参数。
type SearchQuery struct {
	Q string `form:"q" binding:"required,min=1,max=100"`
	// Type 逗号分隔，可选 note / problem，默认两者都搜
	Type  string `form:"type" binding:"omitempty,oneof=note problem"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// SearchResponse 把两类结果放在同一个响应里。
//
// 为什么不做成两个接口：用户在搜索框里输入一次，前端需要同时看到
// 笔记和题目结果（分组展示）。两个请求会让加载状态和竞态处理变复杂。
type SearchResponse struct {
	Keyword  string            `json:"keyword"`
	Notes    []NoteListItem    `json:"notes"`
	Problems []ProblemResponse `json:"problems"`
}
