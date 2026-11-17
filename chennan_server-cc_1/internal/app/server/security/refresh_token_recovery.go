package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// Recovery ciphertext is authenticated to this session. A separate derived key
// keeps its use separate from JWT signing without adding a deployment secret.
func refreshRecoveryCipher(secret string) (cipher.AEAD, error) {
	if len(secret) < 16 {
		return nil, errors.New("refresh recovery signing key is not configured")
	}
	key := sha256.Sum256([]byte("refresh-recovery-v1\x00" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func SealRefreshToken(token, secret, sessionBinding string) (string, error) {
	aead, err := refreshRecoveryCipher(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aead.Seal(nonce, nonce, []byte(token), []byte(sessionBinding))
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func OpenRefreshToken(encoded, secret, sessionBinding string) (string, error) {
	aead, err := refreshRecoveryCipher(secret)
	if err != nil {
		return "", err
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(data) < aead.NonceSize() {
		return "", errors.New("invalid refresh recovery ciphertext")
	}
	plain, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte(sessionBinding))
	if err != nil {
		return "", errors.New("refresh recovery authentication failed")
	}
	return string(plain), nil
}
