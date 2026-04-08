# Double-Hit Simulation

The repo includes `scripts/simulate_double_hit.sh` to simulate two concurrent hits against the same endpoint.

## Scenarios

- `request`: two concurrent `POST /otp/request` calls.
- `validate`: two concurrent `POST /otp/validate` calls using the same OTP.
- `all`: run both scenarios.

## Usage

```bash
bash scripts/simulate_double_hit.sh request
bash scripts/simulate_double_hit.sh validate
bash scripts/simulate_double_hit.sh all
```

Show help:

```bash
bash scripts/simulate_double_hit.sh help
```

## Environment Variables

- `BASE_URL`: target API base URL. Default `http://localhost:8080`.
- `USER_PREFIX`: prefix for generated test users.
- `ITERATIONS`: repeat the selected simulation multiple times. Default `1`.

Example repeated simulation:

```bash
ITERATIONS=10 bash scripts/simulate_double_hit.sh all
```

Example against another port:

```bash
BASE_URL=http://localhost:9000 bash scripts/simulate_double_hit.sh request
```

## Expected Results

For the current design:

- Double hit `request`: exactly one `200` and one `409 otp_already_active`.
- Double hit `validate`: exactly one `200` and one `404 otp_not_found`.

The script prints a `VERDICT: PASS` or `VERDICT: FAIL` summary for each scenario and exits non-zero if any simulation fails.
