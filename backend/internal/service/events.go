package service

import (
	"encoding/json"
)

// Event subjects published on the event bus.
const (
	SubjectOrderCreated   = "order.created"
	SubjectOrderPaid      = "order.paid"
	SubjectOrderCancelled = "order.cancelled"
	SubjectOrderRefunded  = "order.refunded"
)

// OrderEvent is the payload for order lifecycle events. Consumers (email,
// analytics, fulfilment) decode this without importing the order service.
type OrderEvent struct {
	OrderID    string           `json:"orderId"`
	OrderNo    string           `json:"orderNo"`
	UserID     string           `json:"userId"`
	Status     string           `json:"status"`
	TotalCents int64            `json:"totalCents"`
	Currency   string           `json:"currency"`
	PaymentID  string           `json:"paymentId,omitempty"`
	OccurredAt string           `json:"occurredAt"`
	Items      []OrderEventItem `json:"items"`
}

type OrderEventItem struct {
	ProductID  string `json:"productId"`
	Title      string `json:"title"`
	Quantity   int    `json:"quantity"`
	PriceCents int64  `json:"priceCents"`
}

func encodeEvent(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}
