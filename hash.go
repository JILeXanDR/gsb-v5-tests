package main

import (
	"crypto/sha256"
	"encoding/binary"
)

func hashUint32FourBytes(input string) uint32 {
	hash := sha256.Sum256([]byte(input))
	return binary.BigEndian.Uint32(hash[:4])
}

func hashUint256(input string) Uint256 {
	hash := sha256.Sum256([]byte(input))
	return Uint256{
		Part1: binary.BigEndian.Uint64(hash[:8]),
		Part2: binary.BigEndian.Uint64(hash[8:16]),
		Part3: binary.BigEndian.Uint64(hash[16:24]),
		Part4: binary.BigEndian.Uint64(hash[24:32]),
	}
}

func hashUint32FourBytesStrings(strings []string) []uint32 {
	hashes := make([]uint32, len(strings))

	for i, str := range strings {
		hashes[i] = hashUint32FourBytes(str)
	}

	return hashes
}
