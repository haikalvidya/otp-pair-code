# Design Notes

## Architecture

The codebase is split into a small set of layers:

- `cmd/api`: process entrypoint.
- `internal/bootstrap`: app wiring, config, logger, provider setup.
- `internal/adapters/http`: HTTP router, handlers, DTOs, API error mapping.
- `internal/application/otp`: OTP use case orchestration.
- `internal/core/otp`: domain entity and domain errors.
- `internal/adapters/persistence/postgres`: Postgres repository implementation.
- `internal/ports`: interfaces used by the application layer.
- `migrations`: database schema managed by `goose`.

Request flow:

1. HTTP handler validates payload.
2. Application service applies OTP rules.
3. Repository reads and writes OTP state in Postgres.
4. Handler maps result into a consistent API response.

## OTP Lifecycle

Current behavior by design:

- `request OTP`
  Creates a new OTP when there is no active OTP.
- `request OTP` while an active OTP exists
  Returns `409 otp_already_active` when `OTP_ALLOW_REISSUE=false`.
- `request OTP` with `OTP_ALLOW_REISSUE=true`
  Expires the current OTP, then creates a new one.
- `validate OTP` with a correct code
  Marks the OTP as `validated` and removes it from the active set.
- `validate OTP` with a wrong code
  Increments `failed_attempts`.
- `validate OTP` after too many wrong attempts
  Expires the OTP and returns `429 otp_blocked`.

Other design constraints:

- OTP validity window is fixed at 2 minutes.
- Only one active OTP (`status = created`) is allowed per user.
- Successful validation consumes the OTP and prevents reuse.

## Persistence Rules

Double-hit protection is enforced in Postgres with a partial unique index:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_otps_one_created_per_user
    ON otps (user_id)
    WHERE status = 'created';
```

This keeps the active OTP invariant at the database layer, not only in application logic.

## Concurrency Expectations

Expected concurrent outcomes:

- Double hit `POST /otp/request`: one `200`, one `409 otp_already_active`.
- Double hit `POST /otp/validate` for the same OTP: one `200`, one `404 otp_not_found`.

This behavior is also covered by service-level concurrency tests in `internal/application/otp/usecase_test.go` and can be exercised manually with `scripts/simulate_double_hit.sh`.
