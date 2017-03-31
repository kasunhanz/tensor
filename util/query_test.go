package util

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"gopkg.in/mgo.v2/bson"
)

type QueryTestSuite struct {
	suite.Suite
}

func (suite *QueryTestSuite) TestNewQueryParser() {
	values := url.Values{}
	values["test"] = []string{"hello", "hello"}

	c := &gin.Context{
		Request: &http.Request{
			Form:values,
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(parser.context, c, "Context should be equal")
	suite.Equal(parser.From, values, "Form should be equal")
}

func (suite *QueryTestSuite) TestOrderBy() {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal("-mongodb", parser.OrderBy(), "Orderby should be equal")
}

func (suite *QueryTestSuite) TestMatch() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1=test&param2=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1": "test",
		"param2": "test2",
	}, parser.Match(fields, query), "Match should be equal")
}

func (suite *QueryTestSuite) TestExact() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__exact=test&param2__exact=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$regex": bson.RegEx{Pattern: "^test$", Options: ""}},
		"param2": bson.M{"$regex": bson.RegEx{Pattern: "^test2$", Options: ""}},
	}, parser.Exact(fields, query), "Exact should be equal")
}

func (suite *QueryTestSuite) TestIExact() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__iexact=test&param2__iexact=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$regex": bson.RegEx{Pattern: "^test$", Options: "i"}},
		"param2": bson.M{"$regex": bson.RegEx{Pattern: "^test2$", Options: "i"}},
	}, parser.IExact(fields, query), "IExact should be equal")
}

func (suite *QueryTestSuite) TestContains() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__contains=test&param2__contains=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$regex": bson.RegEx{Pattern: ".*test.*", Options: ""}},
		"param2": bson.M{"$regex": bson.RegEx{Pattern: ".*test2.*", Options: ""}},
	}, parser.Contains(fields, query), "Contains should be equal")
}

func (suite *QueryTestSuite) TestIContains() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__icontains=test&param2__icontains=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$regex": bson.RegEx{Pattern: ".*test.*", Options: "i"}},
		"param2": bson.M{"$regex": bson.RegEx{Pattern: ".*test2.*", Options: "i"}},
	}, parser.IContains(fields, query), "IContains should be equal")
}

func (suite *QueryTestSuite) TestStartswith() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__startswith=test&param2__startswith=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$regex": bson.RegEx{Pattern: "^test", Options: ""}},
		"param2": bson.M{"$regex": bson.RegEx{Pattern: "^test2", Options: ""}},
	}, parser.Startswith(fields, query), "Startswith should be equal")
}

func (suite *QueryTestSuite) TestIStartswith() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__istartswith=test&param2__istartswith=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$regex": bson.RegEx{Pattern: "^test", Options: "i"}},
		"param2": bson.M{"$regex": bson.RegEx{Pattern: "^test2", Options: "i"}},
	}, parser.IStartswith(fields, query), "Startswith should be equal")
}

func (suite *QueryTestSuite) TestEndswith() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__endswith=test&param2__endswith=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$regex": bson.RegEx{Pattern: "test$", Options: ""}},
		"param2": bson.M{"$regex": bson.RegEx{Pattern: "test2$", Options: ""}},
	}, parser.Endswith(fields, query), "Startswith should be equal")
}

func (suite *QueryTestSuite) TestIEndswith() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__iendswith=test&param2__iendswith=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$regex": bson.RegEx{Pattern: "test$", Options: "i"}},
		"param2": bson.M{"$regex": bson.RegEx{Pattern: "test2$", Options: "i"}},
	}, parser.IEndswith(fields, query), "Startswith should be equal")
}

func (suite *QueryTestSuite) TestGt() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__gt=test&param2__gt=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$gt": "test"},
		"param2": bson.M{"$gt": "test2"},
	}, parser.Gt(fields, query), "Gt should be equal")
}

func (suite *QueryTestSuite) TestGte() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__gte=test&param2__gte=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$gte": "test"},
		"param2": bson.M{"$gte": "test2"},
	}, parser.Gte(fields, query), "Gt should be equal")
}

func (suite *QueryTestSuite) TestLt() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__lt=test&param2__lt=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$lt": "test"},
		"param2": bson.M{"$lt": "test2"},
	}, parser.Lt(fields, query), "lt should be equal")
}

func (suite *QueryTestSuite) TestLte() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__lte=test&param2__lte=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$lte": "test"},
		"param2": bson.M{"$lte": "test2"},
	}, parser.Lte(fields, query), "Lte should be equal")
}

func (suite *QueryTestSuite) TestIsNull() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__isnull=test&param2__isnull=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$or": bson.M{"param1": nil, "$exists": false}},
		"param2": bson.M{"$or": bson.M{"param2": nil, "$exists": false}},
	}, parser.IsNull(fields, query), "IsNull should be equal")
}

func (suite *QueryTestSuite) TestIn() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__in=test,test,test&param2__in=test2,test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$in": []string{"test", "test", "test"}},
		"param2": bson.M{"$in": []string{"test2", "test2"}},
	}, parser.In(fields, query), "In should be equal")
}

func (suite *QueryTestSuite) TestEq() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__eq=test&param2__eq=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$eq": "test"},
		"param2": bson.M{"$eq": "test2"},
	}, parser.Eq(fields, query), "Eq should be equal")
}

func (suite *QueryTestSuite) TestNe() {
	query := bson.M{}
	fields := []string{"param1", "param2"}

	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "order_by=-mongodb&param1__ne=test&param2__ne=test2",
			},
		},
	}
	parser := NewQueryParser(c)

	suite.Equal(bson.M{
		"param1":  bson.M{"$ne": "test"},
		"param2": bson.M{"$ne": "test2"},
	}, parser.Ne(fields, query), "Ne should be equal")
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestQuerySuite(t *testing.T) {
	suite.Run(t, new(QueryTestSuite))
}
