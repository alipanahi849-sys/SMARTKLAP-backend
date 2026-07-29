package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashRefreshToken creates a SHA-256 hash of a refresh token for secure storage
func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// VerifyRefreshToken compares an incoming token with a stored hash
func VerifyRefreshToken(token string, storedHash string) bool {
	hash := HashRefreshToken(token)
	return hash == storedHash
}
