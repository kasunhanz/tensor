package queue

import (
	"github.com/gamunu/rmq"
	"time"
)

// RMQCleaner runs regularly to return unacked deliveries of stopped
// or crashed consumers back to ready so they can be consumed by a new consumer
func RMQCleaner() {
	cleaner := rmq.NewCleaner(Queue)

	//TODO: add this to configuration
	for _ = range time.Tick(time.Second * time.Duration(15)) {
		cleaner.Clean()
	}
}
