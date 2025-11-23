package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken creates a SHA256 hash of the token string
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
