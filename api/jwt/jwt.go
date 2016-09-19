package jwt

import (
	"time"
	"gopkg.in/dgrijalva/jwt-go.v3"
	"bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
	"log"
	"gopkg.in/mgo.v2/bson"
)

type LocalToken struct {
	Token  string
	Expire string
}

func NewAuthToken() (LocalToken, error) {
	// Initial middleware default setting.
	HeaderAuthMiddleware.MiddlewareInit()

	// Create the token
	token := jwt.New(jwt.GetSigningMethod(HeaderAuthMiddleware.SigningAlgorithm))
	claims := token.Claims.(jwt.MapClaims)

	collection := db.C(models.DBC_USERS)

	var admin models.User

	if err := collection.Find(bson.M{"username": "admin"}).One(&user); err != nil {
		log.Println("User not found, Create JWT Token faild")
		return nil, error("User not found, Create JWT Token faild")
	}

	expire := time.Now().Add(HeaderAuthMiddleware.Timeout)
	claims["id"] = admin.ID
	claims["exp"] = expire.Unix()
	claims["orig_iat"] = time.Now().Unix()

	tokenString, err := token.SignedString(HeaderAuthMiddleware.Key)

	if err != nil {
		log.Println("Create JWT Token faild")
		return nil, error("Create JWT Token faild")
	}

	return LocalToken{"token":  tokenString, "expire": expire.Format(time.RFC3339), }, nil
}
