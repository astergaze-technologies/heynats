package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

type PublishAPI struct {
	router     *gin.RouterGroup
	conns      *NatsConnectionStore
	middleware *ConnectionMiddleware
}

type PublishRequest struct {
	Subject string            `json:"subject" binding:"required"`
	Data    string            `json:"data"`
	Headers map[string]string `json:"headers,omitempty"`
}

type BatchPublishRequest struct {
	Messages []PublishMessage `json:"messages" binding:"required"`
}

type PublishMessage struct {
	Subject string            `json:"subject" binding:"required"`
	Data    string            `json:"data"`
	Headers map[string]string `json:"headers,omitempty"`
}

type PublishResponse struct {
	Success   bool   `json:"success"`
	Subject   string `json:"subject"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

type BatchPublishResponse struct {
	Success   bool              `json:"success"`
	Results   []PublishResponse `json:"results"`
	Total     int               `json:"total"`
	Succeeded int               `json:"succeeded"`
	Failed    int               `json:"failed"`
}

type RequestReplyRequest struct {
	Subject   string            `json:"subject" binding:"required"`
	Data      string            `json:"data"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timeout   int               `json:"timeout,omitempty"` // in seconds, default 5
	ReplySubj string            `json:"reply_subject,omitempty"`
}

type RequestReplyResponse struct {
	Success     bool   `json:"success"`
	Subject     string `json:"subject"`
	ReplySubj   string `json:"reply_subject"`
	RequestData string `json:"request_data"`
	ReplyData   string `json:"reply_data,omitempty"`
	Error       string `json:"error,omitempty"`
	Timeout     bool   `json:"timeout,omitempty"`
}

func NewPublishAPI(
	router *gin.RouterGroup,
	conns *NatsConnectionStore,
	middleware *ConnectionMiddleware,
) *PublishAPI {
	return &PublishAPI{
		router:     router,
		conns:      conns,
		middleware: middleware,
	}
}

func (p *PublishAPI) RegisterRoutes() {
	api := p.router.Group("/publish")

	// Single message publishing
	api.POST("/message", p.middleware.RequireConnection(), p.publishMessage)

	// Batch publishing
	api.POST("/batch", p.middleware.RequireConnection(), p.publishBatch)

	// Request-Reply pattern
	api.POST("/request", p.middleware.RequireConnection(), p.requestReply)

	// Get subjects (for autocomplete/suggestions)
	api.GET("/subjects", p.middleware.RequireConnection(), p.getSubjects)
}

func (p *PublishAPI) publishMessage(c *gin.Context) {
	natsConn, exists := c.Get(NatsConnectionKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	conn := natsConn.(*pkg.NATSCredential)

	// Create message
	msg := &nats.Msg{
		Subject: req.Subject,
		Data:    []byte(req.Data),
	}

	// Add headers if provided
	if len(req.Headers) > 0 {
		msg.Header = make(nats.Header)
		for key, value := range req.Headers {
			msg.Header.Set(key, value)
		}
	}

	// Publish message
	err := conn.Conn.PublishMsg(msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to publish message: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, PublishResponse{
		Success: true,
		Subject: req.Subject,
	})
}

func (p *PublishAPI) publishBatch(c *gin.Context) {
	natsConn, exists := c.Get(NatsConnectionKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	var req BatchPublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	conn := natsConn.(*pkg.NATSCredential)

	var results []PublishResponse
	succeeded := 0
	failed := 0

	for _, msg := range req.Messages {
		natsMsg := &nats.Msg{
			Subject: msg.Subject,
			Data:    []byte(msg.Data),
		}

		// Add headers if provided
		if len(msg.Headers) > 0 {
			natsMsg.Header = make(nats.Header)
			for key, value := range msg.Headers {
				natsMsg.Header.Set(key, value)
			}
		}

		// Publish message
		err := conn.Conn.PublishMsg(natsMsg)
		if err != nil {
			results = append(results, PublishResponse{
				Success: false,
				Subject: msg.Subject,
				Error:   fmt.Sprintf("Failed to publish: %v", err),
			})
			failed++
		} else {
			results = append(results, PublishResponse{
				Success: true,
				Subject: msg.Subject,
			})
			succeeded++
		}
	}

	c.JSON(http.StatusOK, BatchPublishResponse{
		Success:   failed == 0,
		Results:   results,
		Total:     len(req.Messages),
		Succeeded: succeeded,
		Failed:    failed,
	})
}

func (p *PublishAPI) requestReply(c *gin.Context) {
	natsConn, exists := c.Get(NatsConnectionKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	var req RequestReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	conn := natsConn.(*pkg.NATSCredential)

	// Set default timeout if not provided
	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	// Create message
	msg := &nats.Msg{
		Subject: req.Subject,
		Data:    []byte(req.Data),
	}

	// Add headers if provided
	if len(req.Headers) > 0 {
		msg.Header = make(nats.Header)
		for key, value := range req.Headers {
			msg.Header.Set(key, value)
		}
	}

	reply, err := sendRequest(conn.Conn, msg, req.ReplySubj, timeout)
	if err != nil {
		c.JSON(requestErrorStatus(err), RequestReplyResponse{
			Success:     false,
			Subject:     req.Subject,
			ReplySubj:   req.ReplySubj,
			RequestData: req.Data,
			Error:       fmt.Sprintf("Request failed: %v", err),
			Timeout:     errors.Is(err, nats.ErrTimeout),
		})
		return
	}

	c.JSON(http.StatusOK, RequestReplyResponse{
		Success:     true,
		Subject:     req.Subject,
		ReplySubj:   reply.Subject,
		RequestData: req.Data,
		ReplyData:   string(reply.Data),
	})
}

// sendRequest uses a custom reply subject when given, otherwise a NATS inbox.
func sendRequest(nc *nats.Conn, msg *nats.Msg, replySubj string, timeout time.Duration) (*nats.Msg, error) {
	if replySubj == "" {
		return nc.RequestMsg(msg, timeout)
	}
	sub, err := nc.SubscribeSync(replySubj)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sub.Unsubscribe() }()

	msg.Reply = replySubj
	if err := nc.PublishMsg(msg); err != nil {
		return nil, err
	}
	reply, err := sub.NextMsg(timeout)
	if err != nil {
		return nil, err
	}
	if reply.Header.Get("Status") == "503" && len(reply.Data) == 0 {
		return nil, nats.ErrNoResponders
	}
	return reply, nil
}

func requestErrorStatus(err error) int {
	switch {
	case errors.Is(err, nats.ErrTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, nats.ErrNoResponders):
		return http.StatusServiceUnavailable
	default:
		return http.StatusBadGateway
	}
}

func (p *PublishAPI) getSubjects(c *gin.Context) {
	natsConn, exists := c.Get(NatsConnectionKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	conn := natsConn.(*pkg.NATSCredential)

	// Get subjects from streams (if JetStream is available)
	subjects := make([]string, 0)

	if conn.JSConn != nil {
		streams, err := conn.ListStreams()
		if err == nil {
			for _, stream := range streams {
				for _, subject := range stream.Config.Subjects {
					subjects = append(subjects, subject)
				}
			}
		}
	}

	// Add some common subject patterns for suggestions
	commonSubjects := []string{
		"events.user.*",
		"logs.*",
		"metrics.*",
		"notifications.*",
		"system.*",
	}

	subjects = append(subjects, commonSubjects...)

	c.JSON(http.StatusOK, gin.H{
		"subjects": subjects,
	})
}
