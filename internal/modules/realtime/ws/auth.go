package ws

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

// ErrMissingToken is returned when no JWT was found in the request.
var ErrMissingToken = errors.New("missing authentication token")

// ErrInvalidToken is returned when the JWT is present but invalid or expired.
var ErrInvalidToken = errors.New("invalid or expired token")

// AuthResult is the outcome of a successful WebSocket authentication.
type AuthResult struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
}

// Authenticate reads and validates the JWT from the WebSocket upgrade request.
//
// For security (F-010) the token is accepted ONLY from the Authorization header
// using the Bearer scheme. Query-string tokens are not supported because they
// leak into server logs, browser history, and Referer headers.
//
// On success it returns the authenticated user ID and the token expiry, which
// the connection uses to terminate the session when the token expires (F-008).
func Authenticate(r *http.Request) (*AuthResult, error) {
	token := tokenFromHeader(r)
	if token == "" {
		return nil, ErrMissingToken
	}

	claims, err := utils.ValidateAccessToken(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return &AuthResult{UserID: claims.UserID, ExpiresAt: expiresAt}, nil
}

// ExtractAndValidateToken is retained for backward compatibility. It returns
// only the authenticated user ID. New callers should prefer Authenticate to
// also obtain the token expiry for session-expiry enforcement.
func ExtractAndValidateToken(r *http.Request) (uuid.UUID, error) {
	res, err := Authenticate(r)
	if err != nil {
		return uuid.Nil, err
	}
	return res.UserID, nil
}

func tokenFromHeader(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return strings.TrimPrefix(auth, prefix)
}
