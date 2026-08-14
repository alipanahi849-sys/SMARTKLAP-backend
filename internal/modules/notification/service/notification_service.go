package service

import (
	"context"
	"strings"

	"clap/internal/modules/notification/dto"
	"clap/internal/modules/notification/models"
	"clap/internal/modules/notification/repository"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
)

const (
	minFCMTokenLen = 20
	maxFCMTokenLen = 4096
)

type NotificationService interface {
	RegisterDevice(ctx context.Context, userID uuid.UUID, req *dto.RegisterDeviceRequest) (*dto.DeviceResponse, error)
	UnregisterDevice(ctx context.Context, userID uuid.UUID, req *dto.UnregisterDeviceRequest) error
}

type notificationService struct {
	devices repository.PushDeviceRepository
}

func NewNotificationService(devices repository.PushDeviceRepository) NotificationService {
	return &notificationService{devices: devices}
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
