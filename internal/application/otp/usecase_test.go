package otpapp

import (
	"context"
	"errors"
	"testing"
	"time"

	coreotp "otp-pair-code/internal/core/otp"
	"otp-pair-code/internal/ports"
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

type errGenerator struct {
	err error
}

func (g errGenerator) Generate(context.Context) (string, error) {
	return "", g.err
}

type stubRepo struct {
	getLatestFunc               func(context.Context, string) (*coreotp.Record, error)
	createFunc                  func(context.Context, ports.CreateOTPParams) (*coreotp.Record, error)
	markExpiredFunc             func(context.Context, int64, time.Time) error
	incrementFailedAttemptsFunc func(context.Context, int64, time.Time) (int, error)
	markValidatedFunc           func(context.Context, int64, time.Time) error
}

func (s stubRepo) GetLatestCreatedByUserID(ctx context.Context, userID string) (*coreotp.Record, error) {
	if s.getLatestFunc == nil {
		return nil, nil
	}
	return s.getLatestFunc(ctx, userID)
}

func (s stubRepo) Create(ctx context.Context, params ports.CreateOTPParams) (*coreotp.Record, error) {
	if s.createFunc == nil {
		return nil, nil
	}
	return s.createFunc(ctx, params)
}

func (s stubRepo) MarkExpired(ctx context.Context, id int64, updatedAt time.Time) error {
	if s.markExpiredFunc == nil {
		return nil
	}
	return s.markExpiredFunc(ctx, id, updatedAt)
}

func (s stubRepo) IncrementFailedAttempts(ctx context.Context, id int64, updatedAt time.Time) (int, error) {
	if s.incrementFailedAttemptsFunc == nil {
		return 0, nil
	}
	return s.incrementFailedAttemptsFunc(ctx, id, updatedAt)
}

func (s stubRepo) MarkValidated(ctx context.Context, id int64, validatedAt time.Time) error {
	if s.markValidatedFunc == nil {
		return nil
	}
	return s.markValidatedFunc(ctx, id, validatedAt)
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

func TestRequestOTPRejectsBlankUserID(t *testing.T) {
	service := NewService(stubRepo{}, fakeClock{now: time.Now().UTC()}, &fakeGenerator{values: []string{"12345"}}, Config{})

	_, err := service.RequestOTP(context.Background(), "   ")
	if err != coreotp.ErrInvalidUserID {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestRequestOTPWrapsGenerationError(t *testing.T) {
	genErr := errors.New("generator down")
	service := NewService(stubRepo{}, fakeClock{now: time.Now().UTC()}, errGenerator{err: genErr}, Config{})

	_, err := service.RequestOTP(context.Background(), "Robert")
	if !errors.Is(err, coreotp.ErrGenerationFailed) {
		t.Fatalf("expected joined error to contain ErrGenerationFailed, got %v", err)
	}
	if !errors.Is(err, genErr) {
		t.Fatalf("expected joined error to contain generator error, got %v", err)
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

func TestRequestOTPExpiresPreviouslyExpiredRecordBeforeCreatingNewOne(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	oldRecord, _ := repo.Create(context.Background(), ports.CreateOTPParams{UserID: "Robert", Code: "12345", ExpiresAt: now.Add(-time.Second), CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)})
	service := NewService(repo, fakeClock{now: now}, &fakeGenerator{values: []string{"54321"}}, Config{})

	result, err := service.RequestOTP(context.Background(), "Robert")
	if err != nil {
		t.Fatalf("RequestOTP returned error: %v", err)
	}
	if result.OTP != "54321" {
		t.Fatalf("expected new OTP 54321, got %s", result.OTP)
	}
	if oldRecord.Status != coreotp.StatusExpired {
		t.Fatalf("expected expired previous record, got %s", oldRecord.Status)
	}
}

func TestRequestOTPPropagatesRepositoryErrors(t *testing.T) {
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	record := &coreotp.Record{ID: 10, UserID: "Robert", Code: "12345", Status: coreotp.StatusCreated, ExpiresAt: now.Add(time.Minute)}

	tests := []struct {
		name string
		repo ports.OTPRepository
	}{
		{
			name: "get latest",
			repo: stubRepo{getLatestFunc: func(context.Context, string) (*coreotp.Record, error) {
				return nil, errors.New("db read failed")
			}},
		},
		{
			name: "expire active before reissue",
			repo: stubRepo{
				getLatestFunc:   func(context.Context, string) (*coreotp.Record, error) { return record, nil },
				markExpiredFunc: func(context.Context, int64, time.Time) error { return errors.New("expire failed") },
			},
		},
		{
			name: "expire stale record",
			repo: stubRepo{
				getLatestFunc: func(context.Context, string) (*coreotp.Record, error) {
					return &coreotp.Record{ID: 11, UserID: "Robert", Code: "12345", ExpiresAt: now.Add(-time.Second)}, nil
				},
				markExpiredFunc: func(context.Context, int64, time.Time) error { return errors.New("expire failed") },
			},
		},
		{
			name: "create",
			repo: stubRepo{
				getLatestFunc: func(context.Context, string) (*coreotp.Record, error) { return nil, nil },
				createFunc: func(context.Context, ports.CreateOTPParams) (*coreotp.Record, error) {
					return nil, errors.New("insert failed")
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.repo, fakeClock{now: now}, &fakeGenerator{values: []string{"54321"}}, Config{AllowReissue: true})

			_, err := service.RequestOTP(context.Background(), "Robert")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
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

func TestValidateOTPRejectsBlankInputs(t *testing.T) {
	service := NewService(stubRepo{}, fakeClock{now: time.Now().UTC()}, &fakeGenerator{values: []string{"12345"}}, Config{})

	tests := []struct {
		name          string
		userID        string
		otp           string
		expectedError error
	}{
		{name: "blank user id", userID: "   ", otp: "12345", expectedError: coreotp.ErrInvalidUserID},
		{name: "blank otp", userID: "Robert", otp: "   ", expectedError: coreotp.ErrInvalidOTPInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateOTP(context.Background(), tt.userID, tt.otp)
			if err != tt.expectedError {
				t.Fatalf("expected %v, got %v", tt.expectedError, err)
			}
		})
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

func TestValidateOTPPropagatesRepositoryErrors(t *testing.T) {
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		cfg  Config
		repo ports.OTPRepository
	}{
		{
			name: "get latest",
			repo: stubRepo{getLatestFunc: func(context.Context, string) (*coreotp.Record, error) {
				return nil, errors.New("db read failed")
			}},
		},
		{
			name: "mark expired on expired record",
			repo: stubRepo{
				getLatestFunc: func(context.Context, string) (*coreotp.Record, error) {
					return &coreotp.Record{ID: 1, UserID: "Robert", Code: "12345", ExpiresAt: now.Add(-time.Second)}, nil
				},
				markExpiredFunc: func(context.Context, int64, time.Time) error { return errors.New("expire failed") },
			},
		},
		{
			name: "increment failed attempts",
			repo: stubRepo{
				getLatestFunc: func(context.Context, string) (*coreotp.Record, error) {
					return &coreotp.Record{ID: 1, UserID: "Robert", Code: "12345", ExpiresAt: now.Add(time.Minute)}, nil
				},
				incrementFailedAttemptsFunc: func(context.Context, int64, time.Time) (int, error) {
					return 0, errors.New("increment failed")
				},
			},
		},
		{
			name: "mark expired on blocked record",
			cfg:  Config{MaxFailedAttempts: 1},
			repo: stubRepo{
				getLatestFunc: func(context.Context, string) (*coreotp.Record, error) {
					return &coreotp.Record{ID: 1, UserID: "Robert", Code: "12345", ExpiresAt: now.Add(time.Minute)}, nil
				},
				incrementFailedAttemptsFunc: func(context.Context, int64, time.Time) (int, error) {
					return 1, nil
				},
				markExpiredFunc: func(context.Context, int64, time.Time) error { return errors.New("expire failed") },
			},
		},
		{
			name: "mark validated",
			repo: stubRepo{
				getLatestFunc: func(context.Context, string) (*coreotp.Record, error) {
					return &coreotp.Record{ID: 1, UserID: "Robert", Code: "12345", ExpiresAt: now.Add(time.Minute)}, nil
				},
				markValidatedFunc: func(context.Context, int64, time.Time) error { return errors.New("validate failed") },
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.repo, fakeClock{now: now}, &fakeGenerator{values: []string{"00000"}}, tt.cfg)

			_, err := service.ValidateOTP(context.Background(), "Robert", "99999")
			if tt.name == "mark validated" {
				_, err = service.ValidateOTP(context.Background(), "Robert", "12345")
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
