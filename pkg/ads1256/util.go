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

// ConvertADCtoVolts converts the signed 24-bit code to a voltage.
// full-scale range = ±2 * Vref / PGA. For a code of 0x7FFFFF => +FS.
// Credit: https://github.com/vgasparyan/ads1256-rs/blob/eee60a5dc138dd16aa0443ea161692c1246197d4/src/lib.rs#L314-L316
func (adc *ADS1256) ConvertADCtoVolts(code int32, vRef float64, pga int) float64 {
	// The ADS1256 is a 24-bit device with a max positive code of 0x7FFFFF
	// meaning + (2 * vRef / pga), and min code 0x800000 = - (2 * vRef / pga).
	fullScale := (2.0 * vRef) / float64(pga)
	// Convert to normalized -1..+1
	// code range is [-8388608..8388607].
	// But typically we treat 0x7FFFFF as +FS, 0x800000 as -FS
	// so scale by 2^23
	return (float64(code) / 8388607.0) * fullScale
}
