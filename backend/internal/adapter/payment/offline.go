package payment

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// Offline is the manual/bank-transfer channel: the order is placed and awaits
// confirmation by an operator. It has no external API and no webhooks.
type Offline struct{}

func NewOffline() *Offline { return &Offline{} }

func (o *Offline) Name() string { return "offline" }

func (o *Offline) Charge(_ context.Context, _ port.ChargeRequest) (*port.ChargeResult, error) {
	return &port.ChargeResult{
		ProviderRef: "offline_" + uuid.NewString(),
		Status:      domain.PaymentPending,
	}, nil
}

func (o *Offline) Refund(_ context.Context, _ port.RefundRequest) error { return nil }

func (o *Offline) ParseWebhook(_ context.Context, _ map[string]string, _ []byte) (*port.WebhookEvent, error) {
	return nil, errors.New("offline: this channel has no webhooks")
}
