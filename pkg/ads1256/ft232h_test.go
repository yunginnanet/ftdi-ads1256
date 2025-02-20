package ads1256

import (
	"github.com/yunginnanet/ft232h"
	"github.com/l0nax/go-spew/spew"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
)

var pprint = spew.ConfigState{
	Indent:                  "\t",
	MaxDepth:                0,
	DisableMethods:          false,
	DisablePointerMethods:   false,
	DisablePointerAddresses: false,
	DisableCapacities:       false,
	ContinueOnMethod:        true,
	SortKeys:                true,
	SpewKeys:                true,
	HighlightValues:         true,
	HighlightHex:            true,
}

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

func testConnect(t *testing.T, desc *FT232HDescriptor, validMask bool) ft232h.Mask {
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
	t.Logf("FT232H connected: %s", ftdi.String())

	pprint.Dump(ftdi.FT232H)

	msk := copyMask(ftdi.Mask())

	pprint.Dump(msk)

	if err = ftdi.Close(); err != nil {
		t.Errorf("failed to close FT232H: %v", err)
	}

	return msk
}

func copyMask(src *ft232h.Mask) ft232h.Mask {
	msk := ft232h.Mask{}
	if src != nil {
		msk.Index = src.Index
		msk.Serial = src.Serial
		msk.PID = src.PID
		msk.VID = src.VID
		msk.Desc = src.Desc
	}
	return msk
}

func TestConnectFT232h(t *testing.T) {
	if os.Getenv("TEST_FT232H") == "" {
		t.Skip("set 'TEST_FT232H' in environment to run this test")
	}

	var (
		testMaskCh = make(chan ft232h.Mask, 1)
		testMask   *ft232h.Mask
		testMaskMu sync.Mutex
	)

	getTestMask := func() ft232h.Mask {
		t.Helper()
		testMaskMu.Lock()
		if testMask == nil {
			tm := <-testMaskCh
			testMask = &tm
			close(testMaskCh)
		}
		testMaskMu.Unlock()
		return copyMask(testMask)
	}

	t.Run("NoDesc", func(t *testing.T) {
		msk := testConnect(t, nil, false)
		select {
		case testMaskCh <- msk:
		default:
		}
	})

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

		msk := testConnect(t, &desc, true)

		select {
		case testMaskCh <- msk:
		default:
		}
	})

	t.Run("BySerial", func(t *testing.T) {
		serial := ""
		if os.Getenv("TEST_FT232H_SERIAL") != "" {
			serial = strings.TrimSpace(os.Getenv("TEST_FT232H_SERIAL"))
		}

		if serial == "" {
			serial = getTestMask().Serial
		}

		desc := BySerial(serial)

		_ = testConnect(t, &desc, true)
	})

}
