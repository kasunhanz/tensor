package util

import (
	"github.com/gin-gonic/gin"
	"math"
	"strconv"
)

var (
	DefaultLimit = int(10)
	MinLimit     = int(1)
	MaxLimit     = int(500)
	LimitParam   = "page_size"
	DefaultPage  = int(1)
	PageParam    = "page"
)

type Pagination struct {
	limit     int
	page      int
	itemCount int
}

func NewPagination(c *gin.Context, n int) *Pagination {
	p := Pagination{
		page:      pageParser(c.Request.URL.Query().Get(PageParam)),
		limit:     limitParser(c.Request.URL.Query().Get(LimitParam)),
		itemCount: n,
	}

	return &p
}

func (p *Pagination) Offset() int {
	//minimum offset is zero
	if p.limit < 1 || p.page <= 1 {
		return 0
	}

	return int(p.limit * (p.page - 1))
}

func (p *Pagination) Limit() int {
	return p.limit
}
func (p *Pagination) Page() int {
	return p.page
}

func limitParser(limit string) int {
	l, err := strconv.ParseUint(limit, 10, 64)
	lm := int(l)
	if err == nil && lm >= MinLimit && lm <= MaxLimit {
		return lm
	}
	return DefaultLimit
}

func pageParser(page string) int {
	p, err := strconv.ParseUint(page, 10, 64)
	if err == nil {
		return int(p)
	}
	return DefaultPage
}

func (p *Pagination) totalPages() int {
	return int(math.Ceil(float64(p.itemCount) / float64(p.limit)))
}

func (p *Pagination) NextPage() interface{} {
	if p.page >= p.totalPages() {
		return nil
	} else if p.page < 1 {
		return DefaultPage
	}
	return (p.page + 1)
}

func (p *Pagination) PreviousPage() interface{} {
	if p.page <= 1 || p.totalPages() < p.page {
		return nil
	}
	return (p.page - 1)
}

func (p *Pagination) HasPage() bool {
	if p.totalPages() <= 0 {
		return false
	}
	return (p.page <= 0 || p.page > p.totalPages())
}

// for slices
func (p *Pagination) Skip() int {
	skip := p.Offset()
	if skip > p.itemCount {
		skip = p.itemCount
	}
	return skip
}

// for slices
func (p *Pagination) End() int {
	skip := p.Offset()
	size := p.Limit()
	if skip > p.itemCount {
		skip = p.itemCount
	}

	end := skip + size
	if end > p.itemCount {
		end = p.itemCount
	}

	return end
}
