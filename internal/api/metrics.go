package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// getMetrics serves ?series=msgs_in,msgs_out&range=1h as chart-ready columns.
// Server and JetStream series are sampled on different clocks, so one request
// takes series from one of the two only.
func getMetrics(c *gin.Context) {
	conn, _ := GetNatsCredentialFromContext(c)
	if conn.Metrics == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Metrics sampling is turned off"})
		return
	}
	span, err := time.ParseDuration(c.DefaultQuery("range", "1h"))
	if err != nil || span <= 0 || span > 24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "range must be a duration up to 24h, e.g. 15m or 6h"})
		return
	}
	names := strings.Split(c.Query("series"), ",")
	js := isJetStreamSeries(names[0])
	for _, n := range names {
		if n == "" || isJetStreamSeries(n) != js {
			c.JSON(http.StatusBadRequest, gin.H{"error": "series must be non-empty and all server or all JetStream (js_*, consumer_*)"})
			return
		}
	}
	h := conn.Metrics.Server
	if js {
		h = conn.Metrics.JetStream
	}
	w := h.Window(names, span, time.Now())
	w.Source = string(conn.Monitor.Source())
	c.JSON(http.StatusOK, w)
}

func isJetStreamSeries(name string) bool {
	return strings.HasPrefix(name, "js_") || strings.HasPrefix(name, "consumer_")
}
