package ads1256

// Convert24To32 interprets a 3-byte, 24-bit signed value
// in two's complement form, MSB first, as a 32-bit int.
func Convert24To32(data []byte) int32 {
	// data[0] is MSB. If top bit set => negative
	var u32 uint32
	u32 |= uint32(data[0]) << 16
	u32 |= uint32(data[1]) << 8
	u32 |= uint32(data[2])

	// sign extension
	if (u32 & 0x800000) != 0 {
		u32 |= 0xFF000000
	}
	return int32(u32)
}
