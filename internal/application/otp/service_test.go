package otpapp

import (
	"context"
	"testing"
	"time"

	coreotp "otp-pair-code-interview/internal/core/otp"
	"otp-pair-code-interview/internal/ports"
)

type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time {
	return f.now
}

type fakeGenerator struct {
	values []string
	index  int
}

func (f *fakeGenerator) Generate(context.Context) (string, error) {
	value := f.values[f.index]
	f.index++
	return value, nil
}

type fakeRepo struct {
	nextID  int64
	current map[string]*coreotp.Record
	byID    map[int64]*coreotp.Record
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		nextID:  1,
		current: map[string]*coreotp.Record{},
		byID:    map[int64]*coreotp.Record{},
	}
}

func (f *fakeRepo) GetLatestCreatedByUserID(_ context.Context, userID string) (*coreotp.Record, error) {
	record := f.current[userID]
	if record == nil || record.Status != coreotp.StatusCreated {
		return nil, nil
	}
	return record, nil
}

func (f *fakeRepo) Create(_ context.Context, params ports.CreateOTPParams) (*coreotp.Record, error) {
	record := &coreotp.Record{
		ID:             f.nextID,
		UserID:         params.UserID,
		Code:           params.Code,
		Status:         coreotp.StatusCreated,
		FailedAttempts: 0,
		ExpiresAt:      params.ExpiresAt,
		CreatedAt:      params.CreatedAt,
		UpdatedAt:      params.UpdatedAt,
	}
	f.nextID++
	f.current[record.UserID] = record
	f.byID[record.ID] = record
	return record, nil
}

func (f *fakeRepo) MarkExpired(_ context.Context, id int64, updatedAt time.Time) error {
	record := f.byID[id]
	record.Status = coreotp.StatusExpired
	record.UpdatedAt = updatedAt
	delete(f.current, record.UserID)
	return nil
}

func (f *fakeRepo) IncrementFailedAttempts(_ context.Context, id int64, updatedAt time.Time) (int, error) {
	record := f.byID[id]
	record.FailedAttempts++
	record.UpdatedAt = updatedAt
	return record.FailedAttempts, nil
}

func (f *fakeRepo) MarkValidated(_ context.Context, id int64, validatedAt time.Time) error {
	record := f.byID[id]
	record.Status = coreotp.StatusValidated
	record.ValidatedAt = &validatedAt
	record.UpdatedAt = validatedAt
	delete(f.current, record.UserID)
	return nil
}

func TestRequestOTPSuccess(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, fakeClock{now: time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)}, &fakeGenerator{values: []string{"12345"}}, Config{})

	result, err := service.RequestOTP(context.Background(), "Robert")
	if err != nil {
		t.Fatalf("RequestOTP returned error: %v", err)
	}
	if result.OTP != "12345" {
		t.Fatalf("expected generated OTP 12345, got %s", result.OTP)
	}
}

func TestRequestOTPRejectsActiveOTPWhenReissueDisabled(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	_, _ = repo.Create(context.Background(), ports.CreateOTPParams{UserID: "Robert", Code: "12345", ExpiresAt: now.Add(time.Minute), CreatedAt: now, UpdatedAt: now})
	service := NewService(repo, fakeClock{now: now}, &fakeGenerator{values: []string{"54321"}}, Config{AllowReissue: false})

	_, err := service.RequestOTP(context.Background(), "Robert")
	if err != coreotp.ErrAlreadyActive {
		t.Fatalf("expected ErrAlreadyActive, got %v", err)
	}
}

func TestRequestOTPReissuesWhenEnabled(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	oldRecord, _ := repo.Create(context.Background(), ports.CreateOTPParams{UserID: "Robert", Code: "12345", ExpiresAt: now.Add(time.Minute), CreatedAt: now, UpdatedAt: now})
	service := NewService(repo, fakeClock{now: now}, &fakeGenerator{values: []string{"54321"}}, Config{AllowReissue: true})

	result, err := service.RequestOTP(context.Background(), "Robert")
	if err != nil {
		t.Fatalf("RequestOTP returned error: %v", err)
	}
	if result.OTP != "54321" {
		t.Fatalf("expected new OTP 54321, got %s", result.OTP)
	}
	if oldRecord.Status != coreotp.StatusExpired {
		t.Fatalf("expected old OTP to be expired, got %s", oldRecord.Status)
	}
}

func TestValidateOTPSuccess(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	record, _ := repo.Create(context.Background(), ports.CreateOTPParams{UserID: "Robert", Code: "12345", ExpiresAt: now.Add(time.Minute), CreatedAt: now, UpdatedAt: now})
	service := NewService(repo, fakeClock{now: now}, &fakeGenerator{values: []string{"00000"}}, Config{MaxFailedAttempts: 5})

	result, err := service.ValidateOTP(context.Background(), "Robert", "12345")
	if err != nil {
		t.Fatalf("ValidateOTP returned error: %v", err)
	}
	if result.UserID != "Robert" {
		t.Fatalf("expected user Robert, got %s", result.UserID)
	}
	if record.Status != coreotp.StatusValidated {
		t.Fatalf("expected validated status, got %s", record.Status)
	}
}

func TestValidateOTPNotFound(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, fakeClock{now: time.Now().UTC()}, &fakeGenerator{values: []string{"00000"}}, Config{})

	_, err := service.ValidateOTP(context.Background(), "Robert", "12345")
	if err != coreotp.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestValidateOTPExpired(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	record, _ := repo.Create(context.Background(), ports.CreateOTPParams{UserID: "Robert", Code: "12345", ExpiresAt: now.Add(-time.Second), CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)})
	service := NewService(repo, fakeClock{now: now}, &fakeGenerator{values: []string{"00000"}}, Config{})

	_, err := service.ValidateOTP(context.Background(), "Robert", "12345")
	if err != coreotp.ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
	if record.Status != coreotp.StatusExpired {
		t.Fatalf("expected expired status, got %s", record.Status)
	}
}

func TestValidateOTPWrongCode(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	record, _ := repo.Create(context.Background(), ports.CreateOTPParams{UserID: "Robert", Code: "12345", ExpiresAt: now.Add(time.Minute), CreatedAt: now, UpdatedAt: now})
	service := NewService(repo, fakeClock{now: now}, &fakeGenerator{values: []string{"00000"}}, Config{MaxFailedAttempts: 5})

	_, err := service.ValidateOTP(context.Background(), "Robert", "99999")
	if err != coreotp.ErrInvalidCode {
		t.Fatalf("expected ErrInvalidCode, got %v", err)
	}
	if record.FailedAttempts != 1 {
		t.Fatalf("expected failed attempts 1, got %d", record.FailedAttempts)
	}
}

func TestValidateOTPBlocksAfterConfiguredAttempts(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	record, _ := repo.Create(context.Background(), ports.CreateOTPParams{UserID: "Robert", Code: "12345", ExpiresAt: now.Add(time.Minute), CreatedAt: now, UpdatedAt: now})
	service := NewService(repo, fakeClock{now: now}, &fakeGenerator{values: []string{"00000"}}, Config{MaxFailedAttempts: 2})

	_, err := service.ValidateOTP(context.Background(), "Robert", "99999")
	if err != coreotp.ErrInvalidCode {
		t.Fatalf("expected first error to be ErrInvalidCode, got %v", err)
	}
	_, err = service.ValidateOTP(context.Background(), "Robert", "99999")
	if err != coreotp.ErrBlocked {
		t.Fatalf("expected second error to be ErrBlocked, got %v", err)
	}
	if record.Status != coreotp.StatusExpired {
		t.Fatalf("expected blocked OTP to be persisted as expired, got %s", record.Status)
	}
}
