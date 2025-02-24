package ads1256

import "io"

func (adc *ADS1256) setCSLow() error {
	return adc.spi.SetCS(false)
}
func (adc *ADS1256) setCSHigh() error {
	return adc.spi.SetCS(true)
}

func (adc *ADS1256) Write(p []byte) (int, error) {
	n, err := adc.spi.Write(p, false, false)
	return int(n), err
}
func (adc *ADS1256) Read(p []byte) (int, error) {
	b, err := adc.spi.Read(uint(len(p)), false, false)
	if err != nil {
		return 0, err
	}
	if len(p) < len(b) {
		return 0, io.ErrShortBuffer
	}
	return copy(p, b), err
}
