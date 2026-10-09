package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

type StreamAPI struct {
	router     *gin.RouterGroup
	conns      *NatsConnectionStore
	middleware *ConnectionMiddleware
}

func NewStreamAPI(
	router *gin.RouterGroup,
	conns *NatsConnectionStore,
	middleware *ConnectionMiddleware,
) *StreamAPI {
	return &StreamAPI{
		router:     router,
		conns:      conns,
		middleware: middleware,
	}
}

func (e *StreamAPI) RegisterRoutes() {
	api := e.router.Group("/streams")
	api.GET("", e.middleware.RequireConnection(), e.ListStreams)
	api.POST("", e.middleware.RequireConnection(), e.CreateStream)
	// More specific routes must come before generic :stream routes
	api.GET("/:stream/messages", e.middleware.RequireConnection(), e.GetStreamMessages)
	api.GET("/:stream/subjects/:subject/subscribe", e.middleware.RequireConnection(), e.SubscribeToStreamSubject)
	api.GET("/consumers/:stream", e.middleware.RequireConnection(), e.ListConsumers)
	// Generic routes at the end
	api.GET("/:stream", e.middleware.RequireConnection(), e.GetStreamInfo)
	api.DELETE("/:stream", e.middleware.RequireConnection(), e.DeleteStream)
}

// GetConnection retrieves the NATS connection from the context
func (e *StreamAPI) GetConnection(c *gin.Context) (*pkg.NATSCredential, bool) {
	natsConn, exists := c.Get(NatsConnectionKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":     "Not connected to NATS server",
			"connected": false,
		})
		return nil, false
	}

	conn := natsConn.(*pkg.NATSCredential)
	if !conn.IsConnected() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Not connected to NATS server",
		})
		return nil, false
	}

	return conn, true
}

// ListStreams handles GET /streams endpoint
func (e *StreamAPI) ListStreams(c *gin.Context) {
	conn, ok := e.GetConnection(c)
	if !ok {
		return
	}

	streams, err := conn.ListStreams()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list streams",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"streams": streams,
		"total":   len(streams),
	})
}

// GetStreamInfo handles GET /streams/:stream endpoint
func (e *StreamAPI) GetStreamInfo(c *gin.Context) {
	conn, ok := e.GetConnection(c)
	if !ok {
		return
	}

	stream := c.Param("stream")
	streamInfo, err := conn.GetStreamInfo(stream)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get stream info",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, streamInfo)
}

// ListConsumers handles GET /streams/consumers/:stream endpoint
func (e *StreamAPI) ListConsumers(c *gin.Context) {
	conn, ok := e.GetConnection(c)
	if !ok {
		return
	}

	stream := c.Param("stream")
	consumers, err := conn.ListConsumers(stream)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list consumers",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"consumers": consumers,
		"total":     len(consumers),
	})
}

// CreateStream handles POST /streams endpoint
func (e *StreamAPI) CreateStream(c *gin.Context) {
	conn, ok := e.GetConnection(c)
	if !ok {
		return
	}

	var config pkg.StreamConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid stream configuration",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if config.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Stream name is required",
		})
		return
	}

	if len(config.Subjects) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least one subject is required",
		})
		return
	}

	// Set defaults if not provided
	if config.NumReplicas == 0 {
		config.NumReplicas = 1
	}
	if config.Storage == "" {
		config.Storage = "file"
	}
	if config.Retention == "" {
		config.Retention = "limits"
	}
	if config.Discard == "" {
		config.Discard = "old"
	}

	// Create the stream
	streamInfo, err := conn.CreateStream(&config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create stream",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Stream created successfully",
		"stream_info": streamInfo,
	})
}

// SubscribeToStreamSubject handles GET /streams/:stream/subjects/:subject/subscribe endpoint
func (e *StreamAPI) SubscribeToStreamSubject(c *gin.Context) {
	streamName := c.Param("stream")
	subject := c.Param("subject")

	conn, ok := e.GetConnection(c)
	if !ok {
		return
	}

	err := streamSSE(c, sseSubscription{
		conn:    conn,
		subject: subject,
		connected: map[string]any{
			"type":    "connected",
			"subject": subject,
			"stream":  streamName,
		},
		event: func(msg *nats.Msg) map[string]any {
			return map[string]any{
				"subject":   subject,
				"data":      string(msg.Data),
				"timestamp": pkg.GetCurrentTimestamp(),
				"headers":   firstHeaderValues(msg.Header),
			}
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to subscribe to subject",
			"details": err.Error(),
		})
	}
}

// GetStreamMessages handles GET /streams/:stream/messages endpoint
func (e *StreamAPI) GetStreamMessages(c *gin.Context) {
	conn, ok := e.GetConnection(c)
	if !ok {
		return
	}

	streamName := c.Param("stream")

	// Get query parameters for pagination
	offset := 0
	limit := 10

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	// Get search parameter
	search := c.Query("search")

	// Get stream messages with search support
	var response *pkg.StreamMessagesResponse
	var err error

	if search != "" {
		response, err = conn.GetStreamMessagesWithSearch(streamName, offset, limit, search)
	} else {
		response, err = conn.GetStreamMessages(streamName, offset, limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get stream messages",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteStream handles DELETE /streams/:stream endpoint
func (e *StreamAPI) DeleteStream(c *gin.Context) {
	streamName := c.Param("stream")

	conn, ok := e.GetConnection(c)
	if !ok {
		return
	}

	err := conn.DeleteStream(streamName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete stream",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Stream '%s' deleted successfully", streamName),
	})
}
