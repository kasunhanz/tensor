package queue

import (
	"os"

	"github.com/adjust/uniuri"
	"github.com/gamunu/rmq"

	"bitbucket.pearson.com/apseng/tensor/util"
	log "github.com/Sirupsen/logrus"
)

const (
	// JobQueue is the redis queue which stores jobs
	JobQueue = "sysqueue"
)

// Queue is to hold the redis connection created by
// Connect function and make it availble globally
var Queue *rmq.RedisConnection

// Connect creates a connection to Redis
func Connect() {
	hostname, e := os.Hostname()
	// if in case Hostname fails generate randon uniuri
	if e != nil {
		hostname = uniuri.New()
		log.WithFields(log.Fields{
			"hostname": hostname,
		}).Info("Coud not determine server hostname using random string for connection tag")
	}

	Queue = rmq.OpenConnection("tensor_"+hostname, "tcp", util.Config.Redis.Host, 2)
}

// OpenJobQueue returns rmq.Queue
func OpenJobQueue() rmq.Queue {
	return Queue.OpenQueue(JobQueue)
}
