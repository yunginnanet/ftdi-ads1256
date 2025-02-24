package ads1256

import (
	"testing"
)

func TestConvert24To32(t *testing.T) {
	t.Run("PositiveValue", func(t *testing.T) {
		data := []byte{0x7F, 0xFF, 0xFF}
		result := Convert24To32(data)
		if result != int32(8388607) {
			t.Errorf("expected 8388607, got %d", result)
		}
	})

	t.Run("NegativeValue", func(t *testing.T) {
		data := []byte{0x80, 0x00, 0x00}
		result := Convert24To32(data)
		if result != int32(-8388608) {
			t.Errorf("expected -8388608, got %d", result)
		}
	})

	t.Run("ZeroValue", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00}
		result := Convert24To32(data)
		if result != int32(0) {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("MaxPositiveCode", func(t *testing.T) {
		adc := &ADS1256{}
		code := int32(8388607)
		vRef := 2.5
		pga := 1
		result := adc.ConvertADCtoVolts(code, vRef, pga)
		if result != 5.0 {
			t.Errorf("expected 5.0, got %f", result)
		}
	})
}

func TestConvertADCtoVolts(t *testing.T) {
	t.Run("MaxNegativeCode", func(t *testing.T) {
		adc := &ADS1256{}
		code := int32(-8388608)
		vRef := 2.5
		pga := 1
		expected := -5.0
		result := adc.ConvertADCtoVolts(code, vRef, pga)
		// tolerance allowed to account for lack of floating point precision.
		if result < (expected-0.000001) || result > (expected+0.000001) {
			t.Errorf("expected -5.0, got %f", result)
		}
	})

	t.Run("ZeroCode", func(t *testing.T) {
		adc := &ADS1256{}
		code := int32(0)
		vRef := 2.5
		pga := 1
		result := adc.ConvertADCtoVolts(code, vRef, pga)
		if result != 0.0 {
			t.Errorf("expected 0.0, got %f", result)
		}
	})

	t.Run("NonZeroCode", func(t *testing.T) {
		adc := &ADS1256{}
		code := int32(4194304)
		vRef := 2.5
		pga := 1
		expected := 2.5
		result := adc.ConvertADCtoVolts(code, vRef, pga)
		// tolerance allowed to account for lack of floating point precision.
		if result > (expected+0.000001) || result < (expected-0.000001) {
			t.Errorf("expected 2.5, got %f", result)
		}
	})
}
