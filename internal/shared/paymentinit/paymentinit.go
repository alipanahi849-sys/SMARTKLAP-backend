// Package paymentinit builds the configured card payment provider.
package paymentinit

import (
	"clap/internal/shared/config"
	"clap/internal/shared/logger"
	"clap/pkg/payment"
)

// Provider returns the card payment backend configured via STRIPE_SECRET_KEY.
func Provider() payment.Provider {
	cfg := config.AppConfig
	if cfg == nil {
		return payment.NewStripeProvider(payment.StripeConfig{})
	}

	provider := payment.NewStripeProvider(payment.StripeConfig{
		SecretKey:     cfg.Stripe.SecretKey,
		WebhookSecret: cfg.Stripe.WebhookSecret,
	})
	if provider.Enabled() {
		logger.Info().Msg("Stripe payment provider ready")
	} else {
		logger.Warn().Msg("Stripe secret key not set; card payments disabled")
	}
	return provider
}
