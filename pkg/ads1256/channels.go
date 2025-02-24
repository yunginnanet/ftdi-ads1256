package ads1256

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type Channel int

//goland:noinspection GoSnakeCaseUsage
const (
	CH_AIN0 Channel = iota
	CH_AIN1
	CH_AIN2
	CH_AIN3
	CH_AIN4
	CH_AIN5
	CH_AIN6
	CH_AIN7
	CH_AINCOM
)

func (c Channel) Byte() byte {
	return byte(c)
}

func (c Channel) String() string {
	switch c {
	case CH_AIN0:
		return "CH_AIN0"
	case CH_AIN1:
		return "CH_AIN1"
	case CH_AIN2:
		return "CH_AIN2"
	case CH_AIN3:
		return "CH_AIN3"
	case CH_AIN4:
		return "CH_AIN4"
	case CH_AIN5:
		return "CH_AIN5"
	case CH_AIN6:
		return "CH_AIN6"
	case CH_AIN7:
		return "CH_AIN7"
	case CH_AINCOM:
		return "CH_AINCOM"
	default:
		return "(invalid channel)"
	}
}

// ChannelPair is a simple struct that holds positive/negative channel identifiers.
type ChannelPair struct {
	Pos Channel
	Neg Channel
}

type ChannelScan struct {
	Interval time.Duration
	done     *atomic.Bool
	running  *atomic.Bool
	pairs    []ChannelPair
	callback DataCallback
	err      []error
	errMu    sync.Mutex
}

func NewChannelScan(interval time.Duration, pairs []ChannelPair, onData DataCallback) *ChannelScan {
	return &ChannelScan{
		Interval: interval,
		done:     &atomic.Bool{},
		running:  &atomic.Bool{},
		pairs:    pairs,
		callback: onData,
		err:      make([]error, 0),
	}
}

func (cs *ChannelScan) addErr(err error) {
	if err == nil {
		return
	}
	cs.errMu.Lock()
	cs.err = append(cs.err, err)
	if len(cs.err) > 50 {
		cs.done.Store(true)
	}
	cs.errMu.Unlock()
}

func (cs *ChannelScan) Err() error {
	cs.errMu.Lock()
	if len(cs.err) == 0 {
		return nil
	}
	err := errors.Join(cs.err...)
	cs.errMu.Unlock()
	return fmt.Errorf("channel scan errors: %w", err)
}

func (cs *ChannelScan) Stop() {
	cs.done.Store(true)
}

func (cs *ChannelScan) IsDone() bool {
	return cs.done.Load()
}

func (cs *ChannelScan) Wait(ctx ...context.Context) error {
	ctxDone := func() {
		if len(ctx) < 1 {
			return
		}
		select {
		case <-ctx[0].Done():
		default:
		}
	}

	for !cs.done.Load() {
		ctxDone()
		time.Sleep(10 * time.Millisecond)
	}
	for cs.running.Load() {
		ctxDone()
		time.Sleep(10 * time.Millisecond)
	}

	return cs.Err()
}

type DataCallback func(chPair ChannelPair, code int32)

func (adc *ADS1256) scanChannelPairs(cs *ChannelScan, cancel context.CancelFunc) {
	if cs.done.Load() {
		cancel()
		return
	}

	for _, chPair := range cs.pairs {
		println("scanning", chPair.Pos, chPair.Neg)

		if cs.done.Load() {
			cancel()
			return
		}

		adc.mu.Lock()

		if adc.continuousMode.Load() {
			println("exiting continuous mode")

			// exit existing continuous mode if active
			if err := adc.sendCommand(CMD_SDATAC); err != nil {
				cancel()
				cs.addErr(err)
				adc.mu.Unlock()
				return
			}
			adc.continuousMode.Store(false)
		}

		// set multiplexer to read from the current channel pair
		muxVal := byte((chPair.Pos << 4) | (chPair.Neg & 0x0F))

		fmt.Printf("writing to REG_MUX: %08b\n", muxVal)

		if err := adc.writeRegister(REG_MUX, muxVal); err != nil {
			cancel()
			cs.addErr(err)
			adc.mu.Unlock()
			return
		}

		// maybe ensure single-cycle settling?
		// _ = adc.sendCommand(CMD_SYNC)
		// _ = adc.sendCommand(CMD_WAKEUP)
		// might want to wait DRDY or a small delay.

		println("sending RDATAC")

		// start continuous read
		if err := adc.sendCommand(CMD_RDATAC); err != nil {
			cancel()
			cs.addErr(err)
			adc.mu.Unlock()
			return
		}

		adc.continuousMode.Store(true)

		println("waiting for DRDY")
		cs.addErr(adc.WaitDRDY())

		// read 3 bytes
		// In RDATAC, after DRDY you simply clock out 3 bytes
		rawBuf := get3Bytes()

		println("setting CS low")

		cs.addErr(adc.setCSLow())

		println("reading 3 bytes")

		n, err := adc.Read(rawBuf)
		if err != nil && !errors.Is(err, io.EOF) {
			cs.addErr(err)
		}

		if n != 3 {
			cs.addErr(fmt.Errorf("%w: expected 3 bytes, got %d", io.ErrUnexpectedEOF, n))
		}

		fmt.Printf("rawBuf: %08b\n", rawBuf)

		println("setting CS high")

		cs.addErr(adc.setCSHigh())

		// convert to int32
		code := Convert24To32(rawBuf)

		put3Bytes(rawBuf)

		fmt.Printf("code: %d\nrunning call back\n", code)

		cs.callback(chPair, code)

		adc.mu.Unlock()
	}
}

// ScanChannelsContinuously cycles through a list of channel pairs,
// using "continuous" reads for each pair, then moving to the next.
// The callback function is called after each read with the measured code.
//
// This is "continuous" in the sense that we use RDATAC for each channel,
// but we must exit RDATAC (SDATAC) each time we change the MUX register.
// So effectively, we do repeated RDATAC chunks per channel.
//
// This method spawns a go routine that fires off your callback
func (adc *ADS1256) ScanChannelsContinuously(
	ctx context.Context,
	scanInterval time.Duration,
	onData DataCallback,
	pairs ...ChannelPair,
) (*ChannelScan, error) {
	// Quick check that we have at least one channel pair.
	if len(pairs) == 0 {
		return nil, errors.New("no channels to scan")
	}

	if adc.continuousMode.Load() {
		if err := adc.sendCommand(CMD_SDATAC); err != nil {
			return nil, fmt.Errorf("failed to send SDATAC (continuous mode was already enabled): %w", err)
		}
		adc.continuousMode.Store(false)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	chScan := NewChannelScan(scanInterval, pairs, onData)

	go func() {
		chScan.running.Store(true)
		for {
			select {
			case <-ctx.Done():
				chScan.running.Store(false)
				return
			default:
				if chScan.done.Load() {
					cancel()
					continue
				}
			}
			adc.scanChannelPairs(chScan, cancel)
			time.Sleep(scanInterval)
		}
	}()

	return chScan, nil
}
