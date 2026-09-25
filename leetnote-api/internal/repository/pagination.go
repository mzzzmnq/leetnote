package repository

// Pagination 是跨仓储共用的分页参数。
//
// 单独抽出来是为了让各仓储的分页行为保持一致，
// 并且把「offset 怎么算」这件事只写一遍。
type Pagination struct {
	Page int
	Size int
}

// Limit 返回 SQL LIMIT 的值。
func (p Pagination) Limit() int {
	if p.Size <= 0 {
		return 20
	}
	return p.Size
}

// Offset 返回 SQL OFFSET 的值。
//
// 注意 OFFSET 的性能问题：LIMIT 20 OFFSET 100000 会让数据库
// 扫描并丢弃前 10 万行。数据量大时要改用游标分页
// （WHERE id < last_id ORDER BY id DESC LIMIT n）。
// 本项目当前数据量下没问题，先把这点记在注释里。
func (p Pagination) Offset() int {
	page := p.Page
	if page < 1 {
		page = 1
	}
	return (page - 1) * p.Limit()
}

// Pages 由总数计算总页数。
func (p Pagination) Pages(total int64) int {
	size := p.Limit()
	if size <= 0 || total <= 0 {
		return 0
	}
	return int((total + int64(size) - 1) / int64(size))
}
