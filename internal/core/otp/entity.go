package otp

import "time"

type Status string

const (
	StatusCreated   Status = "created"
	StatusValidated Status = "validated"
	StatusExpired   Status = "expired"
)

type Record struct {
	ID             int64
	UserID         string
	Code           string
	Status         Status
	FailedAttempts int
	ExpiresAt      time.Time
	ValidatedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (r Record) IsExpired(now time.Time) bool {
	return !now.Before(r.ExpiresAt)
}
