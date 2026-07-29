package service

import (
	"context"

	"clap/internal/shared/logger"
)

// OTPSender delivers a one-time code to a user. The production deployment
// plugs an email/SMS provider behind this interface; no provider is bundled
// with the backend, so the default sender logs the delivery event (with the
// code redacted outside development).
type OTPSender interface {
	SendOTP(ctx context.Context, email, code string) error
}

type logOTPSender struct {
	// revealCode controls whether the code itself is logged. Only enable in
	// development — production must deliver codes out-of-band only.
	revealCode bool
}

// NewLogOTPSender returns an OTPSender that records delivery via structured
// logs. revealCode=true additionally logs the code for local testing.
func NewLogOTPSender(revealCode bool) OTPSender {
	return &logOTPSender{revealCode: revealCode}
}

func (s *logOTPSender) SendOTP(_ context.Context, email, code string) error {
	evt := logger.Info().Str("email", email)
	if s.revealCode {
		evt = evt.Str("otp_code", code)
	}
	evt.Msg("otp_sent")
	return nil
}
