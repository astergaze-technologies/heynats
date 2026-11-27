package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/astergaze-solutions/heynats/internal/monitor"
	"github.com/gin-gonic/gin"
)

func monitorError(c *gin.Context, what string, err error) {
	switch {
	case errors.Is(err, monitor.ErrSystemOnly):
		c.JSON(http.StatusForbidden, gin.H{"error": "Requires a system account user"})
	case errors.Is(err, monitor.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read " + what, "details": err.Error()})
	}
}

func listServers(c *gin.Context) {
	conn, _ := GetNatsCredentialFromContext(c)
	list, err := conn.Monitor.Servers()
	if err != nil {
		monitorError(c, "servers", err)
		return
	}
	if conn.Metrics != nil {
		now := time.Now()
		in := conn.Metrics.Server.Latest("msgs_in", now)
		out := conn.Metrics.Server.Latest("msgs_out", now)
		for i, s := range list.Servers {
			if v, ok := in[s.Name]; ok && s.Responding {
				list.Servers[i].MsgsIn = &v
			}
			if v, ok := out[s.Name]; ok && s.Responding {
				list.Servers[i].MsgsOut = &v
			}
		}
	}
	c.JSON(http.StatusOK, list)
}

// serverRaw serves one server's monitoring data for one kind, e.g. ROUTEZ.
func serverRaw(c *gin.Context) {
	conn, _ := GetNatsCredentialFromContext(c)
	data, err := conn.Monitor.ServerRaw(c.Param("id"), c.Param("kind"))
	if err != nil {
		monitorError(c, "server data", err)
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}
