package ads1256

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type Register byte

// SPI interface allows for different SPI implementations.
type SPI interface {
}

// ADS1256 provides high-level control over a TI ADS1256 ADC.
//
// It uses an [io.ReadWriter] for SPI communication and simple callbacks/interfaces
// for DRDY and PWDN pin handling. You can adapt it further for your own GPIO usage.
type ADS1256 struct {
	mu  sync.Mutex // Synchronize concurrent operations
	spi SPI        // SPI interface

	waitDRDY func() error     // Called to wait for DRDY = LOW
	setPWDN  func(level bool) // Drive PWDN pin. level=true => high, etc.
	setCS    func(level bool) // Drive chip-select pin if you want outside control

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

// Constants from the datasheet

// Register Addresses
const (
	// RegSTATUS is the STATUS register
	RegSTATUS = 0x00
	// RegMUX is the multiplexer register
	RegMUX = 0x01
	// RegADCON is the ADCON register
	RegADCON = 0x02
	// RegDRATE is the data rate register
	RegDRATE = 0x03
	// RegIO is the I/O register
	RegIO = 0x04
	// RegOFC0 is the offset calibration register 0
	RegOFC0 = 0x05
	// RegOFC1 is the offset calibration register 1
	RegOFC1 = 0x06
	// RegOFC2 is the offset calibration register 2
	RegOFC2 = 0x07
	// RegFSC0 is the full-scale calibration register 0
	RegFSC0 = 0x08
	// RegFSC1 is the full-scale calibration register 1
	RegFSC1 = 0x09
	// RegFSC2 is the full-scale calibration register 2
	RegFSC2 = 0x0A

	// NumRegisters is the total number of registers.
	NumRegisters = 0x0B // 11 total (0 through 0x0A)*/
)

// Command Opcodes
const (
	CMDWakeUp   = 0x00
	CMDRDATA    = 0x01
	CMDRDATAC   = 0x03
	CMDSDATAC   = 0x0F
	CMDRREG     = 0x10 // 0x10 + (reg & 0x0F)
	CMDWREG     = 0x50 // 0x50 + (reg & 0x0F)
	CMDSELFCAL  = 0xF0
	CMDSELFOCAL = 0xF1
	CMDSELFGCAL = 0xF2
	CMDSYSOCAL  = 0xF3
	CMDSYSGCAL  = 0xF4
	CMDSYNC     = 0xFC
	CMDSTANDBY  = 0xFD
	CMDRESET    = 0xFE
	CMDWAKEUP   = 0xFF
)

// DRATE Register Byte constants (for DRATE register):
// fCLKIN assumed = 7.68 MHz. Data Rate is from Table in data sheet.
const (
	DRATE_DR_2p5_SPS   = 0x00
	DRATE_DR_5_SPS     = 0x01
	DRATE_DR_10_SPS    = 0x02
	DRATE_DR_15_SPS    = 0x03
	DRATE_DR_25_SPS    = 0x04
	DRATE_DR_30_SPS    = 0x05
	DRATE_DR_50_SPS    = 0x06
	DRATE_DR_60_SPS    = 0x07
	DRATE_DR_100_SPS   = 0x08
	DRATE_DR_500_SPS   = 0x09
	DRATE_DR_1000_SPS  = 0x0A
	DRATE_DR_2000_SPS  = 0x0B
	DRATE_DR_3750_SPS  = 0x0C
	DRATE_DR_7500_SPS  = 0x0D
	DRATE_DR_15000_SPS = 0x0E
	DRATE_DR_30000_SPS = 0x0F
)

// Bits for the STATUS register
const (
	StatusORDERbit = 0x08 // (bit3)
	StatusACALbit  = 0x04 // (bit2)
	StatusBUFENbit = 0x02 // (bit1)
	StatusDRDYbit  = 0x01 // (bit0, read-only)
)

// Bits for ADCON register
const (
	// AdconCLKOff SCLK freq outputs
	AdconCLKOff  = 0x00
	AdconCLKDiv1 = 0x20
	AdconCLKDiv2 = 0x40
	AdconCLKDiv4 = 0x60

	// AdconSDCSOff SDCS sensor detect bits
	AdconSDCSOff   = 0x00
	AdconSDCS0p5uA = 0x08
	AdconSDCS2uA   = 0x10
	AdconSDCS10uA  = 0x18

	// ADCON_PGA_1 PGA bits
	ADCON_PGA_1  = 0x00
	ADCON_PGA_2  = 0x01
	ADCON_PGA_4  = 0x02
	ADCON_PGA_8  = 0x03
	ADCON_PGA_16 = 0x04
	ADCON_PGA_32 = 0x05
	ADCON_PGA_64 = 0x06
)

// NewADS1256 constructs an ADS1256 object with the given SPI and optional pin callbacks.
func NewADS1256(spi io.ReadWriter, waitDRDY func() error, setPWDN, setCS func(bool)) *ADS1256 {
	return &ADS1256{
		// SPI:      spi,
		waitDRDY: waitDRDY,
		setPWDN:  setPWDN,
		setCS:    setCS,
	}
}

// Initialize sets up the device with the provided config.
// Typically call it once at start-up. The ADS1256 automatically
// does a self-cal on power-up, but you can do an additional SELFCAL if needed.
func (adc *ADS1256) Initialize(cfg Config) error {
	adc.mu.Lock()
	defer adc.mu.Unlock()

	if adc.setCS != nil {
		adc.setCS(true) // Deselect at the start
	}

	// Issue hardware or software reset if desired:
	if err := adc.reset(); err != nil {
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
	adconVal |= AdconSDCSOff
	// set PGA
	adconVal |= (cfg.PGA & 0x07)
	if err := adc.writeRegister(RegADCON, adconVal); err != nil {
		return err
	}

	if err := adc.writeRegister(RegDRATE, cfg.DataRate); err != nil {
		return err
	}

	// Optionally set up the I/O register if you want to manipulate D0..D3
	// By default, all are inputs. Example: 0xE0 is default => (DIR3..DIR0 =1 => inputs)
	// We'll leave that as default for now.

	// Read them back to confirm (optional)
	if err := adc.readAllRegisters(); err != nil {
		return fmt.Errorf("failed to read back registers: %v", err)
	}

	// Optionally do self-cal
	// ADC automatically self-cals after power-on, but if you want to re-run:
	/*
		if err := adc.sendCommand(CmdSELFCAL); err != nil {
			return err
		}
		if adc.waitDRDY != nil {
			if err := adc.waitDRDY(); err != nil {
				return err
			}
		}
	*/
	return nil
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
func (adc *ADS1256) PowerDown() {
	if adc.setPWDN != nil {
		adc.setPWDN(false) // drive PWDN pin low
	}
}

// PowerUp pulls the PWDN pin high if setPWDN is provided.
func (adc *ADS1256) PowerUp() {
	if adc.setPWDN != nil {
		adc.setPWDN(true)
	}
}

// SingleConversion issues a Sync, WakeUp, then RDATA flow to read one sample.
// Often used in "one-shot" mode. The user typically calls Standby() first,
// then SingleConversion() each time they want a measurement.
func (adc *ADS1256) SingleConversion() (int32, error) {
	adc.mu.Lock()
	defer adc.mu.Unlock()

	// SYNC
	if err := adc.sendCommand(CMDSYNC); err != nil {
		return 0, err
	}
	// WAKEUP
	if err := adc.sendCommand(CMDWAKEUP); err != nil {
		return 0, err
	}
	// Wait for DRDY
	if adc.waitDRDY != nil {
		if err := adc.waitDRDY(); err != nil {
			return 0, err
		}
	}
	// Then read data with RDATA
	return adc.readDataByCommand()
}

// ReadDataByCommand performs RDATA to get a single 24-bit result from the device.
func (adc *ADS1256) ReadDataByCommand() (int32, error) {
	adc.mu.Lock()
	defer adc.mu.Unlock()
	return adc.readDataByCommand()
}

func (adc *ADS1256) readDataByCommand() (int32, error) {
	if err := adc.setCSLow(); err != nil {
		return 0, err
	}
	defer adc.setCSHigh()

	// Send RDATA
	_, err := adc.spiWrite([]byte{CMDRDATA})
	if err != nil {
		return 0, err
	}

	// A small delay T6 needed typically
	time.Sleep(200 * time.Microsecond)

	// Read 3 bytes
	buf := make([]byte, 3)
	_, err = adc.spiRead(buf)
	if err != nil {
		return 0, err
	}

	// Combine 24 bits into signed 32
	raw := convert24to32(buf)
	return raw, nil
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
	defer adc.setCSHigh()

	// If in continuous read mode, must send SDATAC first
	if adc.continuousMode {
		if _, err := adc.spiWrite([]byte{CMDSDATAC}); err != nil {
			return err
		}
		adc.continuousMode = false
		// short delay
		time.Sleep(100 * time.Microsecond)
	}

	// WREG: 0x50 + regAddr
	cmd := byte(CMDWREG | (regAddr & 0x0F))
	// second byte: # of registers -1. We only do one register => 0
	out := []byte{cmd, 0x00, value}
	if _, err := adc.spiWrite(out); err != nil {
		return err
	}

	// Delay T6 might be needed.
	time.Sleep(50 * time.Microsecond)

	adc.regLW[regAddr] = value
	return nil
}

// readRegister reads a single register [regAddr].
func (adc *ADS1256) readRegister(regAddr byte) (byte, error) {
	if regAddr >= NumRegisters {
		return 0, fmt.Errorf("invalid register address 0x%02X", regAddr)
	}

	if err := adc.setCSLow(); err != nil {
		return 0, err
	}
	defer adc.setCSHigh()

	// If in continuous read mode, must send SDATAC first
	if adc.continuousMode {
		if _, err := adc.spiWrite([]byte{CMDSDATAC}); err != nil {
			return 0, err
		}
		adc.continuousMode = false
		time.Sleep(100 * time.Microsecond)
	}

	// RREG: 0x10 + regAddr
	cmd := byte(CMDRREG | (regAddr & 0x0F))
	// 2nd byte => # of registers -1 => 0
	out := []byte{cmd, 0x00}
	if _, err := adc.spiWrite(out); err != nil {
		return 0, err
	}
	// small delay t6
	time.Sleep(50 * time.Microsecond)

	// read single register
	buf := make([]byte, 1)
	_, err := adc.spiRead(buf)
	if err != nil {
		return 0, err
	}
	val := buf[0]
	adc.regLR[regAddr] = val
	return val, nil
}

// readAllRegisters is optional, but can be handy for debug
func (adc *ADS1256) readAllRegisters() error {
	for reg := byte(0); reg < NumRegisters; reg++ {
		val, err := adc.readRegister(reg)
		if err != nil {
			return err
		}
		adc.regLR[reg] = val
	}
	return nil
}

// ----- Commands:

func (adc *ADS1256) sendCommand(cmd byte) error {
	if err := adc.setCSLow(); err != nil {
		return err
	}
	defer adc.setCSHigh()

	// If switching out of continuous read:
	if adc.continuousMode && cmd != CMDRDATAC {
		if cmd != CMDRESET { // if reset is called, it also ends continuous mode
			if _, err := adc.spiWrite([]byte{CMDSDATAC}); err != nil {
				return err
			}
			adc.continuousMode = false
			time.Sleep(100 * time.Microsecond)
		}
	}

	// Write the command
	_, err := adc.spiWrite([]byte{cmd})
	if err != nil {
		return err
	}

	// If we just sent RDATAC
	if cmd == CMDRDATAC {
		adc.continuousMode = true
	}

	// Some commands need extra wait or wait for DRDY
	switch cmd {
	case CMDRESET, CMDSELFCAL, CMDSELFOCAL, CMDSELFGCAL,
		CMDSYSOCAL, CMDSYSGCAL, CMDSTANDBY:
		// Must wait for DRDY after these commands.
		// STANDBY won't come out until WAKEUP, so you might not wait DRDY for that.
		if cmd == CMDSTANDBY {
			return nil
		}
		if adc.waitDRDY != nil {
			err := adc.waitDRDY()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ----- Helper / utility

func (adc *ADS1256) setCSLow() error {
	if adc.setCS != nil {
		adc.setCS(false)
	}
	return nil
}
func (adc *ADS1256) setCSHigh() error {
	if adc.setCS != nil {
		adc.setCS(true)
	}
	return nil
}

func (adc *ADS1256) spiWrite(out []byte) (int, error) {
	// n, err := adc.SPI.Write(out)
	return -1, nil
}
func (adc *ADS1256) spiRead(in []byte) (int, error) {
	// n, err := adc.SPI.Read(in)
	return -1, nil
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
