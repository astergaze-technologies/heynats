package infrastructure

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Router struct {
	*gin.Engine
}

// NewRouter serves the SPA from dist; unknown non-API paths fall back to index.html.
func NewRouter(dist fs.FS) *Router {
	r := gin.Default()
	// KV keys may contain "/"; the client sends it as %2F.
	r.UseRawPath = true
	r.UnescapePathValues = true
	fileServer := http.FileServer(http.FS(dist))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		if _, err := fs.Stat(dist, strings.TrimPrefix(path, "/")); err != nil {
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return &Router{r}
}
