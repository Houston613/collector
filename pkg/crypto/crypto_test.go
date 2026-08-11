package crypto_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"collector/pkg/crypto"
)

func TestEncryptDecrypt(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	publicKey := &privateKey.PublicKey

	// Large payload exceeding single RSA chunk size
	payload := bytes.Repeat([]byte("Hello RSA Encryption World! "), 50)

	encrypted, err := crypto.Encrypt(publicKey, payload)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	if bytes.Equal(encrypted, payload) {
		t.Fatalf("encrypted payload should not match original payload")
	}

	decrypted, err := crypto.Decrypt(privateKey, encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, payload) {
		t.Fatalf("decrypted payload does not match original. Got %d bytes, expected %d", len(decrypted), len(payload))
	}
}

func TestLoadKeys(t *testing.T) {
	tempDir := t.TempDir()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	privPath := filepath.Join(tempDir, "private.pem")
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	pubPath := filepath.Join(tempDir, "public.pem")
	if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	loadedPriv, err := crypto.LoadPrivateKey(privPath)
	if err != nil {
		t.Fatalf("failed to load private key: %v", err)
	}

	loadedPub, err := crypto.LoadPublicKey(pubPath)
	if err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}

	msg := []byte("Test PEM loading")
	enc, err := crypto.Encrypt(loadedPub, msg)
	if err != nil {
		t.Fatalf("failed to encrypt with loaded pub: %v", err)
	}

	dec, err := crypto.Decrypt(loadedPriv, enc)
	if err != nil {
		t.Fatalf("failed to decrypt with loaded priv: %v", err)
	}

	if !bytes.Equal(dec, msg) {
		t.Fatalf("decrypted message mismatch")
	}
}
