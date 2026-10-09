package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go/jetstream"
)

type KVAPI struct {
	router     *gin.RouterGroup
	conns      *NatsConnectionStore
	middleware *ConnectionMiddleware
}

func NewKVAPI(
	router *gin.RouterGroup,
	conns *NatsConnectionStore,
	middleware *ConnectionMiddleware,
) *KVAPI {
	return &KVAPI{
		router:     router,
		conns:      conns,
		middleware: middleware,
	}
}

func (e *KVAPI) RegisterRoutes() {
	api := e.router.Group("/kv")

	// list all buckets
	api.GET("/buckets", e.middleware.RequireConnection(), e.ListBuckets)

	// get bucket detail
	api.GET("/buckets/:bucket", e.middleware.RequireConnection(), e.GetBucketDetail)

	// create bucket
	api.POST("/buckets", e.middleware.RequireConnection(), e.CreateBucket)

	// delete bucket
	api.DELETE("/buckets/:bucket", e.middleware.RequireConnection(), e.DeleteBucket)

	// get bucket keys
	api.GET("/buckets/:bucket/keys", e.middleware.RequireConnection(), e.GetBucketKeys)

	// put key value in a bucket
	api.PUT("/buckets/:bucket/keys/:key", e.middleware.RequireConnection(), e.PutKeyValue)

	// get key value
	api.GET("/buckets/:bucket/keys/:key", e.middleware.RequireConnection(), e.GetKeyValue)

	// delete key
	api.DELETE("/buckets/:bucket/keys/:key", e.middleware.RequireConnection(), e.DeleteKey)
}

// ListBuckets handles GET /buckets endpoint
func (e *KVAPI) ListBuckets(c *gin.Context) {
	conn, ok := GetNatsCredentialFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	manager := pkg.NewKVManager(conn)
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JetStream not initialized"})
		return
	}

	buckets, err := manager.ListBucketsWithStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch KV buckets", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"buckets": buckets})
}

// GetBucketDetail handles GET /buckets/:bucket endpoint
func (e *KVAPI) GetBucketDetail(c *gin.Context) {
	bucketName := c.Param("bucket")
	if bucketName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bucket name", "details": "Bucket name should be valid and non-empty"})
		return
	}

	conn, ok := GetNatsCredentialFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	manager := pkg.NewKVManager(conn)
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JetStream not initialized"})
		return
	}

	bucket, err := manager.GetBucket(bucketName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Failed to fetch KV bucket info",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, bucket)
}

// CreateBucket handles POST /buckets endpoint
func (e *KVAPI) CreateBucket(c *gin.Context) {
	conn, ok := GetNatsCredentialFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	manager := pkg.NewKVManager(conn)
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JetStream not initialized"})
		return
	}

	var req struct {
		Bucket  string `json:"bucket" binding:"required"`
		History int64  `json:"history"`       // Optional
		TTL     string `json:"ttl,omitempty"` // Optional, e.g., "60s", "5m"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.History < 0 || req.History > jetstream.KeyValueMaxHistory {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("history must be between 0 and %d", jetstream.KeyValueMaxHistory),
		})
		return
	}

	kvConfig := jetstream.KeyValueConfig{
		Bucket:  req.Bucket,
		History: uint8(req.History),
	}

	// Parse TTL string if provided
	if req.TTL != "" {
		ttl, err := time.ParseDuration(req.TTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid TTL format",
				"details": "Use valid Go duration strings like '60s', '5m', '1h30m'",
			})
			return
		}
		kvConfig.TTL = ttl
	}

	err := manager.CreateBucket(kvConfig)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, jetstream.ErrBucketExists) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{
			"error":   "Failed to create bucket",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Bucket created successfully",
		"bucket":  req.Bucket,
	})
}

// DeleteBucket handles DELETE /buckets/:bucket endpoint
func (e *KVAPI) DeleteBucket(c *gin.Context) {
	bucket := c.Param("bucket")
	conn, ok := GetNatsCredentialFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	manager := pkg.NewKVManager(conn)
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JetStream not initialized"})
		return
	}

	err := manager.DeleteBucket(bucket)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bucket", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Bucket deleted", "bucket": bucket})
}

// GetBucketKeys handles GET /buckets/:bucket/keys endpoint
func (e *KVAPI) GetBucketKeys(c *gin.Context) {
	bucket := c.Param("bucket")
	conn, ok := GetNatsCredentialFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	manager := pkg.NewKVManager(conn)
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JetStream not initialized"})
		return
	}

	// No pageSize returns every key; real paging comes with the KV workbench.
	page := 0
	pageSize := 0
	if p := c.Query("page"); p != "" {
		if pi, err := strconv.Atoi(p); err == nil && pi >= 0 {
			page = pi
		}
	}
	if ps := c.Query("pageSize"); ps != "" {
		if psi, err := strconv.Atoi(ps); err == nil && psi > 0 {
			pageSize = psi
		}
	}

	entries, err := manager.GetBucketKeys(bucket)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list keys in bucket",
			"details": err.Error(),
		})
		return
	}

	pagedEntries := entries
	if pageSize > 0 {
		start := min(page*pageSize, len(entries))
		end := min(start+pageSize, len(entries))
		pagedEntries = entries[start:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"bucket":   bucket,
		"page":     page,
		"pageSize": pageSize,
		"total":    len(entries),
		"items":    pagedEntries,
	})
}

// PutKeyValue handles POST /buckets/:bucket/keys endpoint
func (e *KVAPI) PutKeyValue(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if bucket == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing Bucket name"})
		return
	}

	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing Key"})
		return
	}

	conn, ok := GetNatsCredentialFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	manager := pkg.NewKVManager(conn)
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JetStream not initialized"})
		return
	}

	var req struct {
		Value json.RawMessage `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Strings are stored as-is; objects, numbers etc. as their raw JSON.
	value := []byte(req.Value)
	var str string
	if json.Unmarshal(req.Value, &str) == nil {
		value = []byte(str)
	}
	rev, err := manager.PutValue(bucket, key, value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to put value",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Value stored",
		"revision": rev,
	})
}

// GetKeyValue handles GET /buckets/:bucket/keys/:key endpoint
func (e *KVAPI) GetKeyValue(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")
	conn, ok := GetNatsCredentialFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	manager := pkg.NewKVManager(conn)
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JetStream not initialized"})
		return
	}

	val, err := manager.GetValue(bucket, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Key not found or failed", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "value": string(val)})
}

// DeleteKey handles DELETE /buckets/:bucket/keys/:key endpoint
func (e *KVAPI) DeleteKey(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")
	conn, ok := GetNatsCredentialFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not connected to NATS"})
		return
	}

	manager := pkg.NewKVManager(conn)
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JetStream not initialized"})
		return
	}

	err := manager.DeleteKey(bucket, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete key", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Key deleted", "key": key})
}
