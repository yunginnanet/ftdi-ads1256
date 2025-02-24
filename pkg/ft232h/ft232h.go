package ft232h

import (
	"fmt"
	"github.com/ardnew/ft232h"
)

func ConnectFT232h(choice ...Descriptor) (ft *FT232H, err error) {
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
		if err == nil && ft.FT232H != nil {
			ft.info = ft.Info()
		}
	default:
		return nil, fmt.Errorf("invalid number of arguments")
	}

	return ft, err
}
