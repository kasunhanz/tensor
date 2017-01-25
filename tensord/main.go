package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pearsonappeng/tensor/api"
	"github.com/pearsonappeng/tensor/api/sockets"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/queue"
	"github.com/pearsonappeng/tensor/runners/ansible"
	"github.com/pearsonappeng/tensor/util"
)

func main() {
	log.Infoln("Tensor :", util.Version)
	log.Infoln("Port :", util.Config.Port)
	log.Infoln("MongoDB :", util.Config.MongoDB.Username, util.Config.MongoDB.Hosts, util.Config.MongoDB.DbName)
	log.Infoln("Projects Home:", util.Config.ProjectsHome)

	if err := db.Connect(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Fatalln("Unable to initialize a connection to the database")
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

	go ansible.Run()

	r.Run(util.Config.Port)
}
