package ads1256

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
