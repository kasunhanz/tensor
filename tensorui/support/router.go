package support

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Declare all routes
func Route(r *gin.Engine) {
	r.NoRoute(servePublic)
}
func servePublic(c *gin.Context) {
	path := c.Request.URL.Path

	if !strings.HasPrefix(path, "/public") {
		if len(strings.Split(path, ".")) > 1 {
			c.AbortWithStatus(404)
			return
		}

		path = "/public/html/index.html"
	}

	path = strings.Replace(path, "/", "", 1)
	split := strings.Split(path, ".")
	suffix := split[len(split) - 1]

	res, err := Asset(path)
	if err != nil {
		c.Next()
		return
	}

	contentType := "text/plain"
	switch suffix {
	case "png":
		contentType = "image/png"
	case "jpg", "jpeg":
		contentType = "image/jpeg"
	case "gif":
		contentType = "image/gif"
	case "js":
		contentType = "application/javascript"
	case "css":
		contentType = "text/css"
	case "woff":
		contentType = "application/x-font-woff"
	case "ttf":
		contentType = "application/x-font-ttf"
	case "otf":
		contentType = "application/x-font-otf"
	case "html":
		contentType = "text/html"
	}

	c.Writer.Header().Set("content-type", contentType)
	c.String(200, string(res))
}