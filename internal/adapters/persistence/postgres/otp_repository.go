package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	coreotp "otp-pair-code/internal/core/otp"
	"otp-pair-code/internal/ports"
)

type OTPRepository struct {
	pool *pgxpool.Pool
}

func NewOTPRepository(pool *pgxpool.Pool) *OTPRepository {
	return &OTPRepository{pool: pool}
}

func (r *OTPRepository) GetLatestCreatedByUserID(ctx context.Context, userID string) (*coreotp.Record, error) {
	const query = `
		SELECT id, user_id, otp_code, status, failed_attempts, expires_at, validated_at, created_at, updated_at
		FROM otps
		WHERE user_id = $1 AND status = 'created'
		ORDER BY created_at DESC
		LIMIT 1`

	var record coreotp.Record
	var validatedAt sql.NullTime
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&record.ID,
		&record.UserID,
		&record.Code,
		&record.Status,
		&record.FailedAttempts,
		&record.ExpiresAt,
		&validatedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if validatedAt.Valid {
		t := validatedAt.Time
		record.ValidatedAt = &t
	}

	return &record, nil
}

func (r *OTPRepository) Create(ctx context.Context, params ports.CreateOTPParams) (*coreotp.Record, error) {
	const query = `
		INSERT INTO otps (user_id, otp_code, status, failed_attempts, expires_at, created_at, updated_at)
		VALUES ($1, $2, 'created', 0, $3, $4, $5)
		RETURNING id, user_id, otp_code, status, failed_attempts, expires_at, validated_at, created_at, updated_at`

	var record coreotp.Record
	var validatedAt sql.NullTime
	err := r.pool.QueryRow(
		ctx,
		query,
		params.UserID,
		params.Code,
		params.ExpiresAt,
		params.CreatedAt,
		params.UpdatedAt,
	).Scan(
		&record.ID,
		&record.UserID,
		&record.Code,
		&record.Status,
		&record.FailedAttempts,
		&record.ExpiresAt,
		&validatedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, coreotp.ErrAlreadyActive
		}
		return nil, err
	}
	if validatedAt.Valid {
		t := validatedAt.Time
		record.ValidatedAt = &t
	}

	return &record, nil
}

func (r *OTPRepository) MarkExpired(ctx context.Context, id int64, updatedAt time.Time) error {
	const query = `
		UPDATE otps
		SET status = 'expired', updated_at = $2
		WHERE id = $1 AND status = 'created'`

	_, err := r.pool.Exec(ctx, query, id, updatedAt)
	return err
}

func (r *OTPRepository) IncrementFailedAttempts(ctx context.Context, id int64, updatedAt time.Time) (int, error) {
	const query = `
		UPDATE otps
		SET failed_attempts = failed_attempts + 1, updated_at = $2
		WHERE id = $1 AND status = 'created'
		RETURNING failed_attempts`

	var attempts int
	err := r.pool.QueryRow(ctx, query, id, updatedAt).Scan(&attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, coreotp.ErrNotFound
	}
	if err != nil {
		return 0, err
	}

	return attempts, nil
}

func (r *OTPRepository) MarkValidated(ctx context.Context, id int64, validatedAt time.Time) error {
	const query = `
		UPDATE otps
		SET status = 'validated', validated_at = $2, updated_at = $2
		WHERE id = $1 AND status = 'created'`

	result, err := r.pool.Exec(ctx, query, id, validatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return coreotp.ErrNotFound
	}
	return err
}
