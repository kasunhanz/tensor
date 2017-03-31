package util

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"math/rand"
)

type PaginationTestSuite struct {
	suite.Suite
}

func (suite *PaginationTestSuite) TestNewQueryParser() {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=100&page=3",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))

	suite.Equal(&Pagination{
		page: 3,
		limit: 100,
		itemCount: 16,
	}, pagination, "Pagination should be equal")
}

func (suite *PaginationTestSuite) TestOffset() {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=3",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))

	suite.Equal(10, pagination.Offset(), "Offset should be equal")
}

func (suite *PaginationTestSuite) TestLimit() {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=3",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))

	suite.Equal(5, pagination.Limit(), "Limit should be equal")
}

func (suite *PaginationTestSuite) TestPage() {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=3",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))

	suite.Equal(3, pagination.Page(), "Limit should be equal")
}

func (suite *PaginationTestSuite) TestLimitParser() {
	suite.Equal(100, limitParser("100"), "Limit should be equal")
	suite.Equal(500, limitParser("500"), "Limit should be equal")
	suite.Equal(10, limitParser("-3"), "Limit should be equal")
}

func (suite *PaginationTestSuite) TestPageParser() {
	suite.Equal(100, pageParser("100"), "Page should be equal")
	suite.Equal(500, pageParser("500"), "Page should be equal")
	suite.Equal(1, pageParser("-3"), "Page should be equal")
}

func (suite *PaginationTestSuite) TestTotalPages()  {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=3",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))

	suite.Equal(4, pagination.totalPages(), "Total should be equal")

}

func (suite *PaginationTestSuite) TestNextPage()  {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=3",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))

	suite.Equal(4, pagination.NextPage(), "Next page should be equal")

}

func (suite *PaginationTestSuite) TestPreviousPage()  {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=3",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))

	suite.Equal(2, pagination.PreviousPage(), "Previous page should be equal")

}

func (suite *PaginationTestSuite) TestHasPage()  {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=500",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))
	suite.Equal(true, pagination.HasPage(), "Has page should be equal")

	c = &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=-1",
			},
		},
	}

	elems = (rand.Perm(16)[:16])
	pagination = NewPagination(c, len(elems))

	suite.Equal(false, pagination.HasPage(), "Has page should be equal")

	c = &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=4",
			},
		},
	}

	elems = (rand.Perm(16)[:16])
	pagination = NewPagination(c, len(elems))

	suite.Equal(false, pagination.HasPage(), "Has page should be equal")

}

func (suite *PaginationTestSuite) TestSkip()  {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=3",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))
	suite.Equal(10, pagination.Skip(), "Skip should be equal")

}

func (suite *PaginationTestSuite) TestEnd()  {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page_size=5&page=3",
			},
		},
	}

	elems := (rand.Perm(16)[:16])
	pagination := NewPagination(c, len(elems))
	suite.Equal(15, pagination.End(), "End should be equal")

}
// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestPaginationTestSuite(t *testing.T) {
	suite.Run(t, new(PaginationTestSuite))
}
