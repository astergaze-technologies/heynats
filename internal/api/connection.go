package api

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/astergaze-solutions/heynats/internal/pkg"
)

type ConnectionInfo struct {
	Connection   *pkg.NATSCredential
	LastActivity time.Time
	Config       *pkg.ConnectionRequest // Store original connection config for reconnection
}

type ConnectionStoreConfig struct {
	IdleTimeout   time.Duration // How long to keep idle connections
	CheckInterval time.Duration // How often to check for idle connections
}

type NatsConnectionStore struct {
	nastsConns    map[string]*ConnectionInfo
	mutex         sync.RWMutex
	idleTimeout   time.Duration
	checkInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewNatsConnection() *NatsConnectionStore {
	return NewNatsConnectionWithConfig(ConnectionStoreConfig{
		IdleTimeout:   5 * time.Minute, // Close connections after 30 minutes of inactivity
		CheckInterval: 1 * time.Minute, // Check for idle connections every 5 minutes
	})
}

func NewNatsConnectionWithConfig(config ConnectionStoreConfig) *NatsConnectionStore {
	ctx, cancel := context.WithCancel(context.Background())
	store := &NatsConnectionStore{
		nastsConns:    make(map[string]*ConnectionInfo),
		idleTimeout:   config.IdleTimeout,
		checkInterval: config.CheckInterval,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start the idle connection cleanup goroutine
	go store.startIdleConnectionCleanup()

	log.Printf("Connection store initialized with idle timeout: %v, check interval: %v",
		config.IdleTimeout, config.CheckInterval)

	return store
}

// AddConnection adds a new connection with activity tracking
func (n *NatsConnectionStore) AddConnection(id string, conn *pkg.NATSCredential, config *pkg.ConnectionRequest) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.nastsConns[id] = &ConnectionInfo{
		Connection:   conn,
		LastActivity: time.Now(),
		Config:       config,
	}
}

// GetConfig returns the credentials a connection was dialed with.
func (n *NatsConnectionStore) GetConfig(id string) (*pkg.ConnectionRequest, bool) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	connInfo, exists := n.nastsConns[id]
	if !exists {
		return nil, false
	}
	return connInfo.Config, true
}

// GetConnection retrieves a connection and updates its activity timestamp
func (n *NatsConnectionStore) GetConnection(id string) (*pkg.NATSCredential, bool) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	connInfo, exists := n.nastsConns[id]
	if !exists {
		return nil, false
	}

	// Update last activity timestamp
	connInfo.LastActivity = time.Now()

	return connInfo.Connection, true
}

// GetOrReconnect never does network I/O while holding the lock. nats.go
// reconnects on its own, so only a closed connection is dialed again.
func (n *NatsConnectionStore) GetOrReconnect(id string) (*pkg.NATSCredential, bool, error) {
	n.mutex.Lock()
	connInfo, exists := n.nastsConns[id]
	if !exists {
		n.mutex.Unlock()
		return nil, false, nil
	}
	connInfo.LastActivity = time.Now()
	old := connInfo.Connection
	config := connInfo.Config
	n.mutex.Unlock()

	if old != nil && old.Conn != nil && !old.Conn.IsClosed() {
		return old, true, nil
	}

	log.Printf("Reconnecting to NATS server for connection %s", id)
	newConn := pkg.NewNATSCredential(config)
	err := newConn.Connect()

	n.mutex.Lock()
	defer n.mutex.Unlock()

	current, exists := n.nastsConns[id]
	if !exists || current.Connection != old {
		// Removed or already redialed by another request meanwhile.
		closeConn(newConn)
		if !exists {
			return nil, false, nil
		}
		return current.Connection, true, nil
	}

	if err != nil {
		closeConn(old)
		delete(n.nastsConns, id)
		return nil, false, err
	}

	closeConn(old)
	current.Connection = newConn
	log.Printf("Successfully reconnected to NATS server for connection %s", id)
	return newConn, true, nil
}

func closeConn(conn *pkg.NATSCredential) {
	if conn != nil {
		conn.Disconnect()
	}
}

// RemoveConnection removes and disconnects a specific connection
func (n *NatsConnectionStore) RemoveConnection(id string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if connInfo, exists := n.nastsConns[id]; exists {
		closeConn(connInfo.Connection)
		delete(n.nastsConns, id)
	}
}

// ClearAllConnections removes and disconnects all connections
func (n *NatsConnectionStore) ClearAllConnections() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	for id, connInfo := range n.nastsConns {
		if connInfo.Connection != nil {
			connInfo.Connection.Disconnect()
		}
		delete(n.nastsConns, id)
	}
}

// UpdateActivity updates the last activity timestamp for a connection
func (n *NatsConnectionStore) UpdateActivity(id string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if connInfo, exists := n.nastsConns[id]; exists {
		connInfo.LastActivity = time.Now()
	}
}

// startIdleConnectionCleanup starts a goroutine that periodically checks for and closes idle connections
func (n *NatsConnectionStore) startIdleConnectionCleanup() {
	ticker := time.NewTicker(n.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.cleanupIdleConnections()
		}
	}
}

// cleanupIdleConnections closes connections that have been idle for too long
func (n *NatsConnectionStore) cleanupIdleConnections() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	now := time.Now()
	toRemove := make([]string, 0)

	for id, connInfo := range n.nastsConns {
		if now.Sub(connInfo.LastActivity) > n.idleTimeout {
			log.Printf("Closing idle connection %s (idle for %v)", id, now.Sub(connInfo.LastActivity))
			if connInfo.Connection != nil {
				connInfo.Connection.Disconnect()
			}
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		delete(n.nastsConns, id)
	}

	if len(toRemove) > 0 {
		log.Printf("Cleaned up %d idle connections", len(toRemove))
	}
}

// Shutdown gracefully shuts down the connection store
func (n *NatsConnectionStore) Shutdown() {
	n.cancel()
	n.ClearAllConnections()
}
