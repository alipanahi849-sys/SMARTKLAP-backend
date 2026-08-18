package newsfeed

import (
	"encoding/base64"
	"strings"

	"clap/internal/shared/errors"
)

// EncodeID turns a publisher article id (which may contain slashes) into a
// single URL path segment for /news/:news_id.
func EncodeID(providerID string) string {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(providerID))
}

// DecodeID reverses EncodeID.
func DecodeID(encoded string) (string, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return "", errors.NewBadRequest("Invalid news ID", nil)
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) == 0 {
		return "", errors.NewBadRequest("Invalid news ID", nil)
	}
	id := strings.TrimSpace(string(raw))
	if id == "" {
		return "", errors.NewBadRequest("Invalid news ID", nil)
	}
	return id, nil
}
