/*
Copyright 2017 Pearson, Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
 */
package main

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/api"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/exec/ansible"
	"github.com/pearsonappeng/tensor/exec/terraform"
	"github.com/pearsonappeng/tensor/log"
	"github.com/pearsonappeng/tensor/queue"
	"github.com/pearsonappeng/tensor/util"
	"github.com/pearsonappeng/tensor/validate"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
)

func main() {

	if util.Config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Infoln("Tensor:", util.Version)
	logrus.Infoln("Port:", util.Config.Port)
	logrus.Infoln("MongoDB:", util.Config.MongoDB.Username, util.Config.MongoDB.Hosts, util.Config.MongoDB.DbName)
	logrus.Infoln("Projects Home:", util.Config.ProjectsHome)

	if err := db.Connect(); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Fatalln("Unable to initialize a connection to database")
	}

	// connect to redis queues. this can panic if redis server not available make sure
	// the redis is up and running before running Tensor
	if err := queue.Connect(); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Fatalln("Unable to initialize a connection to redis")
	}

	defer func() {
		db.MongoDb.Session.Close()
	}()

	// Define custom validator
	binding.Validator = &validate.Validator{}
	r := gin.New()
	r.Use(log.Ginrus(logrus.StandardLogger(), time.RFC3339, true))
	r.Use(gin.Recovery())

	if util.Config.Debug {
		// automatically add routers for net/http/pprof
		// e.g. /debug/pprof, /debug/pprof/heap, etc.
		util.Wrapper(r)
	}

	api.Route(r)

	//Background tasks
	go ansible.Run()
	go terraform.Run()
	go queue.RMQCleaner()

	r.Run(util.Config.Port)
}
