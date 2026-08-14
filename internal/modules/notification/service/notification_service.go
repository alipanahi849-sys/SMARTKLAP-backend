package service

import (
	"context"
	"strings"
	"time"

	"clap/internal/modules/notification/dto"
	"clap/internal/modules/notification/models"
	"clap/internal/modules/notification/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

const (
	minFCMTokenLen = 20
	maxFCMTokenLen = 4096
)

type NotificationService interface {
	RegisterDevice(ctx context.Context, userID uuid.UUID, req *dto.RegisterDeviceRequest) (*dto.DeviceResponse, error)
	UnregisterDevice(ctx context.Context, userID uuid.UUID, req *dto.UnregisterDeviceRequest) error
	NotifyChantCountdown(ctx context.Context, chantID, matchID, title string, startsAt time.Time, leadTime time.Duration) error
}

type notificationService struct {
	devices repository.PushDeviceRepository
	sender  MulticastSender
}

func NewNotificationService(devices repository.PushDeviceRepository, sender MulticastSender) NotificationService {
	return &notificationService{devices: devices, sender: sender}
}

func (s *notificationService) RegisterDevice(ctx context.Context, userID uuid.UUID, req *dto.RegisterDeviceRequest) (*dto.DeviceResponse, error) {
	token, err := normalizeFCMToken(req.FCMToken)
	if err != nil {
		return nil, err
	}
	platform, err := normalizePlatform(req.Platform)
	if err != nil {
		return nil, err
	}

	stored, err := s.devices.Upsert(ctx, &models.PushDevice{
		UserID:   userID,
		FCMToken: token,
		Platform: platform,
	})
	if err != nil {
		return nil, err
	}

	return &dto.DeviceResponse{
		ID:        stored.ID,
		FCMToken:  stored.FCMToken,
		Platform:  stored.Platform,
		UpdatedAt: stored.UpdatedAt,
	}, nil
}

func (s *notificationService) UnregisterDevice(ctx context.Context, userID uuid.UUID, req *dto.UnregisterDeviceRequest) error {
	token, err := normalizeFCMToken(req.FCMToken)
	if err != nil {
		return err
	}
	return s.devices.DeleteByUserAndToken(ctx, userID, token)
}

func (s *notificationService) NotifyChantCountdown(
	ctx context.Context,
	chantID, matchID, title string,
	startsAt time.Time,
	leadTime time.Duration,
) error {
	if s.sender == nil {
		return nil
	}

	tokens, err := s.devices.ListTokens(ctx)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		logger.Info().
			Str("chant_id", chantID).
			Msg("push: no registered devices for chant countdown")
		return nil
	}

	minutes := countdownMinutes(startsAt, leadTime)
	notifTitle, body := chantCountdownCopy(title, minutes)
	unregistered, sendErr := s.sender.Send(ctx, tokens, PushMessage{
		Title: notifTitle,
		Body:  body,
		Data: map[string]string{
			"type":      chantCountdownType,
			"path":      chantCountdownPath,
			"chant_id":  chantID,
			"match_id":  matchID,
			"title":     title,
			"starts_at": startsAt.UTC().Format(time.RFC3339),
		},
	})
	if len(unregistered) > 0 {
		if delErr := s.devices.DeleteByTokens(ctx, unregistered); delErr != nil {
			logger.Warn().Err(delErr).Int("count", len(unregistered)).Msg("push: failed to drop stale FCM tokens")
		} else {
			logger.Info().Int("count", len(unregistered)).Msg("push: dropped stale FCM tokens")
		}
	}
	if sendErr != nil {
		return sendErr
	}

	logger.Info().
		Str("chant_id", chantID).
		Int("devices", len(tokens)-len(unregistered)).
		Int("minutes", minutes).
		Msg("push: chant countdown notification sent")
	return nil
}

func normalizeFCMToken(raw string) (string, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return "", errors.NewBadRequest("fcm_token is required", nil)
	}
	if len(token) < minFCMTokenLen || len(token) > maxFCMTokenLen {
		return "", errors.NewBadRequest("Invalid fcm_token", nil)
	}
	return token, nil
}

func normalizePlatform(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case models.PlatformIOS:
		return models.PlatformIOS, nil
	case models.PlatformAndroid:
		return models.PlatformAndroid, nil
	default:
		return "", errors.NewBadRequest("platform must be ios or android", nil)
	}
}
