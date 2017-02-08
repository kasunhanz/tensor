package api

import (
	"gopkg.in/gin-gonic/gin.v1"
	"net/http"
	"github.com/pearsonappeng/tensor/queue"
)

// QueueStats returns statistics about redis rmq
func QueueStats(c *gin.Context)  {
	queues := queue.Queue.GetOpenQueues()
	stats := queue.Queue.CollectStats(queues)

	c.JSON(http.StatusOK, stats)
}