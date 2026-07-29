package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"clap/internal/shared/errors"
	"clap/internal/shared/redis"

	goredis "github.com/redis/go-redis/v9"
)

// OTPRecord is the persisted state of an outstanding one-time code.
type OTPRecord struct {
	CodeHash string `json:"code_hash"`
	Attempts int    `json:"attempts"`
}

// OTPStore persists outstanding OTP codes with a TTL plus a per-email resend
// cooldown. Implementations must be safe for concurrent use.
type OTPStore interface {
	// Save stores (or replaces) the pending code for an email.
	Save(ctx context.Context, email string, rec OTPRecord, ttl time.Duration) error
	// Get returns the pending record, or nil when no code is outstanding.
	Get(ctx context.Context, email string) (*OTPRecord, error)
	// IncrementAttempts bumps the failed-verify counter and returns the new value.
	IncrementAttempts(ctx context.Context, email string) (int, error)
	// Delete removes the pending code (after success or lockout).
	Delete(ctx context.Context, email string) error
	// SetCooldown starts the resend cooldown window for an email.
	SetCooldown(ctx context.Context, email string, ttl time.Duration) error
	// CooldownRemaining returns the remaining cooldown, or 0 when none is active.
	CooldownRemaining(ctx context.Context, email string) (time.Duration, error)
}

// ─── Redis-backed store (production) ─────────────────────────────────────────

type redisOTPStore struct{}

// NewRedisOTPStore returns an OTPStore backed by the shared Redis client.
// Codes expire natively via TTL and are shared across API instances.
func NewRedisOTPStore() OTPStore {
	return &redisOTPStore{}
}

func otpKey(email string) string      { return "otp:code:" + strings.ToLower(email) }
func otpCooldown(email string) string { return "otp:cooldown:" + strings.ToLower(email) }

func (s *redisOTPStore) Save(ctx context.Context, email string, rec OTPRecord, ttl time.Duration) error {
	data, err := json.Marshal(rec)
	if err != nil {
		return errors.NewInternal("Failed to encode OTP record", err)
	}
	if err := redis.GetClient().Set(ctx, otpKey(email), data, ttl).Err(); err != nil {
		return errors.NewInternal("Failed to store OTP", err)
	}
	return nil
}

func (s *redisOTPStore) Get(ctx context.Context, email string) (*OTPRecord, error) {
	raw, err := redis.GetClient().Get(ctx, otpKey(email)).Result()
	if err == goredis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, errors.NewInternal("Failed to read OTP", err)
	}
	var rec OTPRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return nil, errors.NewInternal("Failed to decode OTP record", err)
	}
	return &rec, nil
}

func (s *redisOTPStore) IncrementAttempts(ctx context.Context, email string) (int, error) {
	rec, err := s.Get(ctx, email)
	if err != nil {
		return 0, err
	}
	if rec == nil {
		return 0, nil
	}
	rec.Attempts++
	ttl, err := redis.GetClient().TTL(ctx, otpKey(email)).Result()
	if err != nil || ttl <= 0 {
		ttl = time.Minute
	}
	if err := s.Save(ctx, email, *rec, ttl); err != nil {
		return 0, err
	}
	return rec.Attempts, nil
}

func (s *redisOTPStore) Delete(ctx context.Context, email string) error {
	if err := redis.GetClient().Del(ctx, otpKey(email)).Err(); err != nil {
		return errors.NewInternal("Failed to delete OTP", err)
	}
	return nil
}

func (s *redisOTPStore) SetCooldown(ctx context.Context, email string, ttl time.Duration) error {
	if err := redis.GetClient().Set(ctx, otpCooldown(email), "1", ttl).Err(); err != nil {
		return errors.NewInternal("Failed to set OTP cooldown", err)
	}
	return nil
}

func (s *redisOTPStore) CooldownRemaining(ctx context.Context, email string) (time.Duration, error) {
	ttl, err := redis.GetClient().TTL(ctx, otpCooldown(email)).Result()
	if err != nil {
		return 0, errors.NewInternal("Failed to read OTP cooldown", err)
	}
	if ttl <= 0 {
		return 0, nil
	}
	return ttl, nil
}

// ─── In-memory store (tests / single-node fallback) ──────────────────────────

type memoryOTPEntry struct {
	rec       OTPRecord
	expiresAt time.Time
}

type memoryOTPStore struct {
	mu        sync.Mutex
	codes     map[string]memoryOTPEntry
	cooldowns map[string]time.Time
}

// NewMemoryOTPStore returns a process-local OTPStore. Intended for tests and
// environments without Redis; codes are lost on restart.
func NewMemoryOTPStore() OTPStore {
	return &memoryOTPStore{
		codes:     make(map[string]memoryOTPEntry),
		cooldowns: make(map[string]time.Time),
	}
}

func (s *memoryOTPStore) Save(_ context.Context, email string, rec OTPRecord, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[strings.ToLower(email)] = memoryOTPEntry{rec: rec, expiresAt: time.Now().Add(ttl)}
	return nil
}

func (s *memoryOTPStore) Get(_ context.Context, email string) (*OTPRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.codes[strings.ToLower(email)]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(s.codes, strings.ToLower(email))
		return nil, nil
	}
	rec := entry.rec
	return &rec, nil
}

func (s *memoryOTPStore) IncrementAttempts(_ context.Context, email string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.ToLower(email)
	entry, ok := s.codes[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return 0, nil
	}
	entry.rec.Attempts++
	s.codes[key] = entry
	return entry.rec.Attempts, nil
}

func (s *memoryOTPStore) Delete(_ context.Context, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.codes, strings.ToLower(email))
	return nil
}

func (s *memoryOTPStore) SetCooldown(_ context.Context, email string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cooldowns[strings.ToLower(email)] = time.Now().Add(ttl)
	return nil
}

func (s *memoryOTPStore) CooldownRemaining(_ context.Context, email string) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.cooldowns[strings.ToLower(email)]
	if !ok {
		return 0, nil
	}
	remaining := time.Until(until)
	if remaining <= 0 {
		delete(s.cooldowns, strings.ToLower(email))
		return 0, nil
	}
	return remaining, nil
}
