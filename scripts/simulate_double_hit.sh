#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
SCENARIO="${1:-all}"
USER_PREFIX="${USER_PREFIX:-double-hit-$(date +%s)}"
ITERATIONS="${ITERATIONS:-1}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

OVERALL_EXIT_CODE=0

usage() {
  cat <<'EOF'
Usage: scripts/simulate_double_hit.sh [request|validate|all|help]

Environment variables:
  BASE_URL      Target API base URL (default: http://localhost:8080)
  USER_PREFIX   Prefix for generated test users
  ITERATIONS    Number of times to repeat the selected simulation (default: 1)

Examples:
  bash scripts/simulate_double_hit.sh request
  ITERATIONS=10 bash scripts/simulate_double_hit.sh all
  BASE_URL=http://localhost:9000 bash scripts/simulate_double_hit.sh validate
EOF
}

validate_iterations() {
  if ! [[ "$ITERATIONS" =~ ^[1-9][0-9]*$ ]]; then
    printf 'ITERATIONS must be a positive integer, got: %s\n' "$ITERATIONS" >&2
    exit 1
  fi
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$1" >&2
    exit 1
  fi
}

post_json() {
  local path="$1"
  local payload="$2"
  local output_file="$3"

  curl -sS -X POST "${BASE_URL}${path}" \
    -H 'Content-Type: application/json' \
    -d "$payload" \
    -w '\nHTTP_STATUS:%{http_code}\n' \
    > "$output_file"
}

response_status() {
  sed -n 's/^HTTP_STATUS://p' "$1"
}

response_body() {
  sed '/^HTTP_STATUS:/d' "$1"
}

extract_otp() {
  response_body "$1" | tr -d '\n' | sed -n 's/.*"otp"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
}

extract_error_code() {
  response_body "$1" | tr -d '\n' | sed -n 's/.*"code"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
}

extract_message() {
  response_body "$1" | tr -d '\n' | sed -n 's/.*"message"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
}

print_response() {
  local label="$1"
  local output_file="$2"

  printf '%s\n' "${label}"
  printf 'HTTP %s\n' "$(response_status "$output_file")"
  response_body "$output_file"
  printf '\n'
}

count_status() {
  local expected="$1"
  shift

  local count=0
  local file
  for file in "$@"; do
    if [ "$(response_status "$file")" = "$expected" ]; then
      count=$((count + 1))
    fi
  done
  printf '%s' "$count"
}

print_verdict() {
  local label="$1"
  local passed="$2"
  local details="$3"

  if [ "$passed" = 'true' ]; then
    printf 'VERDICT: PASS - %s\n' "$label"
  else
    printf 'VERDICT: FAIL - %s\n' "$label"
    OVERALL_EXIT_CODE=1
  fi
  printf '%s\n\n' "$details"
}

first_file_with_status() {
  local expected="$1"
  shift

  local file
  for file in "$@"; do
    if [ "$(response_status "$file")" = "$expected" ]; then
      printf '%s' "$file"
      return 0
    fi
  done
  return 1
}

run_parallel_request() {
  local path="$1"
  local payload="$2"
  local left_file="$3"
  local right_file="$4"

  post_json "$path" "$payload" "$left_file" &
  local left_pid=$!
  post_json "$path" "$payload" "$right_file" &
  local right_pid=$!

  wait "$left_pid"
  wait "$right_pid"
}

simulate_request_double_hit() {
  local user_id="$1"
  local payload
  payload=$(printf '{"user_id":"%s"}' "$user_id")

  local first_file="${TMP_DIR}/request-first.out"
  local second_file="${TMP_DIR}/request-second.out"

  printf '== Request OTP double hit ==\n'
  printf 'Base URL: %s\n' "$BASE_URL"
  printf 'User ID: %s\n\n' "$user_id"

  run_parallel_request "/otp/request" "$payload" "$first_file" "$second_file"

  print_response 'Response #1' "$first_file"
  print_response 'Response #2' "$second_file"

  local ok_count conflict_count
  ok_count="$(count_status 200 "$first_file" "$second_file")"
  conflict_count="$(count_status 409 "$first_file" "$second_file")"
  local ok_file conflict_file conflict_code generated_otp
  ok_file="$(first_file_with_status 200 "$first_file" "$second_file" || true)"
  conflict_file="$(first_file_with_status 409 "$first_file" "$second_file" || true)"
  conflict_code=''
  generated_otp=''
  if [ -n "$conflict_file" ]; then
    conflict_code="$(extract_error_code "$conflict_file")"
  fi
  if [ -n "$ok_file" ]; then
    generated_otp="$(extract_otp "$ok_file")"
  fi

  if [ "$ok_count" = '1' ] && [ "$conflict_count" = '1' ] && [ "$conflict_code" = 'otp_already_active' ] && [ -n "$generated_otp" ]; then
    print_verdict 'request double hit keeps a single winner' 'true' 'Expected current design: exactly one HTTP 200 with an OTP payload and one HTTP 409 with error code otp_already_active.'
  else
    print_verdict 'request double hit keeps a single winner' 'false' "Expected one HTTP 200 with OTP payload and one HTTP 409 with error code otp_already_active, got statuses: $(response_status "$first_file"), $(response_status "$second_file"); conflict code: ${conflict_code:-<none>}; otp present: ${generated_otp:+yes}${generated_otp:-no}"
  fi

}

simulate_validate_double_hit() {
  local user_id="$1"
  local setup_file="${TMP_DIR}/validate-setup.out"
  local first_file="${TMP_DIR}/validate-first.out"
  local second_file="${TMP_DIR}/validate-second.out"
  local request_payload
  request_payload=$(printf '{"user_id":"%s"}' "$user_id")

  printf '== Validate OTP double hit ==\n'
  printf 'Base URL: %s\n' "$BASE_URL"
  printf 'User ID: %s\n\n' "$user_id"

  post_json "/otp/request" "$request_payload" "$setup_file"
  print_response 'Setup request' "$setup_file"

  if [ "$(response_status "$setup_file")" != '200' ]; then
    printf '%s\n' 'Setup request failed, validate simulation stopped.' >&2
    exit 1
  fi

  local otp
  otp="$(extract_otp "$setup_file")"
  if [ -z "$otp" ]; then
    printf '%s\n' 'Could not extract OTP from setup response.' >&2
    exit 1
  fi

  local validate_payload
  validate_payload=$(printf '{"user_id":"%s","otp":"%s"}' "$user_id" "$otp")
  run_parallel_request "/otp/validate" "$validate_payload" "$first_file" "$second_file"

  print_response 'Response #1' "$first_file"
  print_response 'Response #2' "$second_file"

  local ok_count not_found_count
  ok_count="$(count_status 200 "$first_file" "$second_file")"
  not_found_count="$(count_status 404 "$first_file" "$second_file")"
  local ok_file not_found_file not_found_code success_message
  ok_file="$(first_file_with_status 200 "$first_file" "$second_file" || true)"
  not_found_file="$(first_file_with_status 404 "$first_file" "$second_file" || true)"
  not_found_code=''
  success_message=''
  if [ -n "$not_found_file" ]; then
    not_found_code="$(extract_error_code "$not_found_file")"
  fi
  if [ -n "$ok_file" ]; then
    success_message="$(extract_message "$ok_file")"
  fi

  if [ "$ok_count" = '1' ] && [ "$not_found_count" = '1' ] && [ "$not_found_code" = 'otp_not_found' ] && [ "$success_message" = 'OTP validated successfully.' ]; then
    print_verdict 'validate double hit allows only one success' 'true' 'Expected current design: exactly one HTTP 200 with the success message and one HTTP 404 with error code otp_not_found.'
  else
    print_verdict 'validate double hit allows only one success' 'false' "Expected one HTTP 200 with success message and one HTTP 404 with error code otp_not_found, got statuses: $(response_status "$first_file"), $(response_status "$second_file"); not_found code: ${not_found_code:-<none>}; success message: ${success_message:-<none>}"
  fi
}

main() {
  require_command curl
  require_command sed
  require_command tr
  require_command mktemp
  validate_iterations

  case "$SCENARIO" in
    help|-h|--help)
      usage
      exit 0
      ;;
    request)
      ;;
    validate)
      ;;
    all)
      ;;
    *)
      usage >&2
      exit 1
      ;;
  esac

  local i run_prefix
  for ((i = 1; i <= ITERATIONS; i++)); do
    run_prefix="$USER_PREFIX"
    if [ "$ITERATIONS" -gt 1 ]; then
      run_prefix="${USER_PREFIX}-${i}"
      printf '== Iteration %s/%s ==\n\n' "$i" "$ITERATIONS"
    fi

    case "$SCENARIO" in
      request)
        simulate_request_double_hit "${run_prefix}-request"
        ;;
      validate)
        simulate_validate_double_hit "${run_prefix}-validate"
        ;;
      all)
        simulate_request_double_hit "${run_prefix}-request"
        simulate_validate_double_hit "${run_prefix}-validate"
        ;;
    esac
  done

  exit "$OVERALL_EXIT_CODE"
}

main "$@"
