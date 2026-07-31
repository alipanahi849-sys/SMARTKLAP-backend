package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"clap/internal/shared/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Roles  []string  `json:"roles"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func GenerateAccessToken(userID uuid.UUID, email string, roles []string) (string, int64, error) {
	cfg := config.AppConfig.JWT

	expiresAt := time.Now().Add(time.Duration(cfg.AccessExpiry) * time.Second)
	claims := Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", 0, err
	}

	return tokenString, int64(cfg.AccessExpiry), nil
}

// GenerateRefreshToken returns an opaque random refresh token (~96 hex chars),
// about half the length of the previous JWT-based refresh tokens. Binding to a
// user and expiry is stored server-side (hashed) — userID is kept for API symmetry.
func GenerateRefreshToken(userID uuid.UUID) (string, time.Time, error) {
	_ = userID
	cfg := config.AppConfig.JWT
	expiresAt := time.Now().Add(time.Duration(cfg.RefreshExpiry) * time.Second)

	// 48 random bytes → 96 hex characters (384 bits of entropy).
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}

	return hex.EncodeToString(b), expiresAt, nil
}

func ValidateAccessToken(tokenString string) (*Claims, error) {
	cfg := config.AppConfig.JWT

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(cfg.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ValidateRefreshToken is no longer used for opaque refresh tokens; lookup is
// done via hashed token in the database. Kept for compatibility and returns an error.
func ValidateRefreshToken(tokenString string) (uuid.UUID, error) {
	_ = tokenString
	return uuid.Nil, errors.New("opaque refresh tokens must be validated via the token store")
}
