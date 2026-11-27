package api

import (
	"net/http"
	"strconv"

	"github.com/astergaze-solutions/heynats/internal/monitor"
	"github.com/gin-gonic/gin"
)

// listConnections serves ?state=open|closed|slow&sort=pending&offset=0&limit=500
// plus filters account, user, server, name, ip, lang.
func listConnections(c *gin.Context) {
	conn, _ := GetNatsCredentialFromContext(c)
	offset, err1 := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, err2 := strconv.Atoi(c.DefaultQuery("limit", "500"))
	if err1 != nil || err2 != nil || limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset and limit must be numbers, limit at most 1000"})
		return
	}
	q := monitor.ConnQuery{
		Sort: c.DefaultQuery("sort", "cid"), Offset: offset, Limit: limit,
		Account: c.Query("account"), User: c.Query("user"),
		Server: c.Query("server"), Name: c.Query("name"), IP: c.Query("ip"), Lang: c.Query("lang"),
	}
	switch c.DefaultQuery("state", "open") {
	case "open":
	case "closed":
		q.State = monitor.Closed
	case "slow":
		q.State, q.SlowOnly = monitor.Closed, true
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "state must be open, closed or slow"})
		return
	}
	page, err := conn.Monitor.Connections(q)
	if err != nil {
		monitorError(c, "connections", err)
		return
	}
	c.JSON(http.StatusOK, page)
}

// kickConnection disconnects a client; most clients reconnect on their own.
func kickConnection(c *gin.Context) {
	conn, _ := GetNatsCredentialFromContext(c)
	cid, err := strconv.ParseUint(c.Param("cid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cid must be a number"})
		return
	}
	if err := conn.Monitor.Kick(c.Param("server"), cid); err != nil {
		monitorError(c, "connection", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"kicked": true})
}

func getConnection(c *gin.Context) {
	conn, _ := GetNatsCredentialFromContext(c)
	cid, err := strconv.ParseUint(c.Param("cid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cid must be a number"})
		return
	}
	detail, err := conn.Monitor.Connection(c.Param("server"), cid)
	if err != nil {
		monitorError(c, "connection", err)
		return
	}
	c.JSON(http.StatusOK, detail)
}
