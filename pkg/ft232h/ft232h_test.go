package ft232h

import (
	"fmt"
	"github.com/yunginnanet/ft232h"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestFT232HDescriptor(t *testing.T) {
	t.Run("ByIndex", func(t *testing.T) {
		desc := ByIndex(0)
		if err := desc.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		t.Run("Invalid", func(t *testing.T) {
			desc = ByIndex(-1)
			if err := desc.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	})
	t.Run("BySerial", func(t *testing.T) {
		desc := BySerial("123456")
		if err := desc.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		t.Run("Invalid", func(t *testing.T) {
			desc = BySerial("")
			if err := desc.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	})
	t.Run("ByMask", func(t *testing.T) {
		mask := new(ft232h.Mask)
		mask.Index = "0"
		desc := ByMask(mask)
		if err := desc.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		t.Run("Invalid", func(t *testing.T) {
			desc = ByMask(nil)
			if err := desc.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	})
	t.Run("Mask", func(t *testing.T) {
		if ByIndex(5).Mask().Index != "5" {
			t.Error("unexpected mask index")
		}
		if BySerial("5").Mask().Serial != "5" {
			t.Error("unexpected mask serial")
		}
	})
}

func testConnect(t *testing.T, desc *Descriptor, validMask bool) DeviceInfo {
	t.Helper()

	var (
		ftdi *FT232H
		err  error
	)

	if validMask {
		if desc == nil {
			t.Fatalf("descriptor is nil")
		}
		if err = desc.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if desc == nil {
		ftdi, err = ConnectFT232h()
	} else {
		ftdi, err = ConnectFT232h(*desc)
	}

	if err != nil {
		t.Fatalf("failed to connect to FT232H: %v", err)
	}

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("Connected to FT232H: %s\n", ftdi.Info().String()))

	if err = ftdi.Close(); err != nil {
		t.Errorf("failed to close FT232H: %v", err)
	}

	return ftdi.Info()
}

func TestConnectFT232h(t *testing.T) {
	if os.Getenv("TEST_FT232H") == "" {
		t.Skip("set 'TEST_FT232H' in environment to run this test")
	}

	testInfo := testConnect(t, nil, false)

	t.Run("ByIndex", func(t *testing.T) {
		desc := ByIndex(0)
		if os.Getenv("TEST_FT232H_INDEX") != "" {
			idx, err := strconv.Atoi(strings.TrimSpace(os.Getenv("TEST_FT232H_INDEX")))
			if err != nil {
				t.Fatalf(
					"bad 'TEST_FT232H_INDEX' environment variable: %v\nvalue: %s",
					err, os.Getenv("TEST_FT232H_INDEX"),
				)
			}
			desc = ByIndex(idx)
		}

		_ = testConnect(t, &desc, true)
	})

	t.Run("BySerial", func(t *testing.T) {
		serial := ""
		if os.Getenv("TEST_FT232H_SERIAL") != "" {
			serial = strings.TrimSpace(os.Getenv("TEST_FT232H_SERIAL"))
		}

		if serial == "" {
			serial = testInfo.Serial
		}

		if serial == "" {
			t.Skip("no serial number provided, try setting 'TEST_FT232H_SERIAL' in environment")
		}

		desc := BySerial(serial)

		_ = testConnect(t, &desc, true)
	})

}
