package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// Consumers registers event handlers. Because Subscribe uses a durable queue
// group, adding replicas increases throughput while each event is still handled
// exactly once by the group.
type Consumers struct {
	bus      port.EventBus
	catalog  *service.CatalogService
	mailer   port.Mailer
	logger   port.Logger
	tracer   port.Tracer
	notifier *service.NotificationService
}

func NewConsumers(bus port.EventBus, catalog *service.CatalogService, mailer port.Mailer, logger port.Logger, tracer port.Tracer, notifier *service.NotificationService) *Consumers {
	if tracer == nil {
		tracer = port.NoopTracer{}
	}
	return &Consumers{bus: bus, catalog: catalog, mailer: mailer, logger: logger, tracer: tracer, notifier: notifier}
}

// notify raises an in-app notification for the customer who owns an order.
// Guest orders have no user id and are skipped. code identifies the message
// template so the client can render it in the reader's own language.
func (c *Consumers) notify(ctx context.Context, payload *service.OrderEvent, ntype domain.NotificationType, code, title string) {
	if c.notifier == nil || payload.UserID == "" {
		return
	}
	_ = c.notifier.Notify(ctx, service.NotifyInput{
		UserID: payload.UserID, Type: ntype, Title: title, Link: "/orders/" + payload.OrderID,
		Data: map[string]any{
			"code": code, "orderId": payload.OrderID, "orderNo": payload.OrderNo,
			"amountCents": payload.TotalCents, "currency": payload.Currency,
		},
	})
}

// Start wires subscriptions. It returns an error if the broker rejects any
// subscription so the process can fail fast at boot.
func (c *Consumers) Start() error {
	handlers := []struct {
		subject string
		queue   string
		durable string
		fn      port.EventHandler
	}{
		{service.SubjectOrderCreated, "order-created", "order-created", c.onOrderCreated},
		{service.SubjectOrderPaid, "order-paid", "order-paid", c.onOrderPaid},
		{service.SubjectOrderCancelled, "order-cancelled", "order-cancelled", c.onOrderCancelled},
		{service.SubjectOrderRefunded, "order-refunded", "order-refunded", c.onOrderRefunded},
		{service.SubjectOrderShipped, "order-shipped", "order-shipped", c.onOrderShipped},
		{service.SubjectOrderCompleted, "order-completed", "order-completed", c.onOrderCompleted},
	}
	for _, h := range handlers {
		if err := c.bus.Subscribe(h.subject, h.queue, h.durable, c.withSpan(h.subject, h.fn)); err != nil {
			return fmt.Errorf("subscribe %s: %w", h.subject, err)
		}
	}
	c.logger.Info("event consumers started", "subjects", len(handlers))
	return nil
}

// withSpan continues the producer's trace and records the outcome of handling.
func (c *Consumers) withSpan(subject string, fn port.EventHandler) port.EventHandler {
	return func(ctx context.Context, evt port.Event) error {
		ctx = c.tracer.Extract(ctx, evt.TraceParent)
		ctx, span := c.tracer.Start(ctx, "consume "+subject,
			port.Attribute{Key: "messaging.system", Value: "nats"},
			port.Attribute{Key: "messaging.destination", Value: subject},
		)
		defer span.End()
		if err := fn(ctx, evt); err != nil {
			span.RecordError(err)
			return err
		}
		return nil
	}
}

func (c *Consumers) decode(evt port.Event) (*service.OrderEvent, error) {
	var payload service.OrderEvent
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}
	return &payload, nil
}

// invalidate refreshes the catalog cache after stock changes. Redis is shared,
// so a single invalidation is visible to every replica.
func (c *Consumers) invalidate(ctx context.Context, evt *service.OrderEvent) {
	if c.catalog == nil {
		return
	}
	ids := make([]string, 0, len(evt.Items))
	for _, it := range evt.Items {
		ids = append(ids, it.ProductID)
	}
	c.catalog.InvalidateProductCache(ctx, ids...)
}

func (c *Consumers) onOrderCreated(ctx context.Context, evt port.Event) error {
	payload, err := c.decode(evt)
	if err != nil {
		return err
	}
	c.invalidate(ctx, payload)
	c.logger.Info("order created", "orderNo", payload.OrderNo, "userId", payload.UserID, "total", payload.TotalCents)
	return nil
}

func (c *Consumers) onOrderPaid(ctx context.Context, evt port.Event) error {
	payload, err := c.decode(evt)
	if err != nil {
		return err
	}
	c.logger.Info("order paid", "orderNo", payload.OrderNo, "paymentId", payload.PaymentID)
	// Downstream effects (receipts, fulfilment) are decoupled from the request
	// that triggered the payment.
	if c.mailer != nil {
		_ = c.mailer.Send(ctx, port.Email{
			To:      payload.UserID + "@openshop.local",
			Subject: "Payment confirmed for order " + payload.OrderNo,
			HTML:    fmt.Sprintf("<p>We received your payment of %d %s.</p>", payload.TotalCents, payload.Currency),
		})
	}
	c.notify(ctx, payload, domain.NotificationOrder, "order.paid", "Order paid")
	return nil
}

func (c *Consumers) onOrderCancelled(ctx context.Context, evt port.Event) error {
	payload, err := c.decode(evt)
	if err != nil {
		return err
	}
	c.invalidate(ctx, payload)
	c.logger.Info("order cancelled", "orderNo", payload.OrderNo)
	c.notify(ctx, payload, domain.NotificationOrder, "order.cancelled", "Order cancelled")
	return nil
}

func (c *Consumers) onOrderRefunded(ctx context.Context, evt port.Event) error {
	payload, err := c.decode(evt)
	if err != nil {
		return err
	}
	c.invalidate(ctx, payload)
	c.logger.Info("order refunded", "orderNo", payload.OrderNo)
	if c.mailer != nil {
		_ = c.mailer.Send(ctx, port.Email{
			To:      payload.UserID + "@openshop.local",
			Subject: "Refund issued for order " + payload.OrderNo,
			HTML:    "<p>We have refunded your order. The amount will appear on your statement shortly.</p>",
		})
	}
	c.notify(ctx, payload, domain.NotificationOrder, "order.refunded", "Order refunded")
	return nil
}

func (c *Consumers) onOrderShipped(ctx context.Context, evt port.Event) error {
	payload, err := c.decode(evt)
	if err != nil {
		return err
	}
	c.logger.Info("order shipped", "orderNo", payload.OrderNo)
	if c.mailer != nil {
		_ = c.mailer.Send(ctx, port.Email{
			To:      payload.UserID + "@openshop.local",
			Subject: "Your order " + payload.OrderNo + " has shipped",
			HTML:    "<p>Good news — your order is on its way.</p>",
		})
	}
	c.notify(ctx, payload, domain.NotificationOrder, "order.shipped", "Order shipped")
	return nil
}

func (c *Consumers) onOrderCompleted(ctx context.Context, evt port.Event) error {
	payload, err := c.decode(evt)
	if err != nil {
		return err
	}
	c.logger.Info("order completed", "orderNo", payload.OrderNo)
	c.notify(ctx, payload, domain.NotificationOrder, "order.completed", "Order completed")
	return nil
}
