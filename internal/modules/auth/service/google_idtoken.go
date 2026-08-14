package service

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"clap/internal/shared/config"
	"clap/internal/shared/errors"

	"github.com/golang-jwt/jwt/v5"
)

const googleCertsURL = "https://www.googleapis.com/oauth2/v3/certs"

var googleIssuers = []string{
	"https://accounts.google.com",
	"accounts.google.com",
}

// GoogleIdentity is the verified identity extracted from a Google ID token.
type GoogleIdentity struct {
	Subject string
	Email   string
	Name    string
	Given   string
	Family  string
}

// GoogleTokenVerifier verifies a Google Sign-In ID token.
type GoogleTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*GoogleIdentity, error)
}

type googleIDTokenVerifier struct {
	audiences  []string
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	keysUntil time.Time
}

type googleJWTClaims struct {
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	jwt.RegisteredClaims
}

type googleJWKS struct {
	Keys []googleJWK `json:"keys"`
}

type googleJWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func googleVerifierFromConfig() GoogleTokenVerifier {
	if config.AppConfig == nil {
		return nil
	}
	ids := config.AppConfig.Google.ClientIDs
	if len(ids) == 0 {
		return nil
	}
	return NewGoogleIDTokenVerifier(ids)
}

// NewGoogleIDTokenVerifier verifies RS256 Google ID tokens whose aud is one of
// the configured OAuth client IDs (web / iOS / Android).
func NewGoogleIDTokenVerifier(audiences []string) GoogleTokenVerifier {
	allowed := make([]string, 0, len(audiences))
	for _, id := range audiences {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	return &googleIDTokenVerifier{
		audiences: allowed,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (v *googleIDTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*GoogleIdentity, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, errors.NewUnauthorized("Invalid Google token", nil)
	}
	if len(v.audiences) == 0 {
		return nil, errors.NewInternal("Google sign-in is not configured", nil)
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithAudience(v.audiences...),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(time.Minute),
	)

	claims := &googleJWTClaims{}
	token, err := parser.ParseWithClaims(idToken, claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("missing kid")
		}
		return v.lookupKey(ctx, kid)
	})
	if err != nil || token == nil || !token.Valid {
		return nil, errors.NewUnauthorized("Invalid Google token", err)
	}
	if !googleIssuerAllowed(claims.Issuer) {
		return nil, errors.NewUnauthorized("Invalid Google token", nil)
	}

	email := normalizeEmail(claims.Email)
	if email == "" || !googleEmailVerified(claims.EmailVerified) {
		return nil, errors.NewUnauthorized("Google account email is not verified", nil)
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return nil, errors.NewUnauthorized("Invalid Google token", nil)
	}

	return &GoogleIdentity{
		Subject: claims.Subject,
		Email:   email,
		Name:    strings.TrimSpace(claims.Name),
		Given:   strings.TrimSpace(claims.GivenName),
		Family:  strings.TrimSpace(claims.FamilyName),
	}, nil
}

func (v *googleIDTokenVerifier) lookupKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
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

func (v *googleIDTokenVerifier) cachedKey(kid string) *rsa.PublicKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if time.Now().After(v.keysUntil) {
		return nil
	}
	return v.keys[kid]
}

func (v *googleIDTokenVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleCertsURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google certs: %s", resp.Status)
	}

	var jwks googleJWKS
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
		return fmt.Errorf("google certs: no rsa keys")
	}

	v.mu.Lock()
	v.keys = keys
	v.keysUntil = time.Now().Add(time.Hour)
	v.mu.Unlock()
	return nil
}

func rsaPublicKeyFromJWK(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	if len(nBytes) == 0 || len(eBytes) == 0 {
		return nil, fmt.Errorf("invalid rsa jwk")
	}

	var eInt int
	for _, b := range eBytes {
		eInt = eInt<<8 + int(b)
	}
	if eInt <= 0 {
		return nil, fmt.Errorf("invalid rsa exponent")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}, nil
}

func googleEmailVerified(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func googleIssuerAllowed(issuer string) bool {
	for _, allowed := range googleIssuers {
		if issuer == allowed {
			return true
		}
	}
	return false
}
