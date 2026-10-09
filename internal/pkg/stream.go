package pkg

import (
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type StreamMessage struct {
	Sequence  uint64            `json:"sequence"`
	Subject   string            `json:"subject"`
	Data      string            `json:"data"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp string            `json:"timestamp"`
	Size      uint32            `json:"size"`
}

type StreamMessagesResponse struct {
	Messages   []StreamMessage `json:"messages"`
	Total      int             `json:"total"`
	Offset     int             `json:"offset"`
	Limit      int             `json:"limit"`
	StreamName string          `json:"stream_name"`
}

func (nc *NATSCredential) ListStreams() ([]*nats.StreamInfo, error) {
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS server")
	}

	if nc.JSConn == nil {
		return nil, fmt.Errorf("not connected to JetStream")
	}

	js := *nc.JSConn

	// Use the JetStream API to get streams
	streamInfos := js.Streams()
	streamNames := make([]*nats.StreamInfo, 0, len(streamInfos))
	for s := range streamInfos {
		streamNames = append(streamNames, s)
	}

	return streamNames, nil
}

func (nc *NATSCredential) ListConsumers(stream string) ([]*nats.ConsumerInfo, error) {
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS server")
	}

	if nc.JSConn == nil {
		return nil, fmt.Errorf("not connected to JetStream")
	}

	js := *nc.JSConn

	consumers := js.Consumers(stream)
	consumerList := make([]*nats.ConsumerInfo, 0, len(consumers))
	for c := range consumers {
		consumerList = append(consumerList, c)
	}

	return consumerList, nil
}

func (nc *NATSCredential) GetStreamInfo(stream string) (*nats.StreamInfo, error) {
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS server")
	}

	if nc.JSConn == nil {
		return nil, fmt.Errorf("not connected to JetStream")
	}

	js := *nc.JSConn

	streamInfo, err := js.StreamInfo(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	return streamInfo, nil
}

// StreamConfig represents the configuration for creating a new NATS stream
type StreamConfig struct {
	Name         string   `json:"name" binding:"required"`
	Subjects     []string `json:"subjects" binding:"required"`
	Storage      string   `json:"storage"`   // "file" or "memory"
	Retention    string   `json:"retention"` // "limits", "interest", or "workqueue"
	Discard      string   `json:"discard"`   // "old" or "new"
	NumReplicas  int      `json:"num_replicas"`
	AllowDirect  bool     `json:"allow_direct"`
	AllowMsgTTL  bool     `json:"allow_msg_ttl"`
	MaxMsgs      int64    `json:"max_msgs"`
	MaxBytes     int64    `json:"max_bytes"`
	MaxAge       int64    `json:"max_age"` // nanoseconds
	MaxConsumers int      `json:"max_consumers"`

	MaxMsgsPerSubject int64  `json:"max_msgs_per_subject"`
	MaxMsgSize        int32  `json:"max_msg_size"`
	DuplicateWindow   int64  `json:"duplicate_window"` // nanoseconds
	Compression       string `json:"compression"`      // "none" or "s2"
}

func (nc *NATSCredential) CreateStream(config *StreamConfig) (*nats.StreamInfo, error) {
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS server")
	}

	if nc.JSConn == nil {
		return nil, fmt.Errorf("not connected to JetStream")
	}

	js := *nc.JSConn

	// Convert our config to NATS StreamConfig
	streamConfig := &nats.StreamConfig{
		Name:        config.Name,
		Subjects:    config.Subjects,
		Replicas:    config.NumReplicas,
		AllowDirect: config.AllowDirect,
		AllowMsgTTL: config.AllowMsgTTL,
	}

	switch config.Compression {
	case "", "none":
		streamConfig.Compression = nats.NoCompression
	case "s2":
		streamConfig.Compression = nats.S2Compression
	default:
		return nil, fmt.Errorf("unknown compression %q", config.Compression)
	}

	// Set storage type
	switch config.Storage {
	case "file":
		streamConfig.Storage = nats.FileStorage
	case "memory":
		streamConfig.Storage = nats.MemoryStorage
	default:
		streamConfig.Storage = nats.FileStorage // default to file storage
	}

	// Set retention policy
	switch config.Retention {
	case "limits":
		streamConfig.Retention = nats.LimitsPolicy
	case "interest":
		streamConfig.Retention = nats.InterestPolicy
	case "workqueue":
		streamConfig.Retention = nats.WorkQueuePolicy
	default:
		streamConfig.Retention = nats.LimitsPolicy // default to limits
	}

	// Set discard policy
	switch config.Discard {
	case "old":
		streamConfig.Discard = nats.DiscardOld
	case "new":
		streamConfig.Discard = nats.DiscardNew
	default:
		streamConfig.Discard = nats.DiscardOld // default to discard old
	}

	// Set limits (handle negative values as unlimited)
	if config.MaxMsgs > 0 {
		streamConfig.MaxMsgs = config.MaxMsgs
	}
	if config.MaxBytes > 0 {
		streamConfig.MaxBytes = config.MaxBytes
	}
	if config.MaxAge > 0 {
		streamConfig.MaxAge = time.Duration(config.MaxAge)
	}
	if config.MaxConsumers > 0 {
		streamConfig.MaxConsumers = config.MaxConsumers
	}
	if config.MaxMsgsPerSubject > 0 {
		streamConfig.MaxMsgsPerSubject = config.MaxMsgsPerSubject
	}
	if config.MaxMsgSize > 0 {
		streamConfig.MaxMsgSize = config.MaxMsgSize
	}
	if config.DuplicateWindow > 0 {
		streamConfig.Duplicates = time.Duration(config.DuplicateWindow)
	}

	// Create the stream
	streamInfo, err := js.AddStream(streamConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	return streamInfo, nil
}

// DeleteStream deletes a JetStream stream
func (nc *NATSCredential) DeleteStream(streamName string) error {
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return fmt.Errorf("not connected to NATS server")
	}

	if nc.JSConn == nil {
		return fmt.Errorf("JetStream not available")
	}

	js := *nc.JSConn

	err := js.DeleteStream(streamName)
	if err != nil {
		return fmt.Errorf("failed to delete stream %s: %w", streamName, err)
	}

	return nil
}

// GetStreamMessages retrieves messages from a stream with pagination
func (nc *NATSCredential) GetStreamMessages(streamName string, offset int, limit int) (*StreamMessagesResponse, error) {
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS server")
	}

	if nc.JSConn == nil {
		return nil, fmt.Errorf("not connected to JetStream")
	}

	js := *nc.JSConn

	// Get stream info
	streamInfo, err := js.StreamInfo(streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	totalMessages := int(streamInfo.State.Msgs)

	// Validate pagination parameters
	if offset < 0 {
		offset = 0
	}
	if offset >= totalMessages && totalMessages > 0 {
		offset = totalMessages - 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}

	messages := make([]StreamMessage, 0, limit)

	// Return empty if no messages
	if totalMessages == 0 {
		return &StreamMessagesResponse{
			Messages:   messages,
			Total:      0,
			Offset:     offset,
			Limit:      limit,
			StreamName: streamName,
		}, nil
	}

	// Calculate starting sequence
	firstSeq := streamInfo.State.FirstSeq
	startSeq := firstSeq + uint64(offset)

	// Ensure we don't go past the last message
	if startSeq > streamInfo.State.LastSeq {
		startSeq = streamInfo.State.LastSeq
	}

	// Fetch messages directly using GetMsg
	for i := 0; i < limit && int(startSeq)+i <= int(streamInfo.State.LastSeq); i++ {
		currentSeq := startSeq + uint64(i)

		msg, err := js.GetMsg(streamName, currentSeq)
		if err != nil {
			// If we can't get this message, continue to next
			continue
		}

		headers := make(map[string]string)
		if msg.Header != nil {
			for key, values := range msg.Header {
				if len(values) > 0 {
					headers[key] = values[0]
				}
			}
		}

		streamMsg := StreamMessage{
			Sequence:  currentSeq,
			Subject:   msg.Subject,
			Data:      string(msg.Data),
			Headers:   headers,
			Timestamp: msg.Time.Format(time.RFC3339),
			Size:      uint32(len(msg.Data)),
		}

		messages = append(messages, streamMsg)
	}

	return &StreamMessagesResponse{
		Messages:   messages,
		Total:      totalMessages,
		Offset:     offset,
		Limit:      limit,
		StreamName: streamName,
	}, nil
}

// GetStreamMessagesWithSearch retrieves messages from a stream with pagination and search using lazy loading
// This fetches messages incrementally while filtering to avoid loading entire stream into memory
func (nc *NATSCredential) GetStreamMessagesWithSearch(streamName string, offset int, limit int, search string) (*StreamMessagesResponse, error) {
	if nc.Conn == nil || !nc.Conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS server")
	}

	if nc.JSConn == nil {
		return nil, fmt.Errorf("not connected to JetStream")
	}

	js := *nc.JSConn

	// Get stream info
	streamInfo, err := js.StreamInfo(streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	totalMessages := int(streamInfo.State.Msgs)

	// Validate pagination parameters
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}

	messages := make([]StreamMessage, 0, limit)

	// Return empty if no messages
	if totalMessages == 0 {
		return &StreamMessagesResponse{
			Messages:   messages,
			Total:      0,
			Offset:     offset,
			Limit:      limit,
			StreamName: streamName,
		}, nil
	}

	// If no search term, use regular pagination
	if search == "" {
		return nc.GetStreamMessages(streamName, offset, limit)
	}

	// Lazy loading: fetch messages in chunks while filtering
	searchLower := strings.ToLower(search)
	firstSeq := streamInfo.State.FirstSeq
	lastSeq := streamInfo.State.LastSeq

	matchedCount := 0
	pageStartIdx := offset
	pageEndIdx := pageStartIdx + limit
	currentMessageIdx := 0
	chunkSize := 100 // Fetch in chunks of 100 to avoid loading entire stream

	// Iterate through all messages from start
	for seq := firstSeq; seq <= lastSeq; seq++ {
		msg, err := js.GetMsg(streamName, seq)
		if err != nil {
			// Skip messages that can't be retrieved
			continue
		}

		// Check if message matches search criteria
		subjectMatch := strings.Contains(strings.ToLower(msg.Subject), searchLower)
		dataMatch := strings.Contains(strings.ToLower(string(msg.Data)), searchLower)

		if subjectMatch || dataMatch {
			// This message matches the search
			if currentMessageIdx >= pageStartIdx && currentMessageIdx < pageEndIdx {
				// This message is in the current page
				headers := make(map[string]string)
				if msg.Header != nil {
					for key, values := range msg.Header {
						if len(values) > 0 {
							headers[key] = values[0]
						}
					}
				}

				streamMsg := StreamMessage{
					Sequence:  seq,
					Subject:   msg.Subject,
					Data:      string(msg.Data),
					Headers:   headers,
					Timestamp: msg.Time.Format(time.RFC3339),
					Size:      uint32(len(msg.Data)),
				}

				messages = append(messages, streamMsg)
			}

			currentMessageIdx++
			matchedCount++

			// Optimization: stop early if we have enough matches for current page and some buffer
			if currentMessageIdx > pageEndIdx+chunkSize {
				break
			}
		}
	}

	return &StreamMessagesResponse{
		Messages:   messages,
		Total:      matchedCount, // Total matches for this search, not total messages
		Offset:     offset,
		Limit:      limit,
		StreamName: streamName,
	}, nil
}
