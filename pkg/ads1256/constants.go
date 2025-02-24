package ads1256

// Source: https://www.ti.com/lit/ds/symlink/ads1256.pdf

// Register Addresses
//
//goland:noinspection GoSnakeCaseUsage,GoUnusedConst
const (
	// REG_STATUS is the STATUS register
	REG_STATUS = 0x00
	// REG_MUX is the multiplexer register
	REG_MUX = 0x01
	// REG_ADCON is the ADCON register
	REG_ADCON = 0x02
	// REG_DRATE is the data rate register
	REG_DRATE = 0x03
	// REG_IO is the I/O register
	REG_IO = 0x04
	// REG_OFC0 is the offset calibration register 0
	REG_OFC0 = 0x05
	// REG_OFC1 is the offset calibration register 1
	REG_OFC1 = 0x06
	// REG_OFC2 is the offset calibration register 2
	REG_OFC2 = 0x07
	// REG_FSC0 is the full-scale calibration register 0
	REG_FSC0 = 0x08
	// REG_FSC1 is the full-scale calibration register 1
	REG_FSC1 = 0x09
	// REG_FSC2 is the full-scale calibration register 2
	REG_FSC2 = 0x0A

	// NumRegisters is the total number of registers.
	NumRegisters = 0x0B // 11 total (0 through 0x0A)*/
)

// Command Opcodes
//
//goland:noinspection GoSnakeCaseUsage,GoUnusedConst
const (
	CMD_WAKEUP0  = 0x00
	CMD_RDATA    = 0x01
	CMD_RDATAC   = 0x03
	CMD_SDATAC   = 0x0F
	CMD_RREG     = 0x10 // 0x10 + (reg & 0x0F)
	CMD_WREG     = 0x50 // 0x50 + (reg & 0x0F)
	CMD_SELFCAL  = 0xF0
	CMD_SELFOCAL = 0xF1
	CMD_SELFGCAL = 0xF2
	CMD_SYSOCAL  = 0xF3
	CMD_SYSGCAL  = 0xF4
	CMD_SYNC     = 0xFC
	CMD_STANDBY  = 0xFD
	CMD_RESET    = 0xFE
	CMD_WAKEUP   = 0xFF
)

// DRATE Register Byte constants (for DRATE register):
// fCLKIN assumed = 7.68 MHz. Data Rate is from Table in data sheet.
//
//goland:noinspection GoSnakeCaseUsage,GoUnusedConst
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
//
//goland:noinspection GoSnakeCaseUsage,GoUnusedConst
const (
	STATUS_ORDER = 0x08 // (bit3)
	STATUS_ACAL  = 0x04 // (bit2)
	STATUS_BUFEN = 0x02 // (bit1)
	STATUS_DRDY  = 0x01 // (bit0, read-only)
)

// Bits for ADCON register
//
//goland:noinspection GoUnusedConst,GoSnakeCaseUsage,GoCommentStart
const (
	// SCLK freq outputs
	ADCON_CLK_OFF  = 0x00
	ADCON_CLK_DIV1 = 0x20
	ADCON_CLK_DIV2 = 0x40
	ADCON_CLK_DIV4 = 0x60

	// SDCS sensor detect bits
	ADCON_SDCS_OFF   = 0x00
	ADCON_SDCS_0p5uA = 0x08
	ADCON_SDCS_2uA   = 0x10
	ADCON_SDCS_10uA  = 0x18

	// PGA (amplifier) bits
	ADCON_PGA_1  = 0x00
	ADCON_PGA_2  = 0x01
	ADCON_PGA_4  = 0x02
	ADCON_PGA_8  = 0x03
	ADCON_PGA_16 = 0x04
	ADCON_PGA_32 = 0x05
	ADCON_PGA_64 = 0x06
)
