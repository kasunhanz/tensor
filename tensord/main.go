package main

import (
	"time"

	"github.com/gamunu/tensor/api"
	"github.com/gamunu/tensor/api/sockets"
	"github.com/gamunu/tensor/db"
	"github.com/gamunu/tensor/queue"
	"github.com/gamunu/tensor/runners"
	"github.com/gamunu/tensor/util"
	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func main() {
	log.Infoln("Tensor :", util.Version)
	log.Infoln("Port :", util.Config.Port)
	log.Infoln("MongoDB :", util.Config.MongoDB.Username, util.Config.MongoDB.Hosts, util.Config.MongoDB.DbName)
	log.Infoln("Projects Home:", util.Config.ProjectsHome)

	if err := db.Connect(); err != nil {
		log.Fatalln(err)
	}

	// connect to redis queues. this can panic if redis server not available make sure
	// the redis is up and running before running Tensor
	queue.Connect()

	defer func() {
		db.MongoDb.Session.Close()
	}()

	go sockets.StartWS()

	//Define custom validator
	binding.Validator = &util.SpaceValidator{}

	r := gin.New()
	r.Use(ginrus.Ginrus(log.StandardLogger(), time.RFC3339, true))
	r.Use(gin.Recovery())

	controllers.Route(r)

	go runners.AnsibleRun()

	r.Run(util.Config.Port)
}
