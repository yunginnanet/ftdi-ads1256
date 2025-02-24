package ads1256

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Register byte

// SerialInterface interface allows for different SerialInterface implementations.
type SerialInterface interface {
	Read(count uint, start bool, stop bool) ([]byte, error)
	Write(data []byte, start bool, stop bool) (uint, error)

	// WaitDRDY is called to wait for DRDY pin == LOW.
	WaitDRDY() error

	// PowerDown pulls the PWDN pin low.
	PowerDown() error

	// PowerUp pulls the PWDN pin high.
	PowerUp() error

	SetCS(bool) error

	Init() error

	// Close closes the interface.
	Close() error
}

// ADS1256 provides high-level control over a TI ADS1256 ADC.
//
// It uses an [io.ReadWriter] for SerialInterface communication and simple callbacks/interfaces
// for DRDY and PWDN pin handling.
type ADS1256 struct {
	mu  sync.RWMutex    // Synchronize concurrent operations
	spi SerialInterface // SerialInterface interface

	// Last read or written register states (for reference or debugging)
	regLR [NumRegisters]byte // "Last Read"  register data
	regLW [NumRegisters]byte // "Last Write" register data

	continuousMode *atomic.Bool
}

// Config represents user-level configuration parameters
type Config struct {
	DataRate byte // DR_xxx from the set of DRATE_DR_XXXX_SPS
	PGA      byte // e.g. ADCON_PGA_1, ADCON_PGA_2, ...
	BufferEn bool // Enable the ADC's internal buffer
	AutoCal  bool // If set, device auto-calibrates after certain register changes
	ClkOut   byte // 0=Off, 1=CLK/1, 2=CLK/2, 3=CLK/4
}

// DefaultConfig provides default config. You can adjust as needed
func DefaultConfig() Config {
	return Config{
		DataRate: DRATE_DR_1000_SPS, // 1k SPS
		PGA:      ADCON_PGA_1,       // gain = 1
		BufferEn: false,
		AutoCal:  false,
		ClkOut:   0, // Turn off CLKOUT
	}
}

// NewADS1256 constructs an ADS1256 object with the given SerialInterface and optional pin callbacks.
func NewADS1256(spi SerialInterface) *ADS1256 {
	return &ADS1256{
		spi:            spi,
		continuousMode: new(atomic.Bool),
	}
}

func (adc *ADS1256) WaitDRDY() error {
	return adc.spi.WaitDRDY()
}

// Initialize sets up the device with the provided config.
// Call it once at start-up. The ADS1256 automatically does a self-cal on power-up,
// but you can do an additional SELFCAL if needed.
func (adc *ADS1256) Initialize(cfg Config) error {
	adc.mu.Lock()

	// Issue hardware or software Reset if desired:
	if err := adc.Reset(); err != nil {
		adc.mu.Unlock()
		return err
	}

	// Wait a bit for device to run internal power-up routines
	time.Sleep(50 * time.Millisecond) // 30ms is typical after hardware Reset

	// Build the STATUS register byte
	var statusVal byte = 0x00
	if cfg.BufferEn {
		statusVal |= StatusBUFENbit
	}
	if cfg.AutoCal {
		statusVal |= StatusACALbit
	}

	// ORDER bit remains 0 => MSB first
	// ID bits are read-only

	// Write the STATUS register
	if err := adc.writeRegister(RegSTATUS, statusVal); err != nil {
		adc.mu.Unlock()
		return err
	}

	// Build the ADCON register
	// bit7 always 0, bits6-5 = CLK bits, bits4-3=SensorDetect, bits2-0=PGA
	var adconVal byte
	switch cfg.ClkOut {
	case 1:
		adconVal = AdconCLKDiv1
	case 2:
		adconVal = AdconCLKDiv2
	case 3:
		adconVal = AdconCLKDiv4
	default:
		adconVal = AdconCLKOff
	}

	// no sensor detect current by default
	// TODO: make this a parameter
	adconVal |= AdconSDCSOff
	// adconVal |= AdconSDCS2uA

	//goland:noinspection GoRedundantParens
	adconVal |= (cfg.PGA & 0x07) // set PGA bits
	if err := adc.writeRegister(RegADCON, adconVal); err != nil {
		adc.mu.Unlock()
		return err
	}

	if err := adc.writeRegister(RegDRATE, cfg.DataRate); err != nil {
		adc.mu.Unlock()
		return err
	}

	// Optionally set up the I/O register if you want to manipulate D0..D3
	// By default, all are inputs. Example: 0xE0 is default => (DIR3..DIR0 =1 => inputs)
	// We'll leave that as default for now.

	// Read them back to confirm (optional)
	if err := adc.readAllRegisters(); err != nil {
		adc.mu.Unlock()
		return fmt.Errorf("failed to read back registers: %v", err)
	}

	if err := adc.sendCommand(CMDSELFCAL); err != nil {
		adc.mu.Unlock()
		return err
	}

	/*	if err := adc.spi.WaitDRDY(); err != nil {
		adc.mu.Unlock()
		return err
	}*/

	adc.mu.Unlock()
	return nil
}

func (adc *ADS1256) Close() error {
	err := adc.Reset()
	err = errors.Join(err, adc.Standby())
	err = errors.Join(err, adc.PowerDown())
	err = errors.Join(err, adc.spi.Close())
	return adc.spi.PowerDown()
}

// Reset triggers a software Reset using the RESET command
func (adc *ADS1256) Reset() error {
	return adc.sendCommand(CMDRESET)
}

// Standby puts the device into standby mode, shutting down analog but leaving the oscillator running.
func (adc *ADS1256) Standby() error {
	return adc.sendCommand(CMDSTANDBY)
}

// WakeUp from SYNC or STANDBY mode.
func (adc *ADS1256) WakeUp() error {
	// ADS1256 has two forms (0x00 or 0xFF). We'll just send 0x00 for clarity.
	return adc.sendCommand(CMDWAKEUP)
}

// Sync sends a SYNC command to synchronize the ADC's data output.
func (adc *ADS1256) Sync() error {
	return adc.sendCommand(CMDSYNC)
}

// PowerDown pulls the PWDN pin low if setPWDN is provided.
// Holding SYNC/PDWN low for 20 DRDY cycles also powers down the chip.
func (adc *ADS1256) PowerDown() error {
	return adc.spi.PowerDown() // drive PWDN pin low
}

// PowerUp pulls the PWDN pin high if setPWDN is provided.
func (adc *ADS1256) PowerUp() error {
	return adc.spi.PowerUp() // drive PWDN pin high
}

// SingleConversion issues a Sync, [WakeUp], then RDATA flow to read one sample.
// Often used in "one-shot" mode. The user typically calls Standby() first,
// then SingleConversion() each time they want a measurement.
func (adc *ADS1256) SingleConversion() (int32, error) {
	adc.mu.Lock()

	// SYNC
	if err := adc.Sync(); err != nil {
		adc.mu.Unlock()
		return 0, err
	}

	// WAKEUP
	if err := adc.WakeUp(); err != nil {
		adc.mu.Unlock()
		return 0, err
	}

	// Wait for DRDY
	if err := adc.spi.WaitDRDY(); err != nil {
		adc.mu.Unlock()
		return 0, err
	}

	// Then read data with RDATA
	n, err := adc.readDataByCommand()

	adc.mu.Unlock()

	return n, err
}

// RData performs RDATA to get a single 24-bit result from the device.
func (adc *ADS1256) RData() (int32, error) {
	adc.mu.Lock()
	i3, err := adc.readDataByCommand()
	adc.mu.Unlock()
	return i3, err
}

// readDataByCommand performs the RDATA command to get a single 24-bit result from the device.
func (adc *ADS1256) readDataByCommand() (int32, error) {
	if err := adc.setCSLow(); err != nil {
		return 0, err
	}

	_, err := adc.Write([]byte{CMDRDATA})
	if err != nil {
		return 0, errors.Join(err, adc.setCSHigh())
	}

	time.Sleep(200 * time.Microsecond)

	// Read 3 bytes
	buf := make([]byte, 3)
	_, err = adc.Read(buf)
	if err != nil {
		return 0, errors.Join(err, adc.setCSHigh())
	}

	// Combine 24 bits into signed 32
	raw := Convert24To32(buf)

	return raw, adc.setCSHigh()
}
