package otp

import (
	"testing"
	"time"
)

func TestRecordIsExpiredUsesExpiryBoundary(t *testing.T) {
	expiresAt := time.Date(2026, 4, 8, 10, 2, 0, 0, time.UTC)
	record := Record{ExpiresAt: expiresAt}

	if record.IsExpired(expiresAt.Add(-time.Nanosecond)) {
		t.Fatal("expected record to still be active before expiry")
	}
	if !record.IsExpired(expiresAt) {
		t.Fatal("expected record to be expired at exact expiry time")
	}
	if !record.IsExpired(expiresAt.Add(time.Nanosecond)) {
		t.Fatal("expected record to remain expired after expiry")
	}
}
