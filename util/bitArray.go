package util

type BitArray []uint8

func NewBitArray(n int) BitArray {
	return make(BitArray, n/8)
}

func (b BitArray) SetBit(index int, value bool) {
	position := index / 8
	i := uint(index % 8)
	if value {
		b[position] |= uint8(1) << i
	} else {
		b[position] &= ^(uint8(1) << i)
	}
}

func (b BitArray) SetBitFromUint8(index int, value uint8) {
	position := index / 8
	i := uint(index % 8)
	if value == 0 {
		b[position] &= ^(uint8(1) << i)
	} else {
		b[position] |= uint8(1) << i
	}
}

func (b BitArray) GetBit(index int) bool {
	// Calculate the position in the array where the bit is stored.
	position := index / 8

	// Calculate the position of the bit within the uint8 at the calculated position.
	bitPosition := uint(index % 8)

	// Create a mask by shifting 1 to the left by the bit position.
	mask := uint8(1) << bitPosition

	// Use bitwise AND to get the value of the bit at the bit position in the uint8.
	// If the bit is set, the result will be non-zero.
	bitIsSet := b[position] & mask

	// Return true if the bit is set, false otherwise.
	return bitIsSet != 0
}

func (b BitArray) GetBitToUint8(index int) uint8 {
	// Calculate the position in the array where the bit is stored.
	position := index / 8

	// Calculate the position of the bit within the uint8 at the calculated position.
	bitPosition := uint(index % 8)

	// Create a mask by shifting 1 to the left by the bit position.
	mask := uint8(1) << bitPosition

	// Use bitwise AND to get the value of the bit at the bit position in the uint8.
	// If the bit is set, the result will be non-zero.
	bitIsSet := b[position] & mask

	// Return 1 if the bit is set, 0 otherwise.
	if bitIsSet != 0 {
		return 1
	}
	return 0
}

func (b BitArray) Len() int {
	return 8 * len(b)
}
