// Package nats implements port.EventBus on top of NATS JetStream. Publish is
// fire-and-forget from the request path, and every consumer uses a durable
// queue group so that, across N instances, exactly one processes each message.
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/port"
)

// Bus is a JetStream-backed event bus.
type Bus struct {
	conn     *nats.Conn
	js       nats.JetStreamContext
	stream   string
	prefix   string
	logger   port.Logger
	handlers []*nats.Subscription
}

func NewBus(cfg config.NATSConfig, logger port.Logger) (*Bus, error) {
	conn, err := nats.Connect(cfg.URL,
		nats.Name("openshop-"+cfg.ConsumerPrefix),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.Timeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := conn.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("nats jetstream: %w", err)
	}

	b := &Bus{conn: conn, js: js, stream: cfg.Stream, prefix: cfg.ConsumerPrefix, logger: logger}
	if err := b.ensureStream(); err != nil {
		conn.Close()
		return nil, err
	}
	return b, nil
}

func (b *Bus) ensureStream() error {
	subjects := []string{b.stream + ".>"}
	if _, err := b.js.StreamInfo(b.stream); err != nil {
		if _, addErr := b.js.AddStream(&nats.StreamConfig{
			Name:      b.stream,
			Subjects:  subjects,
			Retention: nats.WorkQueuePolicy,
			Storage:   nats.FileStorage,
			MaxAge:    24 * time.Hour,
		}); addErr != nil {
			if _, updErr := b.js.UpdateStream(&nats.StreamConfig{
				Name:      b.stream,
				Subjects:  subjects,
				Retention: nats.WorkQueuePolicy,
				Storage:   nats.FileStorage,
				MaxAge:    24 * time.Hour,
			}); updErr != nil {
				return fmt.Errorf("nats ensure stream: %w", updErr)
			}
		}
	}
	return nil
}

func (b *Bus) subject(s string) string { return b.stream + "." + s }

// Ready reports connection health for readiness probes.
func (b *Bus) Ready() error {
	if b.conn.Status() != nats.CONNECTED {
		return fmt.Errorf("nats status: %s", b.conn.Status())
	}
	return nil
}

func (b *Bus) Publish(ctx context.Context, evt port.Event) error {
	payload := evt.Payload
	if payload == nil {
		payload = []byte("{}")
	}
	msg := nats.NewMsg(b.subject(evt.Subject))
	msg.Data = payload
	if evt.ID != "" {
		msg.Header.Set("X-Event-Id", evt.ID)
	}
	if _, err := b.js.PublishMsg(msg, nats.Context(ctx)); err != nil {
		return fmt.Errorf("publish %s: %w", evt.Subject, err)
	}
	return nil
}

func (b *Bus) Subscribe(subject, queue, durable string, handler port.EventHandler) error {
	// A durable consumer named durable+queue means all replicas share one cursor,
	// so work is distributed rather than duplicated.
	name := fmt.Sprintf("%s-%s", b.prefix, durable)
	sub, err := b.js.QueueSubscribe(
		b.subject(subject),
		queue,
		b.wrap(handler),
		nats.Durable(name),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.AckWait(30*time.Second),
		nats.MaxDeliver(5),
	)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", subject, err)
	}
	b.handlers = append(b.handlers, sub)
	return nil
}

func (b *Bus) wrap(handler port.EventHandler) nats.MsgHandler {
	return func(m *nats.Msg) {
		evt := port.Event{
			Subject: m.Subject,
			ID:      m.Header.Get("X-Event-Id"),
			Payload: m.Data,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		if err := handler(ctx, evt); err != nil {
			b.logger.Error("event handler failed", "subject", m.Subject, "error", err)
			_ = m.Nak()
			return
		}
		_ = m.Ack()
	}
}

func (b *Bus) Close() error {
	for _, h := range b.handlers {
		_ = h.Unsubscribe()
	}
	return b.conn.Drain()
}

// Encode is a small helper so producers never hand-roll JSON marshalling.
func Encode(v any) ([]byte, error) { return json.Marshal(v) }

var _ port.EventBus = (*Bus)(nil)
