package ports

import (
	"context"
	"time"

	coreotp "otp-pair-code-interview/internal/core/otp"
)

type CreateOTPParams struct {
	UserID    string
	Code      string
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OTPRepository interface {
	GetLatestCreatedByUserID(ctx context.Context, userID string) (*coreotp.Record, error)
	Create(ctx context.Context, params CreateOTPParams) (*coreotp.Record, error)
	MarkExpired(ctx context.Context, id int64, updatedAt time.Time) error
	IncrementFailedAttempts(ctx context.Context, id int64, updatedAt time.Time) (int, error)
	MarkValidated(ctx context.Context, id int64, validatedAt time.Time) error
}
