package main

import (
	"fmt"
	"log"
	"github.com/gin-gonic/gin/binding"
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/api"
	"bitbucket.pearson.com/apseng/tensor/api/sockets"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/util"
)

func main() {
	fmt.Printf("Tensor : %v\n", util.Version)
	fmt.Printf("Port : %v\n", util.Config.Port)
	fmt.Printf("MongoDB : %v@%v %v\n", util.Config.MongoDB.Username, util.Config.MongoDB.Hosts, util.Config.MongoDB.DbName)
	fmt.Printf("Tmp Path (projects home) : %v\n", util.Config.TmpPath)

	if err := database.Connect(); err != nil {
		log.Fatal(err)
	}

	defer func() {
		database.MongoDb.Session.Close()
	}()

	go sockets.StartWS()

	//Define custom validator
	binding.Validator = &util.SpaceValidator{}

	r := gin.New()
	r.Use(gin.Recovery(), recovery, gin.Logger())

	api.Route(r)

	//go tasks.StartRunner()
	//go addhoctasks.StartRunner()

	r.Run(util.Config.Port)

}

func recovery(c *gin.Context) {

	//report to bug nofiy system
	c.Next()
}