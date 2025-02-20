package ft232h

import (
	"fmt"
	"github.com/yunginnanet/ft232h"
	"strconv"
)

// DeviceInfo represents a snapshot of the device information for the [FT232H] device.
type DeviceInfo struct {
	Index       int
	Serial      string
	Description string
	ProductID   string
	VendorID    string
	IsOpen      bool
	IsHighSpeed bool
}

// String returns a string representation of the device information.
func (ft DeviceInfo) String() string {
	return fmt.Sprintf(
		"DeviceInfo{Index:%d, Serial:%s, Description:%s, ProductID:%s, VendorID:%s, IsOpen:%t, IsHighSpeed:%t}",
		ft.Index, ft.Serial, ft.Description, ft.ProductID, ft.VendorID, ft.IsOpen, ft.IsHighSpeed,
	)
}

// FT232H represents an FT232H device.
type FT232H struct {
	*ft232h.FT232H
	info DeviceInfo
}

// Info returns a snapshot of the device information for the FT232H device. Read-only.
func (ft *FT232H) Info() DeviceInfo {
	vid, pid := ft.vidPid()
	return DeviceInfo{
		Index:       ft.Index(),
		Serial:      ft.Serial(),
		Description: ft.Desc(),
		ProductID:   pid,
		VendorID:    vid,
		IsOpen:      ft.IsOpen(),
		IsHighSpeed: ft.IsHiSpeed(),
	}
}

// String returns a string representation of the FT232H device. It includes the vendor ID, product ID, and description.
func (ft *FT232H) String() string {
	s := fmt.Sprintf("FT232H[%s:%s]: %s", ft.Info().VendorID, ft.Info().ProductID, ft.Desc())
	return s
}

// Descriptor represents a descriptor for the FT232H device. It is used to uniquely identify the device for connection.
type Descriptor struct {
	Index  int
	Serial string
	mask   *ft232h.Mask
}

// Validate checks if [Descriptor] is valid.
func (ftd Descriptor) Validate() error {
	if ftd.Index < 0 && ftd.Serial == "" && emptyMask(ftd.mask) {
		return ErrBadDescriptor
	}
	return nil
}

// Mask returns a pointer to the [ft232h.Mask] representation of the [Descriptor].
func (ftd Descriptor) Mask() *ft232h.Mask {
	if ftd.mask == nil {
		ftd.mask = new(ft232h.Mask)
	}
	if ftd.Serial != "" {
		ftd.mask.Serial = ftd.Serial
	}
	if ftd.Index >= 0 {
		ftd.mask.Index = strconv.Itoa(ftd.Index)
	}
	return ftd.mask
}

// String returns a string representation of the [Descriptor].
func (ftd Descriptor) String() string {
	return fmt.Sprintf("Descriptor{Index:%d, Serial:%s, mask:%v}", ftd.Index, ftd.Serial, ftd.mask)
}

// ByIndex returns a [Descriptor] with the specified index.
func ByIndex(index int) Descriptor {
	return Descriptor{Index: index}
}

// BySerial returns a [Descriptor] with the specified serial number.
func BySerial(serial string) Descriptor {
	return Descriptor{Serial: serial, Index: -1}
}

// ByMask returns a [Descriptor] with the specified mask.
func ByMask(mask *ft232h.Mask) Descriptor {
	return Descriptor{mask: mask, Index: -1}
}
