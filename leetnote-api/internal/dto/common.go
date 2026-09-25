package dto

// PageQuery 是列表接口共用的分页参数（从 query string 解析）。
//
// 注意：binding 里的 omitempty 对数值类型的零值会跳过校验，
// 所以 ?page=0 不会被判为非法，而是被当成「没传」走默认值。
// 这里是有意为之（分页传 0 当成第一页比报错更友好）。
// 若某个参数需要「0 必须报错」的语义，请用 *int（见 stats 的 days）。
type PageQuery struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Normalize 应用默认值。
//
// 注意：越界值（page < 1 / size > 100）在进到这里之前就已经被 binding
// 校验拦下并返回 422 了，所以下面的边界处理是【防御性】的——
// 防止将来有人绕过 HTTP 层直接调用 service 时拿到非法分页。
func (q PageQuery) Normalize() (page, size int) {
	page = q.Page
	if page < 1 {
		page = 1
	}

	size = q.Size
	if size < 1 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return page, size
}
