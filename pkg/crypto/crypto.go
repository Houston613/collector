package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// LoadPublicKey reads a PEM-encoded RSA public key from the given file path.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("key type is not RSA public key")
		}
		return rsaPub, nil
	}

	rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err == nil {
		return rsaPub, nil
	}

	return nil, fmt.Errorf("parse public key: %w", err)
}

// LoadPrivateKey reads a PEM-encoded RSA private key from the given file path.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing private key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return priv, nil
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaPriv, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("key type is not RSA private key")
		}
		return rsaPriv, nil
	}

	return nil, fmt.Errorf("parse private key: %w", err)
}

// Encrypt encrypts data using PKCS#1 v1.5 RSA encryption in chunks.
func Encrypt(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	if pub == nil || len(data) == 0 {
		return data, nil
	}

	keySize := pub.Size()
	maxChunk := keySize - 11
	if maxChunk <= 0 {
		return nil, errors.New("invalid RSA key size")
	}

	var encrypted []byte
	for i := 0; i < len(data); i += maxChunk {
		end := i + maxChunk
		if end > len(data) {
			end = len(data)
		}

		chunk, err := rsa.EncryptPKCS1v15(rand.Reader, pub, data[i:end])
		if err != nil {
			return nil, fmt.Errorf("encrypt chunk: %w", err)
		}
		encrypted = append(encrypted, chunk...)
	}

	return encrypted, nil
}

// Decrypt decrypts cipherText using PKCS#1 v1.5 RSA decryption in chunks.
func Decrypt(priv *rsa.PrivateKey, cipherText []byte) ([]byte, error) {
	if priv == nil || len(cipherText) == 0 {
		return cipherText, nil
	}

	keySize := priv.Size()
	if len(cipherText)%keySize != 0 {
		return nil, fmt.Errorf("invalid cipherText length: %d for key size %d", len(cipherText), keySize)
	}

	var decrypted []byte
	for i := 0; i < len(cipherText); i += keySize {
		chunk, err := rsa.DecryptPKCS1v15(rand.Reader, priv, cipherText[i:i+keySize])
		if err != nil {
			return nil, fmt.Errorf("decrypt chunk: %w", err)
		}
		decrypted = append(decrypted, chunk...)
	}

	return decrypted, nil
}
