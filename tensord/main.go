package main

import (
	"fmt"
	"log"
	"github.com/gin-gonic/gin/binding"
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/api"
	"bitbucket.pearson.com/apseng/tensor/api/sockets"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/runners"
	"bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/crashy"
)

func main() {
	fmt.Printf("Tensor : %v\n", util.Version)
	fmt.Printf("Port : %v\n", util.Config.Port)
	fmt.Printf("MongoDB : %v@%v %v\n", util.Config.MongoDB.Username, util.Config.MongoDB.Hosts, util.Config.MongoDB.DbName)
	fmt.Printf("Tmp Path (projects home) : %v\n", util.Config.TmpPath)

	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}

	defer func() {
		db.MongoDb.Session.Close()
	}()

	go sockets.StartWS()

	//Define custom validator
	binding.Validator = &util.SpaceValidator{}

	r := gin.New()
	r.Use(gin.Recovery(), recovery, gin.Logger(), crashy.Recovery(recoveryHandler))

	api.Route(r)

	go runners.StartAnsibleRunner()
	go runners.StartSystemRunner()

	r.Run(util.Config.Port)

}

func recovery(c *gin.Context) {

	//report to bug nofiy system
	c.Next()
}

func recoveryHandler(c *gin.Context, err interface{}) {
	c.JSON(http.StatusInternalServerError, models.Error{
		Code: http.StatusInternalServerError,
		Messages: "You have not gotten any error messages recently," +
			" so here is a random one just to let you know that we haven't caring.",
	})
	c.Abort()
}