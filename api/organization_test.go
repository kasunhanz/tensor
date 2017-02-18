package api

import (
	"bytes"
	"encoding/json"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/jwt"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/validate"
	"github.com/stretchr/testify/suite"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"net/http/httptest"
	"testing"
)

type OrganizationApiTestSuite struct {
	suite.Suite
	authHeader     jwt.LocalToken
	server         *httptest.Server
	organizationID bson.ObjectId
}

func (suite *OrganizationApiTestSuite) SetupSuite() {
	if err := db.Connect(); err != nil {
		suite.Fail(err.Error(), "Unable to initialize a connection to database")
		return
	}

	// Define custom validator
	binding.Validator = &validate.Validator{}
	engine := gin.New()
	Route(engine)
	suite.server = httptest.NewServer(engine)

	if err := jwt.NewAuthToken(&suite.authHeader); err != nil {
		suite.Fail(err.Error(), "Unable to create auth header")
		return
	}
}

func (suite *OrganizationApiTestSuite) TestCreate() {
	body, err := json.Marshal(common.Organization{
		Name:        "organization",
		Description: "description",
	})

	if err != nil {
		suite.Fail(err.Error(), "Unable to create auth header")
		return
	}
	req, _ := http.NewRequest(http.MethodPost, suite.server.URL+"/v1/organizations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.authHeader.Token)

	// Test credential creation
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		suite.Fail(err.Error(), "Http request failed")
		return
	}

	suite.Equal(http.StatusCreated, r.StatusCode, "Response code should be Created, was: %s", r.StatusCode)
	suite.Equal("application/json; charset=utf-8", r.Header.Get("Content-Type"),
		"Content-Type should be application/json; charset=utf-8, was %s", r.Header.Get("Content-Type"))

	c := common.Credential{}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		suite.Fail(err.Error(), "Unmarshal must not fail")
		return
	}

	suite.organizationID = c.ID

	req2, _ := http.NewRequest(http.MethodPost, suite.server.URL+"/v1/organizations", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Accept", "application/json")
	req2.Header.Set("Authorization", "Bearer "+suite.authHeader.Token)

	// Create same credential must fail with 400
	r2, err := http.DefaultClient.Do(req2)
	if err != nil {
		suite.Fail(err.Error(), "Http request failed")
		return
	}

	suite.Equal(http.StatusBadRequest, r2.StatusCode, "Response code should be Bad Reqest, was: %s", r2.StatusCode)
	suite.Equal("application/json; charset=utf-8", r2.Header.Get("Content-Type"),
		"Content-Type should be application/json; charset=utf-8, was %s", r2.Header.Get("Content-Type"))
}

func (suite *OrganizationApiTestSuite) TestDelete() {

	req, _ := http.NewRequest(http.MethodDelete, suite.server.URL+"/v1/organizations/"+suite.organizationID.Hex(), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.authHeader.Token)

	// Test credential creation
	w, err := http.DefaultClient.Do(req)
	if err != nil {
		suite.Fail(err.Error(), "Http request failed")
		return
	}

	suite.Equal(http.StatusNoContent, w.StatusCode, "Response code should be No Content, was: %s", w.StatusCode)

	// same request must fail with 404
	w, err = http.DefaultClient.Do(req)
	if err != nil {
		suite.Fail(err.Error(), "Http request failed")
		return
	}

	suite.Equal(http.StatusNotFound, w.StatusCode, "Response code should be Not Found, was: %s", w.StatusCode)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestOrganizationApiTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationApiTestSuite))
}
