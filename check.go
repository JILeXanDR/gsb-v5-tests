package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"slices"
)

func checkURLIsSafe(rawURL string, lists []localList) (bool, error) {
	expressions, err := generateExpressions(rawURL)
	if err != nil {
		return false, err
	}

	for _, list := range lists {
		log.Printf("check url %s in list %s", rawURL, list.name)
		found, err := findURLByPrefix(list, expressions)
		if err != nil {
			return false, err
		}
		if found {
			return false, nil
		}
	}

	return true, nil
}

func findURLByPrefix(list localList, expressions []string) (bool, error) {
	if !slices.IsSorted(list.decodedHashes) {
		return false, fmt.Errorf("slice is not sorted")
	}

	for _, expression := range expressions {
		hash := hashPrefix(expression)

		index, found := slices.BinarySearch(list.decodedHashes, hash)
		log.Printf("check hash %d in %s list: prefix=%s, found=%v", hash, list.name, expression, found)
		if found {
			log.Printf("prefix found: index=%d", index)
			return true, nil
		}
	}

	log.Printf("prefix not found in all lists")

	return false, nil
}

func hashPrefix(input string) uint32 {
	hash := sha256.Sum256([]byte(input))
	return binary.BigEndian.Uint32(hash[:4])
}
