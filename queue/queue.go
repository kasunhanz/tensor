package queue

import (
	"os"

	"github.com/adjust/uniuri"
	"github.com/gamunu/rmq"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/util"
)

const (
	// AnsibleQueue is the redis queue which stores jobs
	AnsibleQueue = "ansibleq"
	// TerraformQueue is the redis queue which stores jobs
	TerraformQueue = "terraformq"
)

// Queue is to hold the redis connection created by
// Connect function and make it available globally
var Queue *rmq.RedisConnection

// Connect creates a connection to Redis
func Connect() (err error) {
	hostname, e := os.Hostname()
	// if in case Hostname fails generate randon uniuri
	if e != nil {
		hostname = uniuri.New()
		log.WithFields(log.Fields{
			"hostname": hostname,
		}).Info("Coud not determine server hostname using random string for connection tag")
	}

	Queue, err = rmq.OpenConnection("tensor_"+hostname, "tcp", util.Config.Redis.Host, 2)
	return
}

// OpenAnsibleQueue returns rmq.Queue
func OpenAnsibleQueue() rmq.Queue {
	return Queue.OpenQueue(AnsibleQueue)
}

// OpenTerraformQueue returns rmq.Queue
func OpenTerraformQueue() rmq.Queue {
	return Queue.OpenQueue(TerraformQueue)
}
