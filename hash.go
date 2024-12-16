package main

import (
	"crypto/sha256"
	"encoding/binary"
)

func hashPrefix(input string) uint32 {
	hash := sha256.Sum256([]byte(input))
	return binary.BigEndian.Uint32(hash[:4])
}
