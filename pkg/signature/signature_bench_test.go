package signature

import (
	"testing"
)

func BenchmarkSign(b *testing.B) {
	key := "secret-key"
	data := []byte("hello world, testing signature speed and allocations")

	for b.Loop() {
		Sign(data, key)
	}
}

func BenchmarkVerify(b *testing.B) {
	key := "secret-key"
	data := []byte("hello world, testing signature speed and allocations")
	sig := Sign(data, key)

	for b.Loop() {
		Verify(data, key, sig)
	}
}
