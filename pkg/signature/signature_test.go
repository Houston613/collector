package signature

import (
	"testing"
)

func TestSignAndVerify(t *testing.T) {
	key := "secret-key"
	data := []byte("hello world")

	// Test Sign
	sig := Sign(data, key)
	if sig == "" {
		t.Error("expected signature to be non-empty")
	}

	// Test Verify success
	if !Verify(data, key, sig) {
		t.Error("expected signature to be verified successfully")
	}

	// Test Verify failure with incorrect key
	if Verify(data, "wrong-key", sig) {
		t.Error("expected verification to fail with wrong key")
	}

	// Test Verify failure with incorrect data
	if Verify([]byte("different data"), key, sig) {
		t.Error("expected verification to fail with different data")
	}

	// Test Verify failure with invalid hex signature
	if Verify(data, key, "invalid-hex-signature") {
		t.Error("expected verification to fail with invalid hex signature")
	}

	// Test Verify failure with valid hex but incorrect signature value
	if Verify(data, key, "0000000000000000000000000000000000000000000000000000000000000000") {
		t.Error("expected verification to fail with wrong signature")
	}
}
