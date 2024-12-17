package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gsb-v5-tests/proto"
)

// https://developers.google.com/safe-browsing/reference#decoding-hashes-and-hash-prefixes
//
// 291bc5421f1cd54d99afcc55d166e2b9fe42447025895bf09dd41b2110a687dc  a.example.com/
// 1d32c5084a360e58f1b87109637a6810acad97a861a7769e8f1841410d2a960c  b.example.com/
// f7a502e56e8b01c6dc242b35122683c9d25d07fb1f532d9853eb0ef3ff334f03  y.example.com/
func TestDecodeUint32HashPrefixes_fromExample(t *testing.T) {
	additionsFourBytes := proto.RiceDeltaEncoded32Bit{
		FirstValue:    489866504,
		RiceParameter: 30,
		EntriesCount:  2,
		EncodedData:   []byte(`t\000\322\227\033\355It\000`),
	}

	enc := &golomb32BitEncoding{
		FirstValue:    additionsFourBytes.FirstValue,
		RiceParameter: uint32(additionsFourBytes.RiceParameter),
		EncodedData:   additionsFourBytes.EncodedData,
		EntryCount:    uint32(additionsFourBytes.EntriesCount),
	}

	decodedPrefixes, err := enc.Decode()
	require.NoError(t, err)
	require.Equal(t, 3, len(decodedPrefixes))

	// hash=489866504
	// hash=894104386
	// hash=1736331122
	for _, hash := range decodedPrefixes {
		log.Printf("hash=%v", hash)
	}

	t.Run("b.example.com/", func(t *testing.T) {
		hash := sha256.Sum256([]byte("b.example.com/"))
		assert.Equal(t, "1d32c5084a360e58f1b87109637a6810acad97a861a7769e8f1841410d2a960c", fmt.Sprintf("%x", hash))
		assert.EqualValues(t, 0x1d32c508, binary.BigEndian.Uint32(hash[:4]))
		assert.EqualValues(t, 489866504, binary.BigEndian.Uint32(hash[:4]))

		index, found := slices.BinarySearch(decodedPrefixes, hashUint32FourBytes("b.example.com/"))
		require.Equal(t, true, found)
		assert.Equal(t, 0, index)
	})

	t.Run("a.example.com/", func(t *testing.T) {
		hash := sha256.Sum256([]byte("a.example.com/"))
		assert.Equal(t, "291bc5421f1cd54d99afcc55d166e2b9fe42447025895bf09dd41b2110a687dc", fmt.Sprintf("%x", hash))
		assert.EqualValues(t, 0x291bc542, binary.BigEndian.Uint32(hash[:4]))
		// assert.EqualValues(t, 689685826, binary.BigEndian.Uint32(hash[:4])) TODO: doesn't with result hash

		index, found := slices.BinarySearch(decodedPrefixes, hashUint32FourBytes("a.example.com/"))
		require.Equal(t, true, found)
		assert.Equal(t, 1, index)
	})

	t.Run("y.example.com/", func(t *testing.T) {
		hash := sha256.Sum256([]byte("y.example.com/"))
		assert.Equal(t, "f7a502e56e8b01c6dc242b35122683c9d25d07fb1f532d9853eb0ef3ff334f03", fmt.Sprintf("%x", hash))
		assert.EqualValues(t, 0xf7a502e5, binary.BigEndian.Uint32(hash[:4]))
		// assert.EqualValues(t, 4154786533, binary.BigEndian.Uint32(hash[:4])) TODO: doesn't with result hash

		index, found := slices.BinarySearch(decodedPrefixes, hashUint32FourBytes("y.example.com/"))
		require.Equal(t, true, found)
		assert.Equal(t, 2, index)
	})
}
