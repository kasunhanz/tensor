package gzip

import (
	"compress/gzip"
	"github.com/gin-gonic/gin"
)

const (
	BestCompression = gzip.BestCompression
	BestSpeed = gzip.BestSpeed
	DefaultCompression = gzip.DefaultCompression
	NoCompression = gzip.NoCompression
)

func Gzip(level int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// no file path check since this is an API
		gz, err := gzip.NewWriterLevel(c.Writer, level)
		if err != nil {
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		c.Writer = &gzipWriter{c.Writer, gz}
		defer func() {
			c.Header("Content-Length", "")
			gz.Close()
		}()
		c.Next()
	}
}

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}