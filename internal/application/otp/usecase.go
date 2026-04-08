package otpapp

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	coreotp "otp-pair-code/internal/core/otp"
	"otp-pair-code/internal/ports"
)

const otpTTL = 2 * time.Minute

type Config struct {
	AllowReissue      bool
	MaxFailedAttempts int
}

type Service struct {
	repo      ports.OTPRepository
	clock     ports.Clock
	generator ports.OTPGenerator
	cfg       Config
}

type RequestResult struct {
	UserID string
	OTP    string
}

type ValidateResult struct {
	UserID string
}

func NewService(repo ports.OTPRepository, clock ports.Clock, generator ports.OTPGenerator, cfg Config) *Service {
	return &Service{
		repo:      repo,
		clock:     clock,
		generator: generator,
		cfg:       cfg,
	}
}

func (s *Service) RequestOTP(ctx context.Context, userID string) (RequestResult, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return RequestResult{}, coreotp.ErrInvalidUserID
	}

	now := s.clock.Now()
	code, err := s.generator.Generate(ctx)
	if err != nil {
		return RequestResult{}, errors.Join(coreotp.ErrGenerationFailed, err)
	}

	current, err := s.repo.GetLatestCreatedByUserID(ctx, userID)
	if err != nil {
		return RequestResult{}, err
	}

	if current != nil {
		if current.IsExpired(now) {
			if err := s.repo.MarkExpired(ctx, current.ID, now); err != nil {
				return RequestResult{}, err
			}
		} else {
			if !s.cfg.AllowReissue {
				return RequestResult{}, coreotp.ErrAlreadyActive
			}

			if err := s.repo.MarkExpired(ctx, current.ID, now); err != nil {
				return RequestResult{}, err
			}
		}
	}

	record, err := s.repo.Create(ctx, ports.CreateOTPParams{
		UserID:    userID,
		Code:      code,
		ExpiresAt: now.Add(otpTTL),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return RequestResult{}, err
	}

	return RequestResult{UserID: record.UserID, OTP: record.Code}, nil
}

func (s *Service) ValidateOTP(ctx context.Context, userID, submittedCode string) (ValidateResult, error) {
	userID = strings.TrimSpace(userID)
	submittedCode = strings.TrimSpace(submittedCode)
	if userID == "" {
		return ValidateResult{}, coreotp.ErrInvalidUserID
	}
	if submittedCode == "" {
		return ValidateResult{}, coreotp.ErrInvalidOTPInput
	}

	now := s.clock.Now()
	record, err := s.repo.GetLatestCreatedByUserID(ctx, userID)
	if err != nil {
		return ValidateResult{}, err
	}
	if record == nil {
		return ValidateResult{}, coreotp.ErrNotFound
	}

	if record.IsExpired(now) {
		if err := s.repo.MarkExpired(ctx, record.ID, now); err != nil {
			return ValidateResult{}, err
		}
		return ValidateResult{}, coreotp.ErrExpired
	}

	if subtle.ConstantTimeCompare([]byte(record.Code), []byte(submittedCode)) != 1 {
		attempts, err := s.repo.IncrementFailedAttempts(ctx, record.ID, now)
		if err != nil {
			return ValidateResult{}, err
		}

		if s.cfg.MaxFailedAttempts > 0 && attempts >= s.cfg.MaxFailedAttempts {
			if err := s.repo.MarkExpired(ctx, record.ID, now); err != nil {
				return ValidateResult{}, err
			}
			return ValidateResult{}, coreotp.ErrBlocked
		}

		return ValidateResult{}, coreotp.ErrInvalidCode
	}

	if err := s.repo.MarkValidated(ctx, record.ID, now); err != nil {
		return ValidateResult{}, err
	}

	return ValidateResult{UserID: record.UserID}, nil
}
