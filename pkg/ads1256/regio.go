package ads1256

import (
	"errors"
	"fmt"
	"time"
)

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

// writeRegister writes a single register [regAddr], with the given value.
func (adc *ADS1256) writeRegister(regAddr, value byte) error {
	if regAddr >= NumRegisters {
		return fmt.Errorf("invalid register address 0x%02X", regAddr)
	}
	if err := adc.setCSLow(); err != nil {
		return err
	}

	// If in continuous read mode, must send SDATAC first
	if adc.continuousMode.Load() {
		if _, err := adc.Write([]byte{CMD_SDATAC}); err != nil {
			return errors.Join(err, adc.setCSHigh())
		}
		adc.continuousMode.Store(false)

		time.Sleep(100 * time.Microsecond)
	}

	// WREG: 0x50 + regAddr
	cmd := CMD_WREG | (regAddr & 0x0F)

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
	if adc.continuousMode.Load() {
		if _, err := adc.Write([]byte{CMD_SDATAC}); err != nil {
			return 0, err
		}
		adc.continuousMode.Store(false)
		time.Sleep(100 * time.Microsecond)
	}

	// RREG: 0x10 + regAddr
	cmd := CMD_RREG | (regAddr & 0x0F)

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
