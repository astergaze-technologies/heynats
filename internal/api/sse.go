package api

import (
	"time"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

type sseSubscription struct {
	conn        *pkg.NATSCredential
	subject     string
	queueGroup  string
	queue       bool
	maxMessages int
	connected   map[string]any
	event       func(*nats.Msg) map[string]any
}

// streamSSE streams messages as SSE events. The channel is never closed and is
// read only here, so nothing can panic on the NATS callback goroutine.
func streamSSE(c *gin.Context, s sseSubscription) error {
	msgs := make(chan *nats.Msg, 256)

	var sub *nats.Subscription
	var err error
	if s.queue || s.queueGroup != "" {
		sub, err = s.conn.Conn.ChanQueueSubscribe(s.subject, s.queueGroup, msgs)
	} else {
		sub, err = s.conn.Conn.ChanSubscribe(s.subject, msgs)
	}
	if err != nil {
		return err
	}
	defer func() { _ = sub.Unsubscribe() }()
	if s.maxMessages > 0 {
		if err := sub.AutoUnsubscribe(s.maxMessages); err != nil {
			return err
		}
	}
	_ = s.conn.Conn.FlushTimeout(2 * time.Second)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	write := func(v any) {
		_, _ = c.Writer.WriteString("data: " + string(pkg.MustToJSON(v)) + "\n\n")
		c.Writer.Flush()
	}

	s.connected["timestamp"] = pkg.GetCurrentTimestamp()
	write(s.connected)

	count := 0
	for {
		select {
		case <-c.Request.Context().Done():
			return nil
		case m := <-msgs:
			write(s.event(m))
			count++
			if s.maxMessages > 0 && count >= s.maxMessages {
				write(map[string]any{
					"type":      "completed",
					"subject":   s.subject,
					"count":     count,
					"timestamp": pkg.GetCurrentTimestamp(),
				})
				return nil
			}
		}
	}
}

func firstHeaderValues(h nats.Header) map[string]string {
	headers := make(map[string]string, len(h))
	for key, values := range h {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}
