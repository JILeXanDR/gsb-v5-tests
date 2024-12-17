package main

import "errors"

type golomb32BitEncoding struct {
	FirstValue    uint32
	RiceParameter uint32
	EncodedData   []byte
	EntryCount    uint32
}

// func (g *golomb32BitEncoding) Encode(hashes []uint32) error {
// 	panic("not implemented")
// 	return nil
// }

func (g *golomb32BitEncoding) Decode() ([]uint32, error) {
	if g.RiceParameter > 31 {
		return nil, errors.New("invalid rice parameter: must be <= 31")
	}

	decodedValues := make([]uint32, g.EntryCount+1)
	decodedValues[0] = g.FirstValue

	bitStream := newBitStream32(g.EncodedData)
	currentValue := g.FirstValue

	for i := uint32(0); i < g.EntryCount; i++ {
		quotient, err := bitStream.readUnary()
		if err != nil {
			return nil, err
		}

		remainder, err := bitStream.readBits(g.RiceParameter)
		if err != nil {
			return nil, err
		}

		adjacentDifference := (quotient << g.RiceParameter) | remainder
		currentValue += adjacentDifference
		decodedValues[i+1] = currentValue
	}

	return decodedValues, nil
}

type bitStream32 struct {
	data   []byte
	bitPos int
}

func newBitStream32(data []byte) *bitStream32 {
	return &bitStream32{data: data, bitPos: 0}
}

func (b *bitStream32) readBits(n uint32) (uint32, error) {
	if n > 32 {
		return 0, errors.New("cannot read more than 32 bits at a time")
	}

	value := uint32(0)
	for i := uint32(0); i < n; i++ {
		if b.bitPos/8 >= len(b.data) {
			return 0, errors.New("not enough data in bitstream")
		}

		byteIndex := b.bitPos / 8
		bitIndex := b.bitPos % 8
		bit := (b.data[byteIndex] >> bitIndex) & 1
		value |= uint32(bit) << i
		b.bitPos++
	}

	return value, nil
}

func (b *bitStream32) readUnary() (uint32, error) {
	count := uint32(0)
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
