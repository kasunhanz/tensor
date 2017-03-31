package api

import (
	"github.com/pearsonappeng/tensor/queue"
	"github.com/gin-gonic/gin"
	"net/http"
)

// QueueStats returns statistics about redis rmq
func QueueStats(c *gin.Context) {
	queues := queue.Queue.GetOpenQueues()
	stats := queue.Queue.CollectStats(queues)

	c.JSON(http.StatusOK, stats)
}
