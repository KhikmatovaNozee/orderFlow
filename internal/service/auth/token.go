package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func generateRefreshToken() (string, string, error) {
	randomBytes := make([]byte, 32)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("generate random token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token))

	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	return token, tokenHash, nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))

	return base64.RawURLEncoding.EncodeToString(hash[:])
}
