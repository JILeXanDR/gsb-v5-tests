package main

import "errors"

func DecodeUint32HashPrefixes(encodedData []byte, entriesCount uint32, firstValue uint32, riceParameter uint32) ([]uint32, error) {
	if riceParameter > 31 {
		return nil, errors.New("invalid rice parameter: must be <= 31")
	}

	decodedValues := make([]uint32, entriesCount+1)
	decodedValues[0] = firstValue

	bitStream := NewBitStream32(encodedData)
	currentValue := firstValue

	for i := uint32(0); i < entriesCount; i++ {
		quotient, err := bitStream.ReadUnary()
		if err != nil {
			return nil, err
		}

		remainder, err := bitStream.ReadBits(riceParameter)
		if err != nil {
			return nil, err
		}

		adjacentDifference := (quotient << riceParameter) | remainder
		currentValue += adjacentDifference
		decodedValues[i+1] = currentValue
	}

	return decodedValues, nil
}

type BitStream32 struct {
	data   []byte
	bitPos int
}

func NewBitStream32(data []byte) *BitStream32 {
	return &BitStream32{data: data, bitPos: 0}
}

func (b *BitStream32) ReadBits(n uint32) (uint32, error) {
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

func (b *BitStream32) ReadUnary() (uint32, error) {
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
