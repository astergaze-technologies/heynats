package api

import (
	"log"
	"net/http"
	"time"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type HeyNats struct {
	router     *gin.RouterGroup
	conns      *NatsConnectionStore
	middleware *ConnectionMiddleware
}

func NewHeyNats(
	r *gin.RouterGroup,
	conns *NatsConnectionStore,
	middleware *ConnectionMiddleware,
) *HeyNats {
	return &HeyNats{
		router:     r,
		conns:      conns,
		middleware: middleware,
	}
}

func (e *HeyNats) RegisterRoutes() {
	// NATS connection endpoints
	api := e.router
	api.POST("/connect", func(c *gin.Context) {
		var req pkg.ConnectionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		oldID, _ := c.Cookie(ConnectionIDKey)
		if config, ok := e.conns.GetConfig(oldID); ok && *config == req {
			log.Println("Using existing connection for user:", oldID)
			e.conns.UpdateActivity(oldID)
			c.JSON(http.StatusOK, gin.H{
				"message":   "Successfully connected to NATS server",
				"connected": true,
			})
			return
		}

		natsConn := pkg.NewNATSCredential(&req)
		if err := natsConn.Connect(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Failed to connect to NATS server",
				"details": err.Error(),
			})
			return
		}

		if err := natsConn.TestConnection(); err != nil {
			natsConn.Disconnect()
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Connection test failed",
				"details": err.Error(),
			})
			return
		}

		e.conns.RemoveConnection(oldID)
		connectionID := uuid.New().String()
		e.conns.AddConnection(connectionID, natsConn, &req)
		setSessionCookie(c, connectionID, 3600*24)

		c.JSON(http.StatusOK, gin.H{
			"message":   "Successfully connected to NATS server",
			"connected": true,
		})
	})

	api.GET("/info", e.middleware.RequireConnection(), func(c *gin.Context) {
		natsConn, exists := c.Get(NatsConnectionKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "Not connected to NATS server",
				"connected": false,
			})
			return
		}

		conn := natsConn.(*pkg.NATSCredential)
		info, err := conn.GetInfo()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get NATS server info",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, info)
	})

	api.GET("/connection/stats", e.middleware.RequireConnection(), func(c *gin.Context) {
		connectionID, exists := c.Get(ConnectionIDKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Connection ID not found",
			})
			return
		}

		cID, ok := connectionID.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid connection ID format",
			})
			return
		}

		// Get connection info
		e.conns.mutex.RLock()
		connInfo, exists := e.conns.nastsConns[cID]
		e.conns.mutex.RUnlock()

		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Connection not found",
			})
			return
		}

		stats := gin.H{
			"connection_id":  cID,
			"last_activity":  connInfo.LastActivity,
			"idle_timeout":   e.conns.idleTimeout,
			"time_remaining": e.conns.idleTimeout - time.Since(connInfo.LastActivity),
			"is_healthy":     connInfo.Connection.IsHealthy(),
			"is_connected":   connInfo.Connection.Conn != nil && connInfo.Connection.Conn.IsConnected(),
		}

		c.JSON(http.StatusOK, stats)
	})

	api.GET("/account/info", e.middleware.RequireConnection(), func(c *gin.Context) {
		natsConn, exists := c.Get(NatsConnectionKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "Not connected to NATS server",
				"connected": false,
			})
			return
		}

		conn := natsConn.(*pkg.NATSCredential)
		accountInfo, err := conn.GetAccountInfo()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get account information",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, accountInfo)
	})

	api.GET("/account", e.middleware.RequireConnection(), func(c *gin.Context) {
		natsConn, exists := c.Get(NatsConnectionKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "Not connected to NATS server",
				"connected": false,
			})
			return
		}

		conn := natsConn.(*pkg.NATSCredential)
		accountInfo, err := conn.GetAccountInfo()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get account information",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, accountInfo)
	})

	// Dial with the given credentials, then close; no session is stored.
	api.POST("/test", func(c *gin.Context) {
		var req pkg.ConnectionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		conn := pkg.NewNATSCredential(&req)
		if err := conn.Connect(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Connection failed", "details": err.Error()})
			return
		}
		defer conn.Disconnect()
		if err := conn.TestConnection(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Connection test failed", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ok":          true,
			"server_name": conn.Conn.ConnectedServerName(),
			"version":     conn.Conn.ConnectedServerVersion(),
		})
	})

	api.POST("/disconnect", e.middleware.RequireConnection(), func(c *gin.Context) {
		connectionID, exists := c.Get(ConnectionIDKey)
		if exists {
			if cID, ok := connectionID.(string); ok {
				e.conns.RemoveConnection(cID)
				// Clear the cookie
				setSessionCookie(c, "", -1)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Disconnected from NATS server",
			"connected": false,
		})
	})

	api.GET("/status", e.middleware.Handle(), func(c *gin.Context) {
		natsConn, exists := c.Get(NatsConnectionKey)
		connected := false
		status := gin.H{
			"connected": connected,
		}

		if exists {
			conn := natsConn.(*pkg.NATSCredential)
			connected = conn != nil && conn.Conn != nil && conn.Conn.IsConnected()
			status["connected"] = connected

			if connected {
				status["host"] = conn.Host
				status["port"] = conn.Port
				status["username"] = conn.Username
			}
		}

		c.JSON(http.StatusOK, status)
	})

	api.GET("/health", e.middleware.Handle(), func(c *gin.Context) {
		natsConn, exists := c.Get(NatsConnectionKey)
		if !exists {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"connected": false,
			})
			return
		}

		conn := natsConn.(*pkg.NATSCredential)
		if conn == nil || conn.Conn == nil || !conn.Conn.IsConnected() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"connected": false,
			})
			return
		}

		// Perform a ping to check health
		if err := conn.Conn.Flush(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"connected": false,
				"error":     err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"connected": true,
		})
	})

}
