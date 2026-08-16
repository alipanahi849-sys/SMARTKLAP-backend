package service

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"clap/internal/shared/config"
	"clap/internal/shared/logger"

	"github.com/golang-jwt/jwt/v5"
)

const (
	fcmScope             = "https://www.googleapis.com/auth/firebase.messaging"
	fcmTokenURL          = "https://oauth2.googleapis.com/token"
	fcmSendURLFmt        = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
	androidChannelID     = "default"
	chantCountdownType   = "chant_countdown"
	chantCountdownPath   = "/chant/CountDownScreen"
	fcmSendConcurrency   = 8
	googleJWTBearerGrant = "urn:ietf:params:oauth:grant-type:jwt-bearer"
)

var errUnregistered = errors.New("fcm token unregistered")

// PushMessage is a notification shown by the OS, plus string-only FCM data.
type PushMessage struct {
	Title string
	Body  string
	Data  map[string]string
}

// MulticastSender delivers one notification to many FCM tokens.
type MulticastSender interface {
	Send(ctx context.Context, tokens []string, msg PushMessage) (unregistered []string, err error)
}

type serviceAccountFile struct {
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

type fcmSender struct {
	projectID   string
	clientEmail string
	privateKey  *rsa.PrivateKey
	http        *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewFirebaseSender builds an FCM HTTP v1 client from Admin SDK credentials.
// Returns (nil, nil) when Firebase is not configured so callers can skip sending.
func NewFirebaseSender(ctx context.Context, cfg config.Firebase) (MulticastSender, error) {
	if !cfg.Enabled() {
		return nil, nil
	}

	raw, err := loadServiceAccountJSON(cfg)
	if err != nil {
		return nil, err
	}
	account, key, err := parseServiceAccount(raw)
	if err != nil {
		return nil, err
	}

	return &fcmSender{
		projectID:   account.ProjectID,
		clientEmail: account.ClientEmail,
		privateKey:  key,
		http:        &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func loadServiceAccountJSON(cfg config.Firebase) ([]byte, error) {
	if json := strings.TrimSpace(cfg.CredentialsJSON); json != "" {
		return []byte(json), nil
	}
	file := strings.TrimSpace(cfg.CredentialsFile)
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read firebase credentials: %w", err)
	}
	return raw, nil
}

func parseServiceAccount(raw []byte) (*serviceAccountFile, *rsa.PrivateKey, error) {
	var account serviceAccountFile
	if err := json.Unmarshal(raw, &account); err != nil {
		return nil, nil, fmt.Errorf("parse firebase credentials: %w", err)
	}
	if strings.TrimSpace(account.ProjectID) == "" || strings.TrimSpace(account.ClientEmail) == "" {
		return nil, nil, fmt.Errorf("firebase credentials missing project_id or client_email")
	}
	block, _ := pem.Decode([]byte(account.PrivateKey))
	if block == nil {
		return nil, nil, fmt.Errorf("firebase credentials private_key is not PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		parsed, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("firebase credentials private_key: %w", err)
		}
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("firebase credentials private_key is not RSA")
	}
	return &account, key, nil
}

func (s *fcmSender) Send(ctx context.Context, tokens []string, msg PushMessage) ([]string, error) {
	if s == nil || len(tokens) == 0 {
		return nil, nil
	}

	sem := make(chan struct{}, fcmSendConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var unregistered []string
	var firstErr error
	succeeded := 0

	for _, token := range tokens {
		token := token
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			err := s.sendOne(ctx, token, msg)
			if err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
				return
			}
			if errors.Is(err, errUnregistered) {
				mu.Lock()
				unregistered = append(unregistered, token)
				mu.Unlock()
				return
			}
			logger.Warn().Err(err).Msg("fcm: token send failed")
			mu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	// One delivered device must not retry the whole fan-out: a single iOS
	// token error was re-notifying every Android device every 5 seconds.
	if succeeded > 0 {
		return unregistered, nil
	}
	return unregistered, firstErr
}

func (s *fcmSender) sendOne(ctx context.Context, token string, msg PushMessage) error {
	accessToken, err := s.bearerToken(ctx)
	if err != nil {
		return err
	}

	body, err := json.Marshal(buildFCMMessage(token, msg))
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(fcmSendURLFmt, s.projectID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if isUnregisteredResponse(resp.StatusCode, respBody) {
		return errUnregistered
	}
	return fmt.Errorf("fcm send: %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
}

func (s *fcmSender) bearerToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accessToken != "" && time.Now().Add(time.Minute).Before(s.tokenExpiry) {
		return s.accessToken, nil
	}

	now := time.Now()
	assertion := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   s.clientEmail,
		"scope": fcmScope,
		"aud":   fcmTokenURL,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	})
	signed, err := assertion.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("fcm jwt: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", googleJWTBearerGrant)
	form.Set("assertion", signed)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fcmTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("fcm oauth: %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var token struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &token); err != nil {
		return "", fmt.Errorf("fcm oauth parse: %w", err)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("fcm oauth: empty access_token")
	}
	expiry := time.Hour
	if token.ExpiresIn > 0 {
		expiry = time.Duration(token.ExpiresIn) * time.Second
	}
	s.accessToken = token.AccessToken
	s.tokenExpiry = time.Now().Add(expiry)
	return s.accessToken, nil
}

func buildFCMMessage(token string, msg PushMessage) map[string]any {
	collapseKey := pushCollapseKey(msg)

	androidNotification := map[string]any{
		"channel_id": androidChannelID,
		"sound":      "default",
	}
	android := map[string]any{
		"priority":     "HIGH",
		"notification": androidNotification,
	}

	apnsHeaders := map[string]string{
		"apns-priority":  "10",
		"apns-push-type": "alert",
	}
	if collapseKey != "" {
		android["collapse_key"] = collapseKey
		androidNotification["tag"] = collapseKey
		apnsHeaders["apns-collapse-id"] = collapseKey
	}

	message := map[string]any{
		"token": token,
		"notification": map[string]string{
			"title": msg.Title,
			"body":  msg.Body,
		},
		"android": android,
		"apns": map[string]any{
			"headers": apnsHeaders,
			"payload": map[string]any{
				// A custom aps payload overrides FCM's title/body. Without
				// alert, iOS delivers a sound-only push and shows nothing.
				"aps": map[string]any{
					"alert": map[string]string{
						"title": msg.Title,
						"body":  msg.Body,
					},
					"sound": "default",
				},
			},
		},
	}
	if len(msg.Data) > 0 {
		message["data"] = msg.Data
	}
	return map[string]any{"message": message}
}

func pushCollapseKey(msg PushMessage) string {
	chantID := strings.TrimSpace(msg.Data["chant_id"])
	if chantID == "" {
		return ""
	}
	key := "chant-" + chantID
	if len(key) > 64 {
		return key[:64]
	}
	return key
}

func isUnregisteredResponse(status int, body []byte) bool {
	if status != http.StatusNotFound && status != http.StatusBadRequest {
		return false
	}
	text := string(body)
	return strings.Contains(text, "UNREGISTERED") || strings.Contains(text, "NOT_FOUND")
}

func chantCountdownCopy(title string, minutes int) (string, string) {
	name := strings.TrimSpace(title)
	if name == "" {
		name = "SMARTKLAP"
	}
	if minutes < 1 {
		minutes = 1
	}
	unit := "minutes"
	if minutes == 1 {
		unit = "minute"
	}
	return name, fmt.Sprintf("Chant countdown has started. It begins in %d %s.", minutes, unit)
}

func countdownMinutes(startsAt time.Time, leadTime time.Duration) int {
	remaining := time.Until(startsAt)
	if remaining > 0 {
		minutes := int((remaining + time.Minute - 1) / time.Minute)
		if minutes < 1 {
			return 1
		}
		return minutes
	}
	minutes := int((leadTime + time.Minute - 1) / time.Minute)
	if minutes < 1 {
		return 2
	}
	return minutes
}
