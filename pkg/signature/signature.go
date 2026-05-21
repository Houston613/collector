package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign вычисляет HMAC-SHA256 хеш от данных с использованием ключа.
func Sign(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// Verify проверяет соответствие HMAC-SHA256 хеша данным и ключу.
func Verify(data []byte, key string, signature string) bool {
	return Sign(data, key) == signature
}
