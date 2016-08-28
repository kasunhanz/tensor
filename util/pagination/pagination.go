package pagination

import (
	"strconv"
	"github.com/gin-gonic/gin"
	"pearson.com/tensor/util"
)

var (
	DefaultLimit = uint64(10)
	MinLimit = uint64(1)
	MaxLimit = uint64(20)
	LimitParam = "l"
	DefaultPage = uint64(0)
	PageParam = "p"
)

type Pagination struct {
	Limit uint64
	Page  uint64
}

func NewPagination(c *gin.Context) *Pagination {

	pagination := Pagination{}

	if page, err := util.GetIntParam("p", c); err == nil {
		pagination.Page = page;
	}

	if limit, err := util.GetIntParam("l", c); err == nil {
		pagination.Limit = limit;
	}

	return &pagination;
}

func (pagi Pagination) Offset() uint64 {
	return pagi.Limit * pagi.Page
}

func limitParser(limit string) uint64 {
	l, err := strconv.ParseUint(limit, 10, 64)
	if err == nil && l >= MinLimit && l <= MaxLimit {
		return l
	}
	return DefaultLimit
}

func pageParser(page string) uint64 {
	p, err := strconv.ParseUint(page, 10, 64)
	if err == nil {
		return p
	}
	return DefaultPage
}