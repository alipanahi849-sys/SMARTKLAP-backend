package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/invoice"
	"github.com/stripe/stripe-go/v82/webhook"
)

// StripeConfig holds credentials for the Stripe API.
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
}

type stripeProvider struct {
	webhookSecret string
}

// NewStripeProvider returns a Stripe-backed payment provider.
func NewStripeProvider(cfg StripeConfig) Provider {
	stripe.Key = strings.TrimSpace(cfg.SecretKey)
	return &stripeProvider{webhookSecret: strings.TrimSpace(cfg.WebhookSecret)}
}

func (p *stripeProvider) Enabled() bool {
	return stripe.Key != ""
}

func (p *stripeProvider) CreateCheckoutSession(ctx context.Context, params CreateCheckoutParams) (*CheckoutSessionResult, error) {
	if !p.Enabled() {
		return nil, fmt.Errorf("stripe is not configured")
	}

	currency := strings.ToLower(strings.TrimSpace(params.Currency))
	if currency == "" {
		currency = "eur"
	}

	orderMeta := map[string]string{
		"order_id": params.OrderID.String(),
		"user_id":  params.UserID.String(),
	}

	sessionParams := &stripe.CheckoutSessionParams{
		Mode:             stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:       stripe.String(params.SuccessURL),
		CancelURL:        stripe.String(params.CancelURL),
		CustomerCreation: stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways)),
		InvoiceCreation: &stripe.CheckoutSessionInvoiceCreationParams{
			Enabled: stripe.Bool(true),
			InvoiceData: &stripe.CheckoutSessionInvoiceCreationInvoiceDataParams{
				Description: stripe.String(fmt.Sprintf("SMARTKLAP order %s", params.OrderID)),
				Metadata:    orderMeta,
			},
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(currency),
					UnitAmount: stripe.Int64(params.AmountCents),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("SMARTKLAP Order"),
					},
				},
			},
		},
		Metadata: orderMeta,
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: orderMeta,
		},
		ClientReferenceID: stripe.String(params.OrderID.String()),
	}
	if email := strings.TrimSpace(params.CustomerEmail); email != "" {
		sessionParams.CustomerEmail = stripe.String(email)
	}
	sessionParams.Context = ctx

	checkoutSession, err := session.New(sessionParams)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return &CheckoutSessionResult{
		SessionID:   checkoutSession.ID,
		CheckoutURL: checkoutSession.URL,
	}, nil
}

func (p *stripeProvider) GetCheckoutSession(ctx context.Context, sessionID string) (*CheckoutSessionStatus, error) {
	if !p.Enabled() {
		return nil, fmt.Errorf("stripe is not configured")
	}

	params := &stripe.CheckoutSessionParams{}
	params.Context = ctx
	params.AddExpand("payment_intent")
	params.AddExpand("invoice")

	checkoutSession, err := session.Get(sessionID, params)
	if err != nil {
		return nil, fmt.Errorf("get checkout session: %w", err)
	}

	orderID, err := orderIDFromCheckoutSession(checkoutSession)
	if err != nil {
		return nil, err
	}

	intentID := ""
	if checkoutSession.PaymentIntent != nil {
		intentID = checkoutSession.PaymentIntent.ID
	}

	return &CheckoutSessionStatus{
		SessionID:       checkoutSession.ID,
		PaymentStatus:   string(checkoutSession.PaymentStatus),
		OrderID:         orderID,
		PaymentIntentID: intentID,
	}, nil
}

func (p *stripeProvider) EmailCheckoutInvoice(ctx context.Context, sessionID string) error {
	if !p.Enabled() {
		return nil
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || !strings.HasPrefix(sessionID, "cs_") {
		return nil
	}

	params := &stripe.CheckoutSessionParams{}
	params.Context = ctx
	params.AddExpand("invoice")

	checkoutSession, err := session.Get(sessionID, params)
	if err != nil {
		return fmt.Errorf("get checkout session invoice: %w", err)
	}
	return emailInvoiceFromSession(checkoutSession)
}

func orderIDFromCheckoutSession(checkoutSession *stripe.CheckoutSession) (uuid.UUID, error) {
	if checkoutSession.Metadata != nil {
		if orderID, err := orderIDFromMetadata(checkoutSession.Metadata); err == nil {
			return orderID, nil
		}
	}
	if ref := strings.TrimSpace(checkoutSession.ClientReferenceID); ref != "" {
		return uuid.Parse(ref)
	}
	return uuid.Nil, fmt.Errorf("checkout session missing order_id")
}

func (p *stripeProvider) ParseWebhookEvent(payload []byte, signature string) (*WebhookEvent, error) {
	if p.webhookSecret == "" {
		return nil, fmt.Errorf("stripe webhook secret is not configured")
	}

	event, err := webhook.ConstructEventWithOptions(payload, signature, p.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, fmt.Errorf("verify webhook signature: %w", err)
	}

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted,
		stripe.EventTypePaymentIntentSucceeded,
		stripe.EventTypePaymentIntentPaymentFailed,
		stripe.EventTypeInvoiceFinalized:
	default:
		return nil, nil
	}

	var orderID uuid.UUID
	var intentID string
	succeeded := false

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		var checkoutSession stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &checkoutSession); err != nil {
			return nil, fmt.Errorf("parse checkout session: %w", err)
		}
		orderID, err = orderIDFromCheckoutSession(&checkoutSession)
		if err != nil {
			return nil, err
		}
		intentID = checkoutSession.ID
		succeeded = checkoutSession.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid ||
			checkoutSession.Status == stripe.CheckoutSessionStatusComplete
		if err := emailInvoiceFromSession(&checkoutSession); err != nil {
			return nil, err
		}

	case stripe.EventTypePaymentIntentSucceeded, stripe.EventTypePaymentIntentPaymentFailed:
		var intent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &intent); err != nil {
			return nil, fmt.Errorf("parse payment intent: %w", err)
		}
		orderID, err = orderIDFromMetadata(intent.Metadata)
		if err != nil {
			return nil, err
		}
		intentID = intent.ID
		succeeded = event.Type == stripe.EventTypePaymentIntentSucceeded

	case stripe.EventTypeInvoiceFinalized:
		if err := sendFinalizedInvoice(event.Data.Raw); err != nil {
			return nil, err
		}
		return nil, nil
	}

	return &WebhookEvent{
		ID:        event.ID,
		Type:      string(event.Type),
		IntentID:  intentID,
		OrderID:   orderID,
		Succeeded: succeeded,
	}, nil
}

func orderIDFromMetadata(metadata map[string]string) (uuid.UUID, error) {
	orderIDRaw := strings.TrimSpace(metadata["order_id"])
	if orderIDRaw == "" {
		return uuid.Nil, fmt.Errorf("stripe event missing order_id metadata")
	}
	orderID, err := uuid.Parse(orderIDRaw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid order_id metadata: %w", err)
	}
	return orderID, nil
}

func sendFinalizedInvoice(raw json.RawMessage) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(raw, &inv); err != nil {
		return fmt.Errorf("parse invoice: %w", err)
	}
	return sendInvoiceID(inv.ID)
}

func emailInvoiceFromSession(checkoutSession *stripe.CheckoutSession) error {
	if checkoutSession == nil || checkoutSession.Invoice == nil {
		return nil
	}
	return sendInvoiceID(checkoutSession.Invoice.ID)
}

func sendInvoiceID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if _, err := invoice.SendInvoice(id, &stripe.InvoiceSendInvoiceParams{}); err != nil && !alreadySentInvoice(err) {
		return fmt.Errorf("email invoice %s: %w", id, err)
	}
	return nil
}

func alreadySentInvoice(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already") && strings.Contains(msg, "sent")
}
