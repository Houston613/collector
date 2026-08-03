package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign computes the HMAC-SHA256 signature of data using the key.
func Sign(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// Verify verifies the HMAC-SHA256 signature of data using the key.
func Verify(data []byte, key string, signature string) bool {
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	expected := h.Sum(nil)

	return hmac.Equal(sigBytes, expected)
}
