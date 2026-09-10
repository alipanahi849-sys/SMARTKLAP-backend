package ws

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ErrMissingToken is returned when no JWT was found in the request.
var ErrMissingToken = errors.New("missing authentication token")

// ErrInvalidToken is returned when the JWT is present but invalid or expired.
var ErrInvalidToken = errors.New("invalid or expired token")

// AuthResult is the outcome of a successful WebSocket authentication.
type AuthResult struct {
	UserID           uuid.UUID
	ExpiresAt        time.Time
	SelectedProtocol string
}

// Authenticate reads and validates the JWT from the WebSocket upgrade request.
//
// Tokens are accepted from (in order):
//  1. Authorization: Bearer <token>  (native / non-browser clients)
//  2. Sec-WebSocket-Protocol: bearer.<token>  (browsers cannot set WS headers)
//
// Query-string tokens are not supported because they leak into server logs,
// browser history, and Referer headers.
//
// On success it returns the authenticated user ID and the token expiry, which
// the connection uses to terminate the session when the token expires (F-008).
// SelectedProtocol is the subprotocol the upgrade response must echo when auth
// came from Sec-WebSocket-Protocol.
func Authenticate(r *http.Request) (*AuthResult, error) {
	token, selectedProtocol := tokenFromRequest(r)
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

	return &AuthResult{
		UserID:           claims.UserID,
		ExpiresAt:        expiresAt,
		SelectedProtocol: selectedProtocol,
	}, nil
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

func tokenFromRequest(r *http.Request) (token string, selectedProtocol string) {
	if token := tokenFromHeader(r); token != "" {
		return token, ""
	}
	return tokenFromSubprotocol(r)
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

// tokenFromSubprotocol supports browser clients that cannot set Authorization
// on the WebSocket handshake. Clients pass `bearer.<jwt>` as a subprotocol.
func tokenFromSubprotocol(r *http.Request) (token string, selectedProtocol string) {
	const prefix = "bearer."
	for _, protocol := range websocket.Subprotocols(r) {
		if strings.HasPrefix(protocol, prefix) {
			tok := strings.TrimPrefix(protocol, prefix)
			if tok != "" {
				return tok, protocol
			}
		}
	}
	// Fallback: some stacks expose the raw header as a comma list.
	raw := r.Header.Get("Sec-WebSocket-Protocol")
	for _, part := range strings.Split(raw, ",") {
		protocol := strings.TrimSpace(part)
		if strings.HasPrefix(protocol, prefix) {
			tok := strings.TrimPrefix(protocol, prefix)
			if tok != "" {
				return tok, protocol
			}
		}
	}
	return "", ""
}
