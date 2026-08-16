// 分页参数（docs/04-API接口规范 §3）
package common

const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

type PageQuery struct {
	Page     int
	PageSize int
	Offset   int
}

func ParsePage(page, pageSize int) PageQuery {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return PageQuery{Page: page, PageSize: pageSize, Offset: (page - 1) * pageSize}
}
