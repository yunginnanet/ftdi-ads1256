package ft232h

import (
	"fmt"
	"github.com/yunginnanet/ft232h"
	"time"
)

func (ft *FT232H) SetDRDY(pin uint) {
	ft.drdyPin = ft232h.CPin(pin)
	if err := ft.GPIO.ConfigPin(ft.drdyPin, ft232h.Output, true); err != nil {
		fmt.Printf("failed to configure DRDY pin: %v\n", err)
	}
	fmt.Printf("drdy set: %s, pos: %d\n", ft.drdyPin.String(), ft.drdyPin.Pos())
}

func (ft *FT232H) WaitDRDY() error {
	for {
		hl, err := ft.FT232H.GPIO.Get(ft.drdyPin)
		if err != nil {
			return fmt.Errorf("failed to read DRDY pin: %w", err)
		}
		if !hl {
			break
		}
		time.Sleep(100 * time.Microsecond)
	}
	return nil
}

func (ft *FT232H) SetPWDN(pin uint) {
	ft.pwdnPin = ft232h.CPin(pin)
	fmt.Printf("pwdn set: %s, pos: %d\n", ft.pwdnPin.String(), ft.pwdnPin.Pos())
}

func (ft *FT232H) PowerDown() error {
	if ft.pwdnPin == 0 {
		return fmt.Errorf("PWDN pin not set")
	}
	if err := ft.FT232H.GPIO.Set(ft.pwdnPin, false); err != nil {
		return fmt.Errorf("failed to set PWDN pin: %w", err)
	}
	return nil
}

func (ft *FT232H) PowerUp() error {
	if ft.pwdnPin == 0 {
		return fmt.Errorf("PWDN pin not set")
	}
	if err := ft.FT232H.GPIO.Set(ft.pwdnPin, true); err != nil {
		return fmt.Errorf("failed to set PWDN pin: %w", err)
	}
	return nil
}

func (ft *FT232H) SetCSPin(pin uint) {
	ft.csPin = ft232h.CPin(pin)
	fmt.Printf("cs set: %s, pos: %d\n", ft.csPin.String(), ft.csPin.Pos())
}

func (ft *FT232H) SetCS(high bool) error {
	return ft.FT232H.GPIO.Set(ft.csPin, high)
}

func (ft *FT232H) Read(count uint, start bool, stop bool) ([]byte, error) {
	return ft.SPI.Read(count, start, stop)
}

func (ft *FT232H) Write(data []byte, start bool, stop bool) (uint, error) {
	return ft.SPI.Write(data, start, stop)
}

func (ft *FT232H) Init() error {
	return ft.SPI.Init()
}

func (ft *FT232H) Close() error {
	return ft.SPI.Close()
}
