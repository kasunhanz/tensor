package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin/binding"
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/controllers"
	"bitbucket.pearson.com/apseng/tensor/controllers/sockets"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/runners"
	"bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/crashy"
)

func main() {
	log.Infoln("Tensor :", util.Version)
	log.Infoln("Port :", util.Config.Port)
	log.Infoln("MongoDB :", util.Config.MongoDB.Username, util.Config.MongoDB.Hosts, util.Config.MongoDB.DbName)
	log.Infoln("Tmp Path (projects home) :", util.Config.TmpPath)

	if err := db.Connect(); err != nil {
		log.Fatalln(err)
	}

	defer func() {
		db.MongoDb.Session.Close()
	}()

	go sockets.StartWS()

	//Define custom validator
	binding.Validator = &util.SpaceValidator{}

	r := gin.New()
	r.Use(crashy.Recovery(recoveryHandler))

	controllers.Route(r)

	go runners.AnsiblePool.Run()
	go runners.SystemPool.Run()

	r.Run(util.Config.Port)
}

func recoveryHandler(c *gin.Context, err interface{}) {
	log.Errorln("Panic occurred", err)
	c.JSON(http.StatusInternalServerError, models.Error{
		Code: http.StatusInternalServerError,
		Messages: "You have not gotten any error messages recently," +
			" so here is a random one just to let you know that we haven't caring.",
	})
	c.Abort()
}