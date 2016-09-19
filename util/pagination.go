package util

import (
	"strconv"
	"github.com/gin-gonic/gin"
	"math"
)

var (
	DefaultLimit = int(10)
	MinLimit = int(1)
	MaxLimit = int(500)
	LimitParam = "page_size"
	DefaultPage = int(1)
	PageParam = "page"
)

type Pagination struct {
	Limit     int
	Page      int
	ItemCount int
}

func NewPagination(c *gin.Context, n int) *Pagination {
	p := Pagination{
		Page : pageParser(c.Request.URL.Query().Get(PageParam)),
		Limit : limitParser(c.Request.URL.Query().Get(LimitParam)),
		ItemCount: n,
	}

	return &p;
}

func (p *Pagination) Offset() int {
	//minimum offset is zero
	if p.Limit < 1 || p.Page <= 1 {
		return 0
	}

	//-1 because mongodb offset starts with 0
	return int((p.Limit * p.Page) - 1)
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
	return int(math.Floor(float64(p.ItemCount) / float64(p.Limit)))
}

func (p *Pagination) NextPage() interface{} {
	if (p.totalPages() <= p.Page) {
		return nil
	} else if (p.Page < 1) {
		return DefaultPage
	}
	return (p.Page + 1)
}

func (p *Pagination) PreviousPage() interface{} {
	if p.Page <= 1 || p.totalPages() < p.Page {
		return nil
	}
	return (p.Page - 1)
}

func (p *Pagination) HasPage() bool {
	if p.totalPages() <= 0 {
		return false
	}
	return (p.Page <= 0 || p.Page > p.totalPages())
}