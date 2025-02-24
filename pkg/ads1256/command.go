package ads1256

import (
	"errors"
	"time"
)

func (adc *ADS1256) sendCommand(cmd byte) error {
	if err := adc.setCSLow(); err != nil {
		return err
	}

	// If switching out of continuous read:
	if adc.continuousMode.Load() && cmd != CMDRDATAC && cmd != CMDRESET {
		if _, err := adc.Write([]byte{CMDSDATAC}); err != nil {
			return errors.Join(err, adc.setCSHigh())
		}
		adc.continuousMode.Store(false)
		time.Sleep(100 * time.Microsecond)
	}

	// Write the command
	_, err := adc.Write([]byte{cmd})
	if err != nil {
		return errors.Join(err, adc.setCSHigh())
	}

	// If we just sent RDATAC
	if cmd == CMDRDATAC {
		adc.continuousMode.Store(true)
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
