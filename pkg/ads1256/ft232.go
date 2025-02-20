package ads1256

import (
	"fmt"
	"github.com/yunginnanet/ft232h"
	"strconv"
)

type FT232H struct {
	*ft232h.FT232H
	mask *ft232h.Mask
}

func (ft *FT232H) Mask() *ft232h.Mask {
	return ft.mask
}

func (ft *FT232H) String() string {
	return fmt.Sprintf("FT232H{FT232H:%v}", ft.mask)
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
		ft.mask = desc.Mask()
	default:
		return nil, fmt.Errorf("invalid number of arguments")
	}

	return ft, err
}
