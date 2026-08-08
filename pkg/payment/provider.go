package payment

import (
	"context"

	"github.com/google/uuid"
)

// CreateCheckoutParams is input for creating a Stripe Checkout Session.
type CreateCheckoutParams struct {
	OrderID     uuid.UUID
	UserID      uuid.UUID
	AmountCents int64
	Currency    string
	SuccessURL  string
	CancelURL   string
}

// CheckoutSessionResult is returned after creating a browser checkout session.
type CheckoutSessionResult struct {
	SessionID   string
	CheckoutURL string
}

// WebhookEvent is a verified Stripe webhook payload relevant to checkout.
type WebhookEvent struct {
	ID        string
	Type      string
	IntentID  string
	OrderID   uuid.UUID
	Succeeded bool
}

// CheckoutSessionStatus is the remote payment state for a browser checkout session.
type CheckoutSessionStatus struct {
	SessionID       string
	PaymentStatus   string
	OrderID         uuid.UUID
	PaymentIntentID string
}

// Provider abstracts card payment operations (Stripe in production).
type Provider interface {
	Enabled() bool
	CreateCheckoutSession(ctx context.Context, params CreateCheckoutParams) (*CheckoutSessionResult, error)
	GetCheckoutSession(ctx context.Context, sessionID string) (*CheckoutSessionStatus, error)
	ParseWebhookEvent(payload []byte, signature string) (*WebhookEvent, error)
}
