package credentials

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/util/pagination"
	"bitbucket.pearson.com/apseng/tensor/util"
	"time"
	"strconv"
)

const _CTX_CREDENTIAL = "credential"
const _CTX_CREDENTIAL_ID = "credential_id"

func CredentialMiddleware(c *gin.Context) {

	ID := c.Params.ByName(_CTX_CREDENTIAL_ID)

	dbc := database.MongoDb.C(models.DBC_CREDENTIALS)

	var crd models.Credential

	if err := dbc.FindId(bson.ObjectIdHex(ID)).One(&crd); err != nil {
		log.Print(err) // log error to the system log
		c.AbortWithStatus(http.StatusNotFound) // send not found code if an error
		return
	}

	c.Set(_CTX_CREDENTIAL, crd)
	c.Next()
}

// GetProject returns the project as a JSON object
func GetCredential(c *gin.Context) {

	crd := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	hideEncrypted(&crd)
	setMetadata(&crd)

	c.JSON(200, crd)
}

func GetCredentials(c *gin.Context) {

	dbc := database.MongoDb.C(models.DBC_CREDENTIALS)

	parser := util.NewQueryParser(c)

	match := parser.Match([]string{"kind"})

	if con := parser.IContains([]string{"name", "username"}); con != nil {
		if match != nil {
			for i, v := range con {
				match[i] = v
			}
		} else {
			match = con
		}
	}

	query := dbc.Find(match)

	count, err := query.Count();
	if err != nil {
		log.Println("Unable to count credentials from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	pgi := pagination.NewPagination(c, count)

	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page) + ": That page contains no results."})
		return
	}

	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var crds []models.Credential

	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&crds); err != nil {
		log.Println("Unable to retrive credentials from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	for i, v := range crds {
		hideEncrypted(&v)
		if err := setMetadata(&v); err != nil {
			log.Println("Unable to set metadata", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		crds[i] = v
	}

	c.JSON(200, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": crds, })
}

func AddCredential(c *gin.Context) {
	u := c.MustGet("user").(models.User)

	var crd models.Credential

	if err := c.Bind(&crd); err != nil {
		log.Println("Failed to parse payload", err)
		c.JSON(http.StatusBadRequest,
			gin.H{"status": "Bad Request", "message": "Failed to parse payload"})
		return
	}

	crd.ID = bson.NewObjectId()
	crd.CreatedByID = u.ID
	crd.ModifiedByID = u.ID
	crd.Created = time.Now()
	crd.Modified = time.Now()
	crd.Password = "$encrypted$"
	crd.SshKeyData = "$encrypted$"
	crd.SshKeyUnlock = "$encrypted$"
	crd.BecomePassword = "$encrypted$"
	crd.VaultPassword = "$encrypted$"
	crd.AuthorizePassword = "$encrypted$"

	dbc := database.MongoDb.C(models.DBC_CREDENTIALS)
	dbacl := database.MongoDb.C(models.DBC_ACl)

	if err := dbc.Insert(crd); err != nil {
		log.Println("Failed to create Credential", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Credential"})
		return
	}

	if err := dbacl.Insert(models.ACL{ID:bson.NewObjectId(), Object:crd.ID, Type:"user", UserID:u.ID, Role: "admin"}); err != nil {
		log.Println("Failed to create acl", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create acl"})

		if err := dbc.RemoveId(crd.ID); err != nil {
			log.Println("Failed to remove credential", err)
		}

		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_CREDENTIAL,
		ObjectID:    crd.ID,
		Description: "Credential " + crd.Name + " created",
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	hideEncrypted(&crd)

	if err := setMetadata(&crd); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	c.JSON(http.StatusCreated, crd)
}

func UpdateCredential(c *gin.Context) {
	var req models.Credential

	crd := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	if err := c.Bind(&req); err != nil {
		return
	}

	//update necessary fields
	crd.Name = req.Name
	crd.Type = req.Type
	crd.Secret = req.Secret

	dbc := database.MongoDb.C(models.DBC_CREDENTIALS)

	if err := dbc.UpdateId(crd.ID, crd); err != nil {
		panic(err)
	}

	if err := (models.Event{
		Description: "Credential " + crd.Name + " updated",
		ObjectID:    crd.ID,
		ObjectType:  _CTX_CREDENTIAL,
	}.Insert()); err != nil {
		panic(err)
	}

	hideEncrypted(&crd)
	setMetadata(&crd)

	c.AbortWithStatus(204)
}

func RemoveCredential(c *gin.Context) {
	crd := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	dbc := database.MongoDb.C(models.DBC_CREDENTIALS)

	if err := dbc.RemoveId(crd.ID); err != nil {
		panic(err)
	}

	if err := (models.Event{
		Description: "Credential " + crd.Name + " deleted",
		ObjectID:    crd.ID,
		ObjectType:  _CTX_CREDENTIAL,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}