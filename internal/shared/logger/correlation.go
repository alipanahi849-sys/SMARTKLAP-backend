package logger

import (
	"context"

	"github.com/rs/zerolog"
)

// Structured-log field names used for request correlation.
// Keeping these as constants prevents typos and makes log aggregation reliable.
const (
	FieldRequestID = "request_id"
	FieldSessionID = "session_id"
	FieldMatchID   = "match_id"
	FieldEventID   = "event_id"
	FieldUserID    = "user_id"
)

// WithRequestID returns a child logger with request_id attached.
func WithRequestID(requestID string) zerolog.Logger {
	return Logger.With().Str(FieldRequestID, requestID).Logger()
}

// WithSessionID returns a child logger with session_id attached.
func WithSessionID(sessionID string) zerolog.Logger {
	return Logger.With().Str(FieldSessionID, sessionID).Logger()
}

// WithMatchID returns a child logger with match_id attached.
func WithMatchID(matchID string) zerolog.Logger {
	return Logger.With().Str(FieldMatchID, matchID).Logger()
}

// WithEventID returns a child logger with event_id attached.
func WithEventID(eventID string) zerolog.Logger {
	return Logger.With().Str(FieldEventID, eventID).Logger()
}

// WithUserID returns a child logger with user_id attached.
func WithUserID(userID string) zerolog.Logger {
	return Logger.With().Str(FieldUserID, userID).Logger()
}

// CorrelationFields bundles all standard correlation identifiers.
// Pass whichever are available; empty strings are omitted.
type CorrelationFields struct {
	RequestID string
	SessionID string
	MatchID   string
	EventID   string
	UserID    string
}

// WithCorrelation returns a child logger with every non-empty correlation field attached.
// This is the preferred helper when multiple IDs are available in one place
// (e.g. service methods that receive a full request context).
func WithCorrelation(fields CorrelationFields) zerolog.Logger {
	ctx := Logger.With()
	if fields.RequestID != "" {
		ctx = ctx.Str(FieldRequestID, fields.RequestID)
	}
	if fields.SessionID != "" {
		ctx = ctx.Str(FieldSessionID, fields.SessionID)
	}
	if fields.MatchID != "" {
		ctx = ctx.Str(FieldMatchID, fields.MatchID)
	}
	if fields.EventID != "" {
		ctx = ctx.Str(FieldEventID, fields.EventID)
	}
	if fields.UserID != "" {
		ctx = ctx.Str(FieldUserID, fields.UserID)
	}
	return ctx.Logger()
}

// ─── Context-key helpers ──────────────────────────────────────────────────────

type contextKey string

const (
	contextKeyRequestID contextKey = "request_id"
	contextKeySessionID contextKey = "session_id"
	contextKeyMatchID   contextKey = "match_id"
)

// ContextWithRequestID stores the request ID in a context for downstream propagation.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, contextKeyRequestID, requestID)
}

// RequestIDFromContext extracts the request ID stored by ContextWithRequestID.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyRequestID).(string); ok {
		return v
	}
	return ""
}

// ContextWithSessionID stores the session ID in a context.
func ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, contextKeySessionID, sessionID)
}

// SessionIDFromContext extracts the session ID stored by ContextWithSessionID.
func SessionIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeySessionID).(string); ok {
		return v
	}
	return ""
}

// ContextWithMatchID stores the match ID in a context.
func ContextWithMatchID(ctx context.Context, matchID string) context.Context {
	return context.WithValue(ctx, contextKeyMatchID, matchID)
}

// MatchIDFromContext extracts the match ID stored by ContextWithMatchID.
func MatchIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyMatchID).(string); ok {
		return v
	}
	return ""
}

// FromContext builds a correlated logger from all IDs stored in the context.
func FromContext(ctx context.Context) zerolog.Logger {
	return WithCorrelation(CorrelationFields{
		RequestID: RequestIDFromContext(ctx),
		SessionID: SessionIDFromContext(ctx),
		MatchID:   MatchIDFromContext(ctx),
	})
}
