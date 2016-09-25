package util

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

var (
	OrderParam = "order_by"
)

type QueryParser struct {
	context *gin.Context
}

func NewQueryParser(c *gin.Context) QueryParser {
	parser := QueryParser{}
	parser.context = c

	return parser
}

func (p *QueryParser) OrderBy() string {
	return p.context.Query(OrderParam)
}

func (p *QueryParser) Match(s []string) bson.M {
	m := bson.M{}
	for i := range s {
		if q := p.context.Query(s[i]); q != "" {
			m[s[i]] = q
		}
	}

	if len(m) > 0 {
		return m
	}
	return nil
}

func (p *QueryParser) IContains(s []string) bson.M {
	m := bson.M{}

	p.context.Request.ParseForm()

	for i := range s {
		ic := s[i] + "__icontains"
		if ar := p.context.Request.Form[ic]; len(ar) > 0 {
			for j := range ar {
				m[s[i]] = bson.M{"$regex": ".*" + ar[j] + "*.", "$options": "i" }
			}
		}
	}

	if len(m) > 0 {
		return m
	}
	return nil
}