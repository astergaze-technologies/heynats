package pkg

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/astergaze-solutions/heynats/internal/util"
	"github.com/dustin/go-humanize"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
)

type NATSCredential struct {
	Conn      *nats.Conn
	JSConn    *nats.JetStreamContext
	JetStream *jetstream.JetStream
	Host      string `json:"host"`
	Port      string `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Token     string `json:"-"`
	NKeySeed  string `json:"-"`
	Creds     string `json:"-"`
}

type ConnectionRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     string `json:"port" binding:"required"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
	NKeySeed string `json:"nkeySeed"`
	Creds    string `json:"creds"` // contents of a .creds file (JWT + NKey seed)
}

func NewNATSCredential(req *ConnectionRequest) *NATSCredential {
	return &NATSCredential{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Token:    req.Token,
		NKeySeed: req.NKeySeed,
		Creds:    req.Creds,
	}
}

// authOption picks one auth method; precedence: creds > nkey > token > user/pass.
func (nc *NATSCredential) authOption() (nats.Option, error) {
	switch {
	case nc.Creds != "":
		jwt, err := nkeys.ParseDecoratedJWT([]byte(nc.Creds))
		if err != nil {
			return nil, fmt.Errorf("invalid creds JWT: %w", err)
		}
		kp, err := nkeys.ParseDecoratedUserNKey([]byte(nc.Creds))
		if err != nil {
			return nil, fmt.Errorf("invalid creds seed: %w", err)
		}
		seed, err := kp.Seed()
		if err != nil {
			return nil, fmt.Errorf("invalid creds seed: %w", err)
		}
		return nats.UserJWTAndSeed(jwt, string(seed)), nil
	case nc.NKeySeed != "":
		kp, err := nkeys.FromSeed([]byte(strings.TrimSpace(nc.NKeySeed)))
		if err != nil {
			return nil, fmt.Errorf("invalid nkey seed: %w", err)
		}
		pub, err := kp.PublicKey()
		if err != nil {
			return nil, fmt.Errorf("invalid nkey seed: %w", err)
		}
		return nats.Nkey(pub, kp.Sign), nil
	case nc.Token != "":
		return nats.Token(nc.Token), nil
	case nc.Username != "" && nc.Password != "":
		return nats.UserInfo(nc.Username, nc.Password), nil
	}
	return nil, nil
}

type NATSInfo struct {
	ServerInfo   map[string]interface{} `json:"server_info"`
	Stats        nats.Statistics        `json:"stats"`
	IsConnected  bool                   `json:"is_connected"`
	ConnectedURL string                 `json:"connected_url"`
}

type AccountInfo struct {
	AccountInformation map[string]interface{} `json:"account_information"`
	ConnectionLimits   map[string]interface{} `json:"connection_limits"`
	Stats              map[string]interface{} `json:"stats"`
	JetStreamInfo      *JSAccountInfo         `json:"jetstream_info,omitempty"`
}

type JSAccountInfo struct {
	Memory    uint64 `json:"memory"`
	Store     uint64 `json:"store"`
	Streams   int    `json:"streams"`
	Consumers int    `json:"consumers"`
}

func (nc *NATSCredential) Connect() error {
	var opts []nats.Option

	// Build connection URL
	url := fmt.Sprintf("nats://%s:%s", nc.Host, nc.Port)

	// Add authentication if provided
	auth, err := nc.authOption()
	if err != nil {
		return err
	}
	if auth != nil {
		opts = append(opts, auth)
	}

	// Add connection options with better reconnection handling
	opts = append(opts,
		nats.Name("HeyNATS Web Client"),
		nats.Timeout(10*time.Second),
		nats.PingInterval(20*time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),           // Unlimited reconnects
		nats.ReconnectBufSize(1024*1024), // 1MB buffer for reconnect
		// Add callbacks for connection events
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			log.Printf("NATS connection disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			log.Printf("NATS connection reconnected to %s", conn.ConnectedUrl())
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			log.Printf("NATS connection closed")
		}),
	)

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	nc.Conn = conn
	js, err := conn.JetStream()
	if err != nil {
		nc.JSConn = nil
	}

	nc.JSConn = &js

	jsJetStream, err := jetstream.New(nc.Conn)
	if err != nil {
		nc.JetStream = nil
	}
	nc.JetStream = &jsJetStream

	return nil
}

// IsHealthy checks if the connection is healthy and responsive
func (nc *NATSCredential) IsHealthy() bool {
	if nc.Conn == nil {
		return false
	}

	// Check if connection is still active
	if !nc.Conn.IsConnected() {
		return false
	}

	// Check if we can flush (send a ping)
	err := nc.Conn.FlushTimeout(2 * time.Second)
	return err == nil
}

func (nc *NATSCredential) Disconnect() {
	if nc.Conn != nil && nc.Conn.IsConnected() {
		nc.Conn.Close()
	}
}

func (nc *NATSCredential) GetInfo() (*NATSInfo, error) {
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return &NATSInfo{IsConnected: false}, nil
	}

	stats := nc.Conn.Stats()

	// Get basic server information
	serverInfo := map[string]interface{}{
		"server_id":     nc.Conn.ConnectedServerId(),
		"server_name":   nc.Conn.ConnectedServerName(),
		"connected_url": nc.Conn.ConnectedUrl(),
		"last_error":    nc.Conn.LastError(),
	}

	info := &NATSInfo{
		ServerInfo:   serverInfo,
		Stats:        stats,
		IsConnected:  nc.Conn.IsConnected(),
		ConnectedURL: nc.Conn.ConnectedUrl(),
	}

	return info, nil
}

func (nc *NATSCredential) GetAccountInfo() (*AccountInfo, error) {
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS server")
	}

	// Request account info using NATS request-reply pattern
	// This is a simplified version - in a real implementation you might need
	// to use the NATS system account or have proper permissions
	resp, err := nc.Conn.Request("$SYS.REQ.ACCOUNT.PING.CONNZ", nil, 2*time.Second)
	if err != nil {
		// If system requests are not available, return basic info
		return &AccountInfo{
			AccountInformation: nc.infoAction(),
			ConnectionLimits: map[string]interface{}{
				"max_connections":   "N/A",
				"max_subscriptions": "N/A",
			},
			Stats: map[string]interface{}{
				"connections":   "N/A",
				"subscriptions": "N/A",
			},
		}, nil
	}

	var accountData map[string]interface{}
	if err := json.Unmarshal(resp.Data, &accountData); err == nil {
		return &AccountInfo{
			AccountInformation: nc.infoAction(),
			ConnectionLimits:   accountData["data"].(map[string]interface{}),
			Stats:              accountData["data"].(map[string]interface{}),
		}, nil
	}

	// Return basic connection info if system requests fail
	stats := nc.Conn.Stats()
	return &AccountInfo{
		AccountInformation: nc.infoAction(),
		ConnectionLimits: map[string]interface{}{
			"max_connections": "N/A",
		},
		Stats: map[string]interface{}{
			"in_msgs":    stats.InMsgs,
			"out_msgs":   stats.OutMsgs,
			"in_bytes":   stats.InBytes,
			"out_bytes":  stats.OutBytes,
			"reconnects": stats.Reconnects,
		},
	}, nil
}

func (nc *NATSCredential) TestConnection() error {
	// Test basic connectivity with a ping
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return fmt.Errorf("not connected to NATS server")
	}

	// Test with a simple RTT measurement
	return nc.Conn.FlushTimeout(2 * time.Second)
}

func (nc *NATSCredential) infoAction() map[string]any {

	id, _ := nc.Conn.GetClientID()
	ip, _ := nc.Conn.GetClientIP()
	lip := nc.Conn.LocalAddr()
	rtt, _ := nc.Conn.RTT()
	tlsc, _ := nc.Conn.TLSConnectionState()

	// Simplified user info without server dependency
	var userInfo map[string]interface{}
	if util.ServerMinVersion(nc.Conn, 2, 10, 0) {
		subj := "$SYS.REQ.USER.INFO"
		resp, err := nc.Conn.Request(subj, nil, time.Second)
		if err == nil {
			var res map[string]interface{}
			err = json.Unmarshal(resp.Data, &res)
			if err == nil {
				if data, ok := res["data"].(map[string]interface{}); ok {
					userInfo = data
				}
			}
		}
	}

	// Extract user info safely
	var userID, account interface{}
	var expires interface{}
	var permissions interface{}
	if userInfo != nil {
		userID = userInfo["user_id"]
		account = userInfo["account"]
		expires = userInfo["expires"]
		permissions = userInfo["permissions"]
	}

	accountInfo := map[string]any{
		"user":             userID,
		"account":          account,
		"expires":          expires,
		"permissions":      permissions,
		"client_id":        id,
		"client_ip":        ip,
		"rtt":              rtt.String(),
		"header_supported": nc.Conn.HeadersSupported(),
		"max_payload":      humanize.IBytes(uint64(nc.Conn.MaxPayload())),
		"connected_url":    nc.Conn.ConnectedUrl(),
		"connected_addr":   nc.Conn.ConnectedAddr(),
		"server_id":        nc.Conn.ConnectedServerId(),
		"server_version":   nc.Conn.ConnectedServerVersion(),
		"server_name":      nc.Conn.ConnectedServerName(),
	}
	if expires == 0 {
		accountInfo["expires"] = "never"
	}
	if lip != "" && !strings.HasPrefix(lip, ip.String()) {
		accountInfo["local_ip"] = lip
	}

	if tlsc.HandshakeComplete {
		version := ""
		switch tlsc.Version {
		case tls.VersionTLS10:
			version = "1.0"
		case tls.VersionTLS11:
			version = "1.1"
		case tls.VersionTLS12:
			version = "1.2"
		case tls.VersionTLS13:
			version = "1.3"
		default:
			version = fmt.Sprintf("unknown (%x)", tlsc.Version)
		}

		accountInfo["tls_version"] = fmt.Sprintf("%s using %s", version, tls.CipherSuiteName(tlsc.CipherSuite))
		accountInfo["tls_server_name"] = tlsc.ServerName
		if len(tlsc.VerifiedChains) > 0 {
			accountInfo["tls_verified"] = fmt.Sprintf("issuer %s", tlsc.PeerCertificates[0].Issuer.String())
		} else {
			accountInfo["tls_verified"] = "no"
		}
	}

	if userInfo != nil && permissions != nil {
		accountInfo["permissions"] = permissions
	}

	return accountInfo
}

// IsConnected checks if the NATS connection is active
func (nc *NATSCredential) IsConnected() bool {
	return nc.Conn != nil && nc.Conn.IsConnected()
}

// Utility functions for JSON handling and timestamps
func ToJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func MustToJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"error": "json marshal failed"}`)
	}
	return data
}

func GetCurrentTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
