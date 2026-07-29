package unit

import (
	"net/http"
	"testing"

	"clap/internal/modules/realtime/ws"

	"github.com/stretchr/testify/assert"
)

func TestExtractToken_MissingToken(t *testing.T) {
	req, _ := http.NewRequest("GET", "/realtime/ws", nil)
	_, err := ws.ExtractAndValidateToken(req)
	assert.ErrorIs(t, err, ws.ErrMissingToken)
}

func TestExtractToken_InvalidBearerToken(t *testing.T) {
	req, _ := http.NewRequest("GET", "/realtime/ws", nil)
	req.Header.Set("Authorization", "Bearer not.a.real.jwt")
	_, err := ws.ExtractAndValidateToken(req)
	assert.ErrorIs(t, err, ws.ErrInvalidToken)
}

func TestExtractToken_QueryTokenIgnored(t *testing.T) {
	// Query-string tokens are no longer accepted (F-010). A request with only a
	// query token must be treated as missing a token.
	req, _ := http.NewRequest("GET", "/realtime/ws?token=not.a.real.jwt", nil)
	_, err := ws.ExtractAndValidateToken(req)
	assert.ErrorIs(t, err, ws.ErrMissingToken)
}

func TestExtractToken_MalformedAuthorization(t *testing.T) {
	req, _ := http.NewRequest("GET", "/realtime/ws", nil)
	req.Header.Set("Authorization", "Token something") // wrong scheme
	_, err := ws.ExtractAndValidateToken(req)
	// The "Token" scheme is not "Bearer" so the header is ignored,
	// and query tokens are not supported.
	assert.ErrorIs(t, err, ws.ErrMissingToken)
}

func TestExtractToken_HeaderOnlyEvenWithQueryPresent(t *testing.T) {
	// A query token is present but must be ignored; the (invalid) header token
	// is the only one considered, yielding ErrInvalidToken.
	req, _ := http.NewRequest("GET", "/realtime/ws?token=querytoken", nil)
	req.Header.Set("Authorization", "Bearer headertoken")

	_, err := ws.ExtractAndValidateToken(req)
	assert.ErrorIs(t, err, ws.ErrInvalidToken)
}
