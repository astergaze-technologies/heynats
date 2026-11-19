package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type KVManager struct {
	conn *NATSCredential
}

func NewKVManager(nc *NATSCredential) *KVManager {
	return &KVManager{conn: nc}
}

type KVBucketsStats struct {
	Bucket         string            `json:"bucket"`
	Values         uint64            `json:"values"`        // Total entries (including history)
	History        int64             `json:"history"`       // Per-key history
	TTL            string            `json:"ttl"`           // TTL as string
	BackingStore   string            `json:"backing_store"` // "file" or "memory"
	Bytes          uint64            `json:"bytes"`         // Total size
	IsCompressed   bool              `json:"is_compressed"` // Compression flag
	LimitMarkerTTL time.Duration     `json:"limit_marker_ttl"`
	Metadata       map[string]string `json:"metadata"`
}

func (kvm *KVManager) ListBucketsWithStats() ([]KVBucketsStats, error) {
	if kvm.conn == nil || kvm.conn.JetStream == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	stream := *kvm.conn.JetStream
	ctx := context.Background()
	kvLister := stream.KeyValueStores(ctx)
	if kvLister == nil {
		return nil, fmt.Errorf("failed to get KeyValueStores lister")
	}

	var stats []KVBucketsStats

	// Read from the Status channel until it's closed
	for status := range kvLister.Status() {
		if status == nil {
			continue
		}

		ttl := status.TTL()
		ttlStr := ""
		if ttl > 0 {
			ttlStr = ttl.String()
		}

		stats = append(stats, KVBucketsStats{
			Bucket:         status.Bucket(),
			Values:         status.Values(),
			History:        status.History(),
			TTL:            ttlStr,
			BackingStore:   status.BackingStore(),
			Bytes:          status.Bytes(),
			IsCompressed:   status.IsCompressed(),
			LimitMarkerTTL: status.LimitMarkerTTL(),
			Metadata:       status.Metadata(),
		})
	}

	// Check for errors during listing
	if err := kvLister.Error(); err != nil {
		return nil, fmt.Errorf("error listing KV buckets: %w", err)
	}

	return stats, nil
}

// GetBucket returns detailed information about a specific KV bucket
func (kvm *KVManager) GetBucket(bucketName string) (*KVBucketsStats, error) {
	if kvm.conn == nil || kvm.conn.JetStream == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	stream := *kvm.conn.JetStream

	ctx := context.Background()
	kv, err := stream.KeyValue(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to access bucket %s: %w", bucketName, err)
	}

	status, err := kv.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket status: %w", err)
	}

	return &KVBucketsStats{
		Bucket:         status.Bucket(),
		Values:         status.Values(),
		History:        status.History(),
		TTL:            status.TTL().String(),
		BackingStore:   status.BackingStore(),
		Bytes:          status.Bytes(),
		IsCompressed:   status.IsCompressed(),
		LimitMarkerTTL: status.LimitMarkerTTL(),
		Metadata:       status.Metadata(),
	}, nil
}

func (kvm *KVManager) CreateBucket(config jetstream.KeyValueConfig) error {
	if kvm.conn == nil || kvm.conn.JetStream == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	stream := *kvm.conn.JetStream

	ctx := context.Background()
	_, err := stream.CreateKeyValue(ctx, config)
	return err
}

// DeleteBucket removes a KV bucket entirely.
func (kvm *KVManager) DeleteBucket(bucketName string) error {
	if kvm.conn == nil || kvm.conn.JetStream == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	stream := *kvm.conn.JetStream

	ctx := context.Background()
	return stream.DeleteKeyValue(ctx, bucketName)
}

// KVEntry represents a key-value entry
type KVEntry struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Created  string `json:"created"`
	Revision uint64 `json:"revision,string"`
}

// GetBucketKeys returns all keys in a bucket
func (kvm *KVManager) GetBucketKeys(bucketName string) ([]KVEntry, error) {
	if kvm.conn == nil || kvm.conn.JetStream == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	ctx := context.Background()
	js := *kvm.conn.JetStream

	kv, err := js.KeyValue(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to access bucket %q: %w", bucketName, err)
	}

	keyLister, err := kv.ListKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var entries []KVEntry
	for key := range keyLister.Keys() {
		entry, err := kv.Get(ctx, key)
		if err != nil {
			continue
		}

		entries = append(entries, KVEntry{
			Key:      key,
			Value:    string(entry.Value()),
			Created:  entry.Created().Format(time.RFC3339),
			Revision: entry.Revision(),
		})
	}

	if errLister, ok := keyLister.(interface {
		Error() error
	}); ok {
		if err := errLister.Error(); err != nil {
			return nil, fmt.Errorf("error during key listing: %w", err)
		}
	}

	return entries, nil
}

func (kvm *KVManager) PutValue(bucket, key string, value []byte) (uint64, error) {
	if kvm.conn == nil || kvm.conn.JetStream == nil {
		return 0, fmt.Errorf("JetStream not initialized")
	}

	stream := *kvm.conn.JetStream

	ctx := context.Background()
	kv, err := stream.KeyValue(ctx, bucket)
	if err != nil {
		return 0, err
	}
	return kv.Put(ctx, key, value)
}

func (kvm *KVManager) GetValue(bucket, key string) ([]byte, error) {
	if kvm.conn == nil || kvm.conn.JetStream == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	stream := *kvm.conn.JetStream

	ctx := context.Background()
	kv, err := stream.KeyValue(ctx, bucket)
	if err != nil {
		return nil, err
	}
	entry, err := kv.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return entry.Value(), nil
}

func (kvm *KVManager) DeleteKey(bucket, key string) error {
	if kvm.conn == nil || kvm.conn.JetStream == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	stream := *kvm.conn.JetStream

	ctx := context.Background()
	kv, err := stream.KeyValue(ctx, bucket)
	if err != nil {
		return err
	}
	return kv.Delete(ctx, key)
}

// GetKey returns a specific key's value
// func (nc *NATSCredential) GetKey(bucketName, key string) (*KVEntry, error) {
// 	js := *nc.JSConn
// 	if js == nil {
// 		return nil, fmt.Errorf("JetStream not initialized")
// 	}

// 	kv, err := js.KeyValue(bucketName)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to access bucket %s: %w", bucketName, err)
// 	}

// 	entry, err := kv.Get(key)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
// 	}

// 	return &KVEntry{
// 		Key:      key,
// 		Value:    string(entry.Value()),
// 		Created:  entry.Created().Format(time.RFC3339),
// 		Revision: entry.Revision(),
// 	}, nil
// }

// // SetKey sets a key's value
// func (nc *NATSCredential) SetKey(bucketName, key, value string) (*KVEntry, error) {
// 	js := *nc.JSConn
// 	if js == nil {
// 		return nil, fmt.Errorf("JetStream not initialized")
// 	}

// 	kv, err := js.KeyValue(bucketName)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to access bucket %s: %w", bucketName, err)
// 	}

// 	revision, err := kv.Put(key, []byte(value))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to set key %s: %w", key, err)
// 	}

// 	// Return the updated entry
// 	entry, err := kv.Get(key)
// 	if err != nil {
// 		// Fallback to basic info if we can't retrieve the entry
// 		return &KVEntry{
// 			Key:      key,
// 			Value:    value,
// 			Created:  time.Now().Format(time.RFC3339),
// 			Revision: revision,
// 		}, nil
// 	}

// 	return &KVEntry{
// 		Key:      key,
// 		Value:    string(entry.Value()),
// 		Created:  entry.Created().Format(time.RFC3339),
// 		Revision: entry.Revision(),
// 	}, nil
// }

// func (nc *NATSCredential) ListKeyValues(bucket string, page, pageSize int) ([]struct {
// 	Key   string      `json:"key"`
// 	Value interface{} `json:"value"`
// }, error) {
// 	if nc.JSConn == nil {
// 		return nil, fmt.Errorf("JetStream not initialized")
// 	}

// 	js := *nc.JSConn
// 	kv, err := js.KeyValue(bucket)
// 	if err != nil {
// 		return nil, err
// 	}

// 	keys, err := kv.Keys()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Pagination
// 	start := page * pageSize
// 	if start >= len(keys) {
// 		return []struct {
// 			Key   string      `json:"key"`
// 			Value interface{} `json:"value"`
// 		}{}, nil
// 	}
// 	end := start + pageSize
// 	if end > len(keys) {
// 		end = len(keys)
// 	}
// 	sliceKeys := keys[start:end]

// 	result := make([]struct {
// 		Key   string      `json:"key"`
// 		Value interface{} `json:"value"`
// 	}, 0, len(sliceKeys))

// 	for _, key := range sliceKeys {
// 		entry, err := kv.Get(key)
// 		if err != nil {
// 			continue
// 		}

// 		raw := entry.Value()
// 		var parsed interface{}
// 		if err := json.Unmarshal(raw, &parsed); err != nil {
// 			parsed = string(raw) // fallback to string if not JSON
// 		}

// 		result = append(result, struct {
// 			Key   string      `json:"key"`
// 			Value interface{} `json:"value"`
// 		}{
// 			Key:   key,
// 			Value: parsed,
// 		})
// 	}

// 	return result, nil
// }

// func (nc *NATSCredential) GetValue(bucket, key string) ([]byte, error) {
// 	if nc.JSConn == nil {
// 		return nil, fmt.Errorf("JetStream not initialized")
// 	}

// 	stream := *nc.JSConn

// 	kv, err := stream.KeyValue(bucket)
// 	if err != nil {
// 		return nil, err
// 	}
// 	entry, err := kv.Get(key)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return entry.Value(), nil
// }
