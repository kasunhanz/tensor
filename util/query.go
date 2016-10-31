package util

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"net/url"
	"strings"
	"log"
)

var (
	OrderParam = "order_by"
)

type QueryParser struct {
	context *gin.Context
	From    url.Values
}

func NewQueryParser(c *gin.Context) QueryParser {
	parser := QueryParser{}
	parser.context = c

	// parses the raw query from the URL and updates r.Form
	parser.context.Request.ParseForm()
	// copy to a local form
	parser.From = parser.context.Request.Form

	return parser
}

func (p *QueryParser) OrderBy() string {
	return p.context.Query(OrderParam)
}

func (p *QueryParser) Match(s []string, query bson.M) bson.M {
	for i := range s {
		if q := p.context.Query(s[i]); q != "" {
			query[s[i]] = q
		}
	}
	return query
}

func (p *QueryParser) Lookups(fields []string, query bson.M) bson.M {

	query = p.Exact(fields, query)
	query = p.IExact(fields, query)
	query = p.Contains(fields, query)
	query = p.IContains(fields, query)
	query = p.Startswith(fields, query)
	query = p.IStartswith(fields, query)
	query = p.Endswith(fields, query)
	query = p.IEndswith(fields, query)
	query = p.Gt(fields, query)
	query = p.Gte(fields, query)
	query = p.Lt(fields, query)
	query = p.Lte(fields, query)
	query = p.IsNull(fields, query)
	query = p.In(fields, query)
	query = p.Eq(fields, query)
	query = p.Ne(fields, query)
	query = p.Gte(fields, query)

	log.Println(query)
	return query
}

// Field lookups
// ------------------------

// Exact adds regex to mgo query to check a field is an exact match
// accepting fields must pass to fields parameter
func (p *QueryParser) Exact(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__exact"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$regex": bson.RegEx{"/^" + ar[j] + "$/", ""} }
			}
		}
	}
	return query
}

// Exact adds regex to mgo query to check a field is an exact match
// this is the case insensitive version of Exact
// accepting fields must pass to fields parameter
func (p *QueryParser) IExact(s []string, query bson.M) bson.M {
	for i := range s {
		// avoid hacks by using defined parameters
		ic := s[i] + "__iexact"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[s[i]] = bson.M{"$regex": bson.RegEx{"/^" + ar[j] + "$/", "i"} }
			}
		}
	}
	return query
}


// Contains adds regex to mgo query to check a field contain a value
// accepting fields must pass to fields parameter
func (p *QueryParser) Contains(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__contains"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$regex": bson.RegEx{"/.*" + ar[j] + ".*/", ""} }
			}
		}
	}
	return query
}

// IContains adds regex to mgo query to check a field contain a value
// this is the case insensitive version of contains
// accepting fields must pass to fields parameter
func (p *QueryParser) IContains(s []string, query bson.M) bson.M {
	for i := range s {
		// avoid hacks by using defined parameters
		ic := s[i] + "__icontains"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[s[i]] = bson.M{"$regex": bson.RegEx{"/.*" + ar[j] + ".*/", "i"} }
			}
		}
	}
	return query
}

// Startswith adds regex to mgo query to check a field startswith a value
// accepting fields must pass to fields parameter
func (p *QueryParser) Startswith(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__startswith"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$regex": bson.RegEx{"/^" + ar[j] + "/", ""} }
			}
		}
	}
	return query
}

// IStartswith adds regex to mgo query to check a field startswith a value
// this is the case insensitive version of startswith
// accepting fields must pass to fields parameter
func (p *QueryParser) IStartswith(s []string, query bson.M) bson.M {
	for i := range s {
		// avoid hacks by using defined parameters
		ic := s[i] + "__istartswith"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[s[i]] = bson.M{"$regex": bson.RegEx{"/^" + ar[j] + "/", "i"} }
			}
		}
	}
	return query
}

// Endswith adds regex to mgo query to check a field endswith a value
// accepting fields must pass to fields parameter
func (p *QueryParser) Endswith(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__endswith"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$regex": bson.RegEx{"/^" + ar[j] + "$/", ""} }
			}
		}
	}
	return query
}

// IEndswith adds regex to mgo query to check a field endswith a value
// this is the case insensitive version of endswith
// accepting fields must pass to fields parameter
func (p *QueryParser) IEndswith(s []string, query bson.M) bson.M {
	for i := range s {
		// avoid hacks by using defined parameters
		ic := s[i] + "__iendswith"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[s[i]] = bson.M{"$regex": bson.RegEx{"/" + ar[j] + "$/", "i"} }
			}
		}
	}
	return query
}


// Gt adds $gt to mgo query to check a field greater than the comparison value
// accepting fields must pass to fields parameter
func (p *QueryParser) Gt(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__gt"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$gt": ar[j] }
			}
		}
	}
	return query
}

// Gte adds $gte to mgo query to check a field greater or equal to the comparison value
// accepting fields must pass to fields parameter
func (p *QueryParser) Gte(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__gte"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$gte": ar[j] }
			}
		}
	}
	return query
}

// Lt adds $lt to mgo query to check a field less than the comparison value
// accepting fields must pass to fields parameter
func (p *QueryParser) Lt(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__lt"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$lt": ar[j] }
			}
		}
	}
	return query
}

// Gt adds $lte to mgo query to check a field less than  or equal to the comparison value
// accepting fields must pass to fields parameter
func (p *QueryParser) Lte(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__lte"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$lte": ar[j] }
			}
		}
	}
	return query
}

// IsNull check the given field or related object is null
func (p *QueryParser) IsNull(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__isnull"
		if ar := p.From[ic]; len(ar) > 0 {
			for range ar {
				query[fields[i]] = bson.M{"$lte": nil }
			}
		}
	}
	return query
}

// In Check the given field's value is present in the list provide
// accepting fields must pass to fields parameter
func (p *QueryParser) In(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__in"
		// if [ field__in=value ] available in form
		if ar := p.From[ic]; len(ar) > 0 {
			// loop though all in operators
			for j := range ar {
				// split string by `,`
				values := strings.Split(ar[j], ",")
				if len(values) > 0 {
					query[fields[i]] = bson.M{"$in": values }
				}
			}
		}
	}
	return query
}


// Eq check the given field or related object is equel to given value
func (p *QueryParser) Eq(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__eq"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$eq": ar[j] }
			}
		}
	}
	return query
}

// Eq check the given field or related object is not equal to given value
func (p *QueryParser) Ne(fields []string, query bson.M) bson.M {
	for i := range fields {
		// avoid hacks by using defined parameters
		ic := fields[i] + "__ne"
		if ar := p.From[ic]; len(ar) > 0 {
			for j := range ar {
				query[fields[i]] = bson.M{"$ne": ar[j] }
			}
		}
	}
	return query
}