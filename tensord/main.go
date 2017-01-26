package main

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/api"
	"github.com/pearsonappeng/tensor/api/sockets"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/log"
	"github.com/pearsonappeng/tensor/queue"
	"github.com/pearsonappeng/tensor/runners/ansible"
	"github.com/pearsonappeng/tensor/runners/terraform"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
)

func main() {
	logrus.Infoln("Tensor :", util.Version)
	logrus.Infoln("Port :", util.Config.Port)
	logrus.Infoln("MongoDB :", util.Config.MongoDB.Username, util.Config.MongoDB.Hosts, util.Config.MongoDB.DbName)
	logrus.Infoln("Projects Home:", util.Config.ProjectsHome)

	if err := db.Connect(); err != nil {
		logrus.WithFields(logrus.Fields{
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
	r.Use(log.Ginrus(logrus.StandardLogger(), time.RFC3339, true))
	r.Use(gin.Recovery())

	controllers.Route(r)

	go ansible.Run()
	go terraform.Run()

	r.Run(util.Config.Port)
}
