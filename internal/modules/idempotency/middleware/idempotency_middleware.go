package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"clap/internal/modules/idempotency/service"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// idempotencyHeader is the client-supplied deduplication token.
const idempotencyHeader = "X-Idempotency-Key"

// bodyCapture wraps gin.ResponseWriter to record the response body
// so it can be stored for future replay.
type bodyCapture struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyCapture) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Idempotency is a Gin middleware that short-circuits duplicate mutating
// requests when the client supplies an X-Idempotency-Key header.
//
// Flow:
//  1. If header absent → pass through untouched.
//  2. Hash method + full path + raw body → requestHash.
//  3. Look up key+endpoint in DB.
//  4. If found:  validate body hash (same body?), replay cached response.
//  5. If absent: wrap writer to capture response body, execute handler,
//     persist result if status is 2xx.
//
// Only 2xx responses are cached; errors are never stored.
func Idempotency(svc service.IdempotencyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader(idempotencyHeader)
		if key == "" {
			c.Next()
			return
		}

		// Read and restore body so the handler can consume it normally.
		rawBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.BadRequest(c, "cannot read request body")
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))

		requestHash := computeHash(c.Request.Method, c.FullPath(), rawBody)
		endpoint := fmt.Sprintf("%s:%s", c.Request.Method, c.FullPath())

		existing, findErr := svc.FindExisting(c.Request.Context(), key, endpoint)
		if findErr == nil {
			// Cached result exists — validate it was the same request.
			if err := svc.ValidateRequestHash(existing, requestHash); err != nil {
				response.UnprocessableEntity(c, err.Error())
				c.Abort()
				return
			}
			// Replay the cached response.
			c.Data(existing.StatusCode, "application/json; charset=utf-8", []byte(existing.ResponsePayload))
			c.Abort()
			return
		}

		// No cached result — execute handler while capturing the response body.
		capture := &bodyCapture{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = capture

		c.Next()

		// Persist only successful responses.
		statusCode := capture.Status()
		if statusCode >= 200 && statusCode < 300 {
			_ = svc.Store(
				c.Request.Context(),
				key,
				endpoint,
				requestHash,
				capture.body.String(),
				statusCode,
			)
		}
	}
}

func computeHash(method, path string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte(":"))
	h.Write([]byte(path))
	h.Write([]byte(":"))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
