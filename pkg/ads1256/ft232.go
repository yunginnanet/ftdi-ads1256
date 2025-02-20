package ads1256

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/yunginnanet/ft232h"
	"strconv"
)

type DeviceInfo struct {
	Index       int
	Serial      string
	Description string
	ProductID   string
	VendorID    string
	IsOpen      bool
	IsHighSpeed bool
}

type FT232H struct {
	*ft232h.FT232H
	info DeviceInfo
}

func (ft *FT232H) vidPid() (vid string, pid string) {
	vid = strconv.Itoa(int(ft.VID()))
	pid = strconv.Itoa(int(ft.PID()))

	b := bytes.NewBuffer(nil)
	h := hex.NewEncoder(b)

	if err := binary.Write(h, binary.BigEndian, ft.VID()); err == nil && len(b.String()) > 5 {
		vid = b.String()[4:]
	}

	b.Reset()

	if err := binary.Write(h, binary.BigEndian, ft.PID()); err == nil && len(b.String()) > 5 {
		pid = b.String()[4:]
	}

	return vid, pid
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

func (ft *FT232H) String() string {
	s := fmt.Sprintf("FT232H[%s:%s]: %s", ft.Info().VendorID, ft.Info().ProductID, ft.Desc())
	return s
}

type FT232HDescriptor struct {
	Index  int
	Serial string
	mask   *ft232h.Mask
}

var ErrBadDescriptor = fmt.Errorf("invalid FT232H descriptor provided")

func emptyMask(mask *ft232h.Mask) bool {
	return mask == nil || (mask.Serial == "" && mask.PID == "" && mask.VID == "" && mask.Desc == "" && mask.Index == "")
}

func (ftd FT232HDescriptor) Validate() error {
	if ftd.Index < 0 && ftd.Serial == "" && emptyMask(ftd.mask) {
		return ErrBadDescriptor
	}
	return nil
}

func (ftd FT232HDescriptor) Mask() *ft232h.Mask {
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

func (ftd FT232HDescriptor) String() string {
	return fmt.Sprintf("FT232HDescriptor{Index:%d, Serial:%s, mask:%v}", ftd.Index, ftd.Serial, ftd.mask)
}

func ByIndex(index int) FT232HDescriptor {
	return FT232HDescriptor{Index: index}
}

func BySerial(serial string) FT232HDescriptor {
	return FT232HDescriptor{Serial: serial, Index: -1}
}

func ByMask(mask *ft232h.Mask) FT232HDescriptor {
	return FT232HDescriptor{mask: mask, Index: -1}
}

func ConnectFT232h(choice ...FT232HDescriptor) (ft *FT232H, err error) {
	ft = &FT232H{}

	switch len(choice) {
	case 0:
		ft.FT232H, err = ft232h.New()
		return ft, err
	case 1:
		desc := choice[0]
		if err = choice[0].Validate(); err != nil {
			return nil, ErrBadDescriptor
		}
		ft.FT232H, err = ft232h.OpenMask(desc.Mask())
		ft.info = ft.Info()
	default:
		return nil, fmt.Errorf("invalid number of arguments")
	}

	return ft, err
}
