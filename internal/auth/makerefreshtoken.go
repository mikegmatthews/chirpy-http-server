package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() string {
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	return hex.EncodeToString(tokenBytes)
}
