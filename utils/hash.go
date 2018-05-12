package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func Hash(input string) string {
	sha_256 := sha256.New()
	sha_256.Write([]byte(input))

	return hex.EncodeToString(sha_256.Sum(nil))
}
