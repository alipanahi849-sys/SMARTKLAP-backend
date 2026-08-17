package service

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"clap/internal/shared/config"
	"clap/internal/shared/errors"

	"github.com/golang-jwt/jwt/v5"
)

const appleCertsURL = "https://appleid.apple.com/auth/keys"

const appleIssuer = "https://appleid.apple.com"

// AppleIdentity is the verified identity extracted from an Apple identity token.
// Email is only present on the first authorization (and sometimes later);
// Subject is the stable Apple user identifier.
type AppleIdentity struct {
	Subject string
	Email   string
	Given   string
	Family  string
}

// AppleTokenVerifier verifies a Sign in with Apple identity token.
type AppleTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*AppleIdentity, error)
}

type appleIDTokenVerifier struct {
	audiences  []string
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	keysUntil time.Time
}

type appleJWTClaims struct {
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	jwt.RegisteredClaims
}

type appleJWKS struct {
	Keys []googleJWK `json:"keys"`
}

func appleVerifierFromConfig() AppleTokenVerifier {
	if config.AppConfig == nil {
		return nil
	}
	ids := config.AppConfig.Apple.ClientIDs
	if len(ids) == 0 {
		return nil
	}
	return NewAppleIDTokenVerifier(ids)
}

// NewAppleIDTokenVerifier verifies RS256 Apple identity tokens whose aud is
// one of the configured client IDs (iOS bundle ID / Services ID).
func NewAppleIDTokenVerifier(audiences []string) AppleTokenVerifier {
	allowed := make([]string, 0, len(audiences))
	for _, id := range audiences {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	return &appleIDTokenVerifier{
		audiences: allowed,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (v *appleIDTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*AppleIdentity, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, errors.NewUnauthorized("Invalid Apple token", nil)
	}
	if len(v.audiences) == 0 {
		return nil, errors.NewInternal("Apple sign-in is not configured", nil)
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithAudience(v.audiences...),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(time.Minute),
	)

	claims := &appleJWTClaims{}
	token, err := parser.ParseWithClaims(idToken, claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("missing kid")
		}
		return v.lookupKey(ctx, kid)
	})
	if err != nil || token == nil || !token.Valid {
		return nil, errors.NewUnauthorized("Invalid Apple token", err)
	}
	if claims.Issuer != appleIssuer {
		return nil, errors.NewUnauthorized("Invalid Apple token", nil)
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return nil, errors.NewUnauthorized("Invalid Apple token", nil)
	}

	email := normalizeEmail(claims.Email)
	if email != "" && appleEmailRejected(claims.EmailVerified) {
		return nil, errors.NewUnauthorized("Apple account email is not verified", nil)
	}

	return &AppleIdentity{
		Subject: claims.Subject,
		Email:   email,
	}, nil
}

func (v *appleIDTokenVerifier) lookupKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if key := v.cachedKey(kid); key != nil {
		return key, nil
	}
	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}
	if key := v.cachedKey(kid); key != nil {
		return key, nil
	}
	return nil, fmt.Errorf("unknown signing key")
}

func (v *appleIDTokenVerifier) cachedKey(kid string) *rsa.PublicKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if time.Now().After(v.keysUntil) {
		return nil
	}
	return v.keys[kid]
}

func (v *appleIDTokenVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleCertsURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apple certs: %s", resp.Status)
	}

	var jwks appleJWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kid == "" || k.Kty != "RSA" {
			continue
		}
		pub, err := rsaPublicKeyFromJWK(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return fmt.Errorf("apple certs: no rsa keys")
	}

	v.mu.Lock()
	v.keys = keys
	v.keysUntil = time.Now().Add(time.Hour)
	v.mu.Unlock()
	return nil
}

func appleEmailRejected(value any) bool {
	switch v := value.(type) {
	case bool:
		return !v
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return false
		}
		return !strings.EqualFold(s, "true")
	default:
		return false
	}
}
