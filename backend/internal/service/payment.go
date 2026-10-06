package service

import (
	"context"
	"fmt"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// PaymentService orchestrates charges. It talks to whatever provider the
// registry returns, so adding a gateway is a configuration change.
type PaymentService struct {
	payments port.PaymentRepository
	orders   port.OrderRepository
	registry port.PaymentRegistry
	orderSvc *OrderService
	ids      port.IDGenerator
	clock    port.Clock
	logger   port.Logger
}

func NewPaymentService(
	payments port.PaymentRepository,
	orders port.OrderRepository,
	registry port.PaymentRegistry,
	orderSvc *OrderService,
	ids port.IDGenerator,
	clock port.Clock,
	logger port.Logger,
) *PaymentService {
	return &PaymentService{
		payments: payments, orders: orders, registry: registry, orderSvc: orderSvc,
		ids: ids, clock: clock, logger: logger,
	}
}

type CreatePaymentInput struct {
	UserID       string
	OrderID      string
	ProviderName string
	ReturnURL    string
}

type CreatePaymentResult struct {
	Payment     *domain.Payment
	RedirectURL string
}

func (s *PaymentService) Create(ctx context.Context, in CreatePaymentInput) (*CreatePaymentResult, error) {
	order, err := s.orders.FindByID(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != in.UserID {
		return nil, domain.ErrNotFound
	}
	if !order.Payable() {
		return nil, domain.ErrOrderNotPayable
	}

	provider, err := s.provider(in.ProviderName)
	if err != nil {
		return nil, err
	}
	charge, err := provider.Charge(ctx, port.ChargeRequest{
		OrderNo:     order.OrderNo,
		AmountCents: order.TotalCents,
		Currency:    order.Currency,
		Subject:     order.UserID,
		ReturnURL:   in.ReturnURL,
		Metadata:    map[string]string{"orderId": order.ID},
	})
	if err != nil {
		return nil, fmt.Errorf("charge: %w", err)
	}

	now := s.clock.Now()
	payment := &domain.Payment{
		ID: s.ids.NewID(), OrderID: order.ID, UserID: order.UserID,
		Provider: provider.Name(), ProviderRef: charge.ProviderRef,
		AmountCents: order.TotalCents, Currency: order.Currency,
		Status: charge.Status, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.payments.Create(ctx, payment); err != nil {
		return nil, err
	}

	order.PaymentID = payment.ID
	order.UpdatedAt = now
	if err := s.orders.Update(ctx, order); err != nil {
		return nil, err
	}

	return &CreatePaymentResult{Payment: payment, RedirectURL: charge.RedirectURL}, nil
}

// HandleWebhook verifies and applies a provider callback. Duplicate callbacks
// are ignored, which matters because brokers deliver at least once.
func (s *PaymentService) HandleWebhook(ctx context.Context, providerName string, headers map[string]string, body []byte) error {
	provider, err := s.provider(providerName)
	if err != nil {
		return err
	}
	evt, err := provider.ParseWebhook(ctx, headers, body)
	if err != nil {
		return err
	}

	payment, err := s.payments.FindByProviderRef(ctx, provider.Name(), evt.ProviderRef)
	if err != nil {
		return err
	}
	if payment.Status == domain.PaymentSucceeded || payment.Status == domain.PaymentRefunded {
		s.logger.Info("duplicate payment webhook ignored", "paymentId", payment.ID, "status", payment.Status)
		return nil
	}

	payment.Status = evt.Status
	payment.UpdatedAt = s.clock.Now()
	if evt.Status == domain.PaymentFailed {
		payment.FailureReason = "provider reported failure"
	}
	if err := s.payments.Update(ctx, payment); err != nil {
		return err
	}

	if evt.Status == domain.PaymentSucceeded {
		if _, err := s.orderSvc.MarkPaid(ctx, payment.OrderID, payment.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *PaymentService) provider(name string) (port.PaymentProvider, error) {
	if name == "" {
		if p := s.registry.Default(); p != nil {
			return p, nil
		}
		return nil, domain.ErrInvalidArgument
	}
	return s.registry.Get(name)
}

// Simulate drives a sandbox provider callback for the given payment. It is
// used by the development storefront so the whole checkout can be completed
// without a real gateway. It enforces ownership and only works for providers
// that implement port.SandboxProvider.
func (s *PaymentService) Simulate(ctx context.Context, userID, providerName, providerRef string) error {
	provider, err := s.provider(providerName)
	if err != nil {
		return err
	}
	sandbox, ok := provider.(port.SandboxProvider)
	if !ok {
		return fmt.Errorf("%w: provider does not support simulation", domain.ErrInvalidArgument)
	}

	payment, err := s.payments.FindByProviderRef(ctx, provider.Name(), providerRef)
	if err != nil {
		return err
	}
	if payment.UserID != userID {
		return domain.ErrNotFound
	}
	order, err := s.orders.FindByID(ctx, payment.OrderID)
	if err != nil {
		return err
	}

	headers, body := sandbox.BuildWebhook(order.OrderNo, payment.ProviderRef, domain.PaymentSucceeded, payment.AmountCents)
	return s.HandleWebhook(ctx, provider.Name(), headers, body)
}
