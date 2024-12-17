package main

import (
	"errors"
)

type golomb256BitEncoding struct {
	FirstValuePart1 uint64
	FirstValuePart2 uint64
	FirstValuePart3 uint64
	FirstValuePart4 uint64
	RiceParameter   uint32
	EncodedData     []byte
	EntryCount      uint32
}

// Decode decodes Rice-Golomb encoded 256-bit delta-encoded numbers.
func (g *golomb256BitEncoding) Decode() ([]Uint256, error) {
	if g.RiceParameter < 227 || g.RiceParameter > 254 {
		return nil, errors.New("invalid rice parameter: must be between 227 and 254")
	}

	firstValue := Uint256{
		Part1: g.FirstValuePart1,
		Part2: g.FirstValuePart2,
		Part3: g.FirstValuePart3,
		Part4: g.FirstValuePart4,
	}

	decodedValues := make([]Uint256, g.EntryCount+1)
	decodedValues[0] = firstValue

	bitStream := NewBitStream256(g.EncodedData)
	currentValue := firstValue

	// Determine remainder size for each part
	remainderBits := g.RiceParameter / 4

	for i := uint32(0); i < g.EntryCount; i++ {
		// Read the unary-encoded quotient
		quotient, err := bitStream.ReadUnary()
		if err != nil {
			return nil, err
		}

		// Read the remainder parts
		r1, err := bitStream.ReadBits(remainderBits)
		if err != nil {
			return nil, err
		}
		r2, err := bitStream.ReadBits(remainderBits)
		if err != nil {
			return nil, err
		}
		r3, err := bitStream.ReadBits(remainderBits)
		if err != nil {
			return nil, err
		}
		r4, err := bitStream.ReadBits(remainderBits)
		if err != nil {
			return nil, err
		}

		// Combine quotient and remainders into a delta
		delta := Uint256{
			Part1: (quotient << remainderBits) | r1,
			Part2: r2,
			Part3: r3,
			Part4: r4,
		}

		// Add delta to the current value
		currentValue = currentValue.Add(delta)
		decodedValues[i+1] = currentValue
	}

	return decodedValues, nil
}

// BitStream256 reads bits and unary-encoded data from a byte slice.
type BitStream256 struct {
	data   []byte
	bitPos int
}

func NewBitStream256(data []byte) *BitStream256 {
	return &BitStream256{data: data, bitPos: 0}
}

func (b *BitStream256) ReadBits(n uint32) (uint64, error) {
	if n > 64 {
		return 0, errors.New("cannot read more than 64 bits at a time")
	}

	value := uint64(0)
	for i := uint32(0); i < n; i++ {
		if b.bitPos/8 >= len(b.data) {
			return 0, errors.New("not enough data in bitstream")
		}

		byteIndex := b.bitPos / 8
		bitIndex := b.bitPos % 8
		bit := (b.data[byteIndex] >> bitIndex) & 1
		value |= uint64(bit) << i
		b.bitPos++
	}

	return value, nil
}

func (b *BitStream256) ReadUnary() (uint64, error) {
	count := uint64(0)
	for {
		if b.bitPos/8 >= len(b.data) {
			return 0, errors.New("not enough data in bitstream")
		}

		byteIndex := b.bitPos / 8
		bitIndex := b.bitPos % 8
		bit := (b.data[byteIndex] >> bitIndex) & 1
		b.bitPos++

		if bit == 0 {
			break
		}
		count++
	}
	return count, nil
}

// Uint256 represents a 256-bit unsigned integer using four 64-bit parts.
type Uint256 struct {
	Part1 uint64 // First 64 bits
	Part2 uint64 // Second 64 bits
	Part3 uint64 // Third 64 bits
	Part4 uint64 // Last 64 bits
}

// Add adds a 256-bit delta to the current Uint256 value.
func (u Uint256) Add(delta Uint256) Uint256 {
	p4 := u.Part4 + delta.Part4
	p3 := u.Part3 + delta.Part3
	p2 := u.Part2 + delta.Part2
	p1 := u.Part1 + delta.Part1

	// Handle carry propagation
	if p4 < u.Part4 {
		p3++
	}
	if p3 < u.Part3 {
		p2++
	}
	if p2 < u.Part2 {
		p1++
	}

	return Uint256{Part1: p1, Part2: p2, Part3: p3, Part4: p4}
}

// Compare compares two Uint256 values. Returns:
//   - -1 if u < other
//   - 0 if u == other
//   - 1 if u > other
func (u Uint256) Compare(other Uint256) int {
	if u.Part1 < other.Part1 {
		return -1
	} else if u.Part1 > other.Part1 {
		return 1
	}

	if u.Part2 < other.Part2 {
		return -1
	} else if u.Part2 > other.Part2 {
		return 1
	}

	if u.Part3 < other.Part3 {
		return -1
	} else if u.Part3 > other.Part3 {
		return 1
	}

	if u.Part4 < other.Part4 {
		return -1
	} else if u.Part4 > other.Part4 {
		return 1
	}

	return 0
}
