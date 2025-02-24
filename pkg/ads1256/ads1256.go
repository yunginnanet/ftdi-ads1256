package ads1256

import (
	"errors"
	"fmt"
	"io"
	"sync"
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

	// We track whether we're in "Read Data Continuously" mode, so we can exit if needed
	continuousMode bool
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
		spi: spi,
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

	// Issue hardware or software reset if desired:
	if err := adc.reset(); err != nil {
		adc.mu.Unlock()
		return err
	}

	// Wait a bit for device to run internal power-up routines
	time.Sleep(50 * time.Millisecond) // 30ms is typical after hardware reset

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
	// adconVal |= AdconSDCSOff
	adconVal |= AdconSDCS2uA

	// set PGA
	adconVal |= (cfg.PGA & 0x07)
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

func (adc *ADS1256) LastReadRegister(reg Register) byte {
	adc.mu.RLock()
	b := adc.regLR[reg]
	adc.mu.RUnlock()
	return b
}

func (adc *ADS1256) Registers() map[Register]byte {
	adc.mu.RLock()
	r := make(map[Register]byte, NumRegisters)
	for reg, val := range adc.regLR {
		r[Register(reg)] = val
	}
	adc.mu.RUnlock()
	return r
}

func (adc *ADS1256) Close() error {
	err := adc.reset()
	err = errors.Join(err, adc.Standby())
	err = errors.Join(err, adc.PowerDown())
	err = errors.Join(err, adc.spi.Close())
	return adc.spi.PowerDown()
}

// reset triggers a software reset using the RESET command
func (adc *ADS1256) reset() error {
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

// PowerDown pulls the PWDN pin low if setPWDN is provided.
// Holding SYNC/PDWN low for 20 DRDY cycles also powers down the chip.
func (adc *ADS1256) PowerDown() error {
	return adc.spi.PowerDown() // drive PWDN pin low
}

// PowerUp pulls the PWDN pin high if setPWDN is provided.
func (adc *ADS1256) PowerUp() error {
	return adc.spi.PowerUp() // drive PWDN pin high
}

// SingleConversion issues a Sync, WakeUp, then RDATA flow to read one sample.
// Often used in "one-shot" mode. The user typically calls Standby() first,
// then SingleConversion() each time they want a measurement.
func (adc *ADS1256) SingleConversion() (int32, error) {
	adc.mu.Lock()

	// SYNC
	if err := adc.sendCommand(CMDSYNC); err != nil {
		adc.mu.Unlock()
		return 0, err
	}

	// WAKEUP
	if err := adc.sendCommand(CMDWAKEUP); err != nil {
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

// ReadDataByCommand performs RDATA to get a single 24-bit result from the device.
func (adc *ADS1256) ReadDataByCommand() (int32, error) {
	adc.mu.Lock()
	i3, err := adc.readDataByCommand()
	adc.mu.Unlock()
	return i3, err
}

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
	raw := convert24to32(buf)

	return raw, adc.setCSHigh()
}

// ----- Lower-level register reads/writes:

// writeRegister writes a single register [regAddr], with the given value.
func (adc *ADS1256) writeRegister(regAddr, value byte) error {
	if regAddr >= NumRegisters {
		return fmt.Errorf("invalid register address 0x%02X", regAddr)
	}
	if err := adc.setCSLow(); err != nil {
		return err
	}

	// If in continuous read mode, must send SDATAC first
	if adc.continuousMode {
		if _, err := adc.Write([]byte{CMDSDATAC}); err != nil {
			return errors.Join(err, adc.setCSHigh())
		}
		adc.continuousMode = false

		time.Sleep(100 * time.Microsecond)
	}

	// WREG: 0x50 + regAddr
	cmd := CMDWREG | (regAddr & 0x0F)

	// second byte: # of registers -1. We only do one register => 0
	out := []byte{cmd, 0x00, value}
	if _, err := adc.Write(out); err != nil {
		return errors.Join(err, adc.setCSHigh())
	}

	// Delay T6 might be needed.
	time.Sleep(50 * time.Microsecond)

	adc.regLW[regAddr] = value
	return adc.setCSHigh()
}

// readRegister reads a single register [regAddr].
func (adc *ADS1256) readRegister(regAddr byte) (byte, error) {
	if regAddr >= NumRegisters {
		return 0, fmt.Errorf("invalid register address 0x%02X", regAddr)
	}

	if err := adc.setCSLow(); err != nil {
		return 0, err
	}

	// If in continuous read mode, must send SDATAC first
	if adc.continuousMode {
		if _, err := adc.Write([]byte{CMDSDATAC}); err != nil {
			return 0, err
		}
		adc.continuousMode = false
		time.Sleep(100 * time.Microsecond)
	}

	// RREG: 0x10 + regAddr
	cmd := CMDRREG | (regAddr & 0x0F)

	// 2nd byte => # of registers -1 => 0
	out := []byte{cmd, 0x00}
	if _, err := adc.Write(out); err != nil {
		return 0, err
	}

	time.Sleep(50 * time.Microsecond)

	// read single register
	buf := get1Byte()

	_, err := adc.Read(buf)

	if err != nil {
		put1Byte(buf)
		return 0, err
	}

	copy(adc.regLR[regAddr:], buf)

	put1Byte(buf)
	return adc.regLR[regAddr], nil
}

func (adc *ADS1256) ReadAllRegisters() (registers map[Register]byte, err error) {
	adc.mu.Lock()
	err = adc.readAllRegisters()
	if err == nil {
		registers = make(map[Register]byte, NumRegisters)
		for reg, val := range adc.regLR {
			registers[Register(reg)] = val
		}
	}
	adc.mu.Unlock()
	return
}

// readAllRegisters is optional, but can be handy for debug
func (adc *ADS1256) readAllRegisters() error {
	for reg := byte(0); reg < NumRegisters; reg++ {
		val, err := adc.readRegister(reg)
		if err != nil {
			return err
		}
		adc.regLR[reg] = val // done in readRegister
	}

	return nil
}

// ----- Commands:

func (adc *ADS1256) sendCommand(cmd byte) error {
	if err := adc.setCSLow(); err != nil {
		return err
	}

	// If switching out of continuous read:
	if adc.continuousMode && cmd != CMDRDATAC {
		if cmd != CMDRESET { // if reset is called, it also ends continuous mode
			if _, err := adc.Write([]byte{CMDSDATAC}); err != nil {
				return errors.Join(err, adc.setCSHigh())
			}
			adc.continuousMode = false
			time.Sleep(100 * time.Microsecond)
		}
	}

	// Write the command
	_, err := adc.Write([]byte{cmd})
	if err != nil {
		return errors.Join(err, adc.setCSHigh())
	}

	// If we just sent RDATAC
	if cmd == CMDRDATAC {
		adc.continuousMode = true
	}

	// Some commands need extra wait or wait for DRDY
	switch cmd {
	// Must wait for DRDY after these commands.
	case CMDRESET, CMDSELFCAL, CMDSELFOCAL, CMDSELFGCAL,
		CMDSYSOCAL, CMDSYSGCAL, CMDSTANDBY:

		// STANDBY won't come out until WAKEUP, so you might not wait DRDY for that.
		if cmd == CMDSTANDBY {
			return adc.setCSHigh()
		}

		/*if err = adc.spi.WaitDRDY(); err != nil {
			return errors.Join(err, adc.setCSHigh())
		}*/
	}

	return nil
}

// ----- Helper / utility

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

// convert24to32 interprets a 3-byte, 24-bit signed value
// in two's complement form, MSB first, as a 32-bit int.
func convert24to32(data []byte) int32 {
	// data[0] is MSB. If top bit set => negative
	var u32 uint32
	u32 |= uint32(data[0]) << 16
	u32 |= uint32(data[1]) << 8
	u32 |= uint32(data[2])

	// sign extension
	if (u32 & 0x800000) != 0 {
		u32 |= 0xFF000000
	}
	return int32(u32)
}
