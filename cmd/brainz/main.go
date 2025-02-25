package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	ft232h2 "github.com/ardnew/ft232h"
	"github.com/rs/zerolog"
	"github.com/yunginnanet/ftdi-ads1256/pkg/ads1256"
	"github.com/yunginnanet/ftdi-ads1256/pkg/ft232h"
	"os"
	"strconv"
	"strings"
	"time"
)

var log zerolog.Logger

func init() {
	cw := zerolog.ConsoleWriter{Out: os.Stdout}
	log = zerolog.New(cw).With().Timestamp().Logger()
}

func pCheck(csi, dri, pwi *uint) {
	for _, pu := range []*uint{dri, pwi, csi} {
		n := ""
		switch pu {
		case dri:
			n = "DRDY"
		case pwi:
			n = "PWDN"
		case csi:
			n = "CS"
		}
		p := ft232h2.CPin(*pu)
		if !p.Valid() {
			log.Fatal().Msgf("invalid pin: %s", p)
		}
		log.Info().Str("caller", n).Msg(p.String())
	}

	os.Exit(0)
}

func strToChannelPairs(str string) ([]ads1256.ChannelPair, error) {
	nums := make([]int, 0)
	split := strings.Split(str, ",")

	var err error

	for _, s := range split {
		var n int
		if n, err = strconv.Atoi(s); err != nil {
			return nil, fmt.Errorf("failed to parse comma separated channel numbers: %w", err)
		}
		if n < 0 || n > 7 {
			return nil, fmt.Errorf("channel number %d out of range", n)
		}
		nums = append(nums, n)
	}

	pairs := make([]ads1256.ChannelPair, len(nums))

	for i, n := range nums {
		pairs[i] = ads1256.ChannelPair{
			Pos: intToChannel[n],
			Neg: ads1256.CH_AINCOM,
		}
	}

	return pairs, nil
}

var intToChannel = map[int]ads1256.Channel{
	0: ads1256.CH_AIN0,
	1: ads1256.CH_AIN1,
	2: ads1256.CH_AIN2,
	3: ads1256.CH_AIN3,
	4: ads1256.CH_AIN4,
	5: ads1256.CH_AIN5,
	6: ads1256.CH_AIN6,
	7: ads1256.CH_AIN7,
}

func flags() (ftindex int, cs uint, drdy uint, pwdn uint, channels []ads1256.ChannelPair) {
	fti := flag.Int("FT232H", 0, "FT232H Index")
	csi := flag.Uint("CS", 0x10, "Chip Select (SPI, Digital)")
	dri := flag.Uint("DRDY", 0x01, "Data Ready (GPIO)")
	pwi := flag.Uint("PWDN", 0x40, "Power Down (GPIO)")
	channelsStr := flag.String("channels", "0,1,2,3,4,5,6,7", "Comma-separated list of channels to scan")
	pinCheck := flag.Bool("pin-check", false, "Check GPIO pin validity and debug positions, then exit")
	flag.Parse()
	if !*pinCheck {
		var err error
		if channels, err = strToChannelPairs(*channelsStr); err != nil {
			log.Fatal().Err(err).Msg("failed to parse channel numbers")
		}
		return *fti, *csi, *dri, *pwi, channels
	}
	pCheck(csi, dri, pwi)
	return 0, 0, 0, 0, nil
}

func checkPin(serial *ft232h.FT232H, pin ft232h2.CPin, old bool) bool {
	hl, err := serial.GPIO.Get(pin)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to read pin %s", pin)
	}
	return hl != old
}

func hlStr(hl bool) string {
	if hl {
		return "high"
	}
	return "low"
}

func printOnPinChange(serial *ft232h.FT232H, pin ft232h2.CPin, hl bool, name string, ctx context.Context) {
	changes := 0
	last := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if checkPin(serial, pin, hl) {
			changes++
			hl = !hl
		}
		if changes-last > 100 {
			log.Info().Str("caller", name).Str("state", hlStr(hl)).
				Msgf("pin %s changed %d times", pin, changes)
			last = changes
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func setGPIOPins(serial *ft232h.FT232H, drdy, cs, pwdn uint) {
	log.Debug().
		Uint8("drdy", uint8(drdy)).
		Uint8("pwdn", uint8(pwdn)).
		Uint8("cs", uint8(cs)).
		Msg("setting gpio pins")

	log.Trace().Str("caller", "drdy").Msgf("setting to: %d", drdy)
	err1 := serial.SetDRDY(drdy)

	log.Trace().Str("caller", "pwdn").Msgf("setting to: %d", pwdn)
	err2 := serial.SetPWDN(pwdn)

	log.Trace().Str("caller", "cs").Msgf("setting to: %d", cs)
	err3 := serial.SetCSPin(cs)

	if err := errors.Join(err1, err2, err3); err != nil {
		log.Fatal().Msgf("failed to set GPIO pins: %v", err)
	}
}

func main() {
	ftindex, cs, drdy, pwdn, channels := flags()

	serial, err := ft232h.ConnectFT232h(ft232h.ByIndex(ftindex))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to FT232H")
	}

	log.Info().Any("info", serial.Info()).
		Msgf("connected to FT232H: %s", serial)

	log.Debug().Msg("initializing GPIO")

	if err = serial.GPIO.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize GPIO")
	}
	setGPIOPins(serial, drdy, cs, pwdn)

	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 480*time.Second)

	/*	csHL, err1 := serial.GPIO.Get(serial.CSPin())
		drHL, err2 := serial.GPIO.Get(serial.DRDYPin())
		pwHL, err3 := serial.GPIO.Get(serial.PWDNPin())

		if errors.Join(err1, err2, err3) != nil {
			log.Fatal().Msg("failed to read initial pin states")
		}

		log.Info().
			Str("cs", hlStr(csHL)).
			Str("drdy", hlStr(drHL)).
			Str("pwdn", hlStr(pwHL)).
			Msg("GPIO states")

		time.Sleep(10 * time.Millisecond)*/

	/*	go printOnPinChange(serial, serial.CSPin(), csHL, "chip select", context.Background())
		go printOnPinChange(serial, serial.DRDYPin(), drHL, "data ready", context.Background())
		go printOnPinChange(serial, serial.PWDNPin(), pwHL, "power down", context.Background())*/

	log.Debug().Msg("initializing SPI")

	spiCfg := serial.FT232H.SPI.GetConfig()
	spiCfg.Clock = 1500000
	spiCfg.CS = ft232h2.C(cs)
	spiCfg.Mode = 0x00000001
	spiCfg.ActiveLow = true

	log.Debug().Any("spiCfg", spiCfg).Msg("pushing SPI config")
	if err = serial.SPI.Config(spiCfg); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize SPI")
	}

	adc := ads1256.NewADS1256(serial)

	cls := func() {
		cancel()
		log.Debug().Msg("closing ADS1256")
		if err = adc.Close(); err != nil {
			log.Fatal().Err(err).Msg("failed to close ADS1256")
			return // unreachable
		}

		log.Info().Msg("closed ADS1256")
	}

	defer cls()

	log.Debug().Msg("resetting ADS1256")
	if err = adc.Reset(); err != nil {
		log.Fatal().Err(err).Msg("failed to reset ADS1256")
	}

	cfg := ads1256.DefaultConfig()
	// cfg.ClkOut = ads1256.ADCON_CLK_DIV1
	cfg.AutoCal = true
	cfg.BufferEn = true
	cfg.DataRate = ads1256.DRATE_DR_2000_SPS
	cfg.PGA = ads1256.ADCON_PGA_16

	log.Debug().Any("config", cfg).Msg("initializing ADS1256")
	if err = adc.Initialize(cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize ADS1256")
	}

	log.Info().Msg("initialized ADS1256")

	time.Sleep(200 * time.Millisecond)

	go func() {
		_, _ = fmt.Scanln()
		println("cancelling")
		cancel()
	}()

	cb := func(chPair ads1256.ChannelPair, code int32) {
		log.Info().Int32("code", code).Any("chPair", chPair).Msg("data callback")
	}

	var chScan *ads1256.ChannelScan

	if chScan, err = adc.ScanChannelsContinuously(ctx, 0, cb, channels...); err != nil {
		log.Fatal().Err(err).Msg("failed to scan channels")
	}

	if err = chScan.Wait(ctx); err != nil {
		if errc := adc.Close(); errc != nil {
			log.Error().Err(errc).Msg("failed to close ADS1256")
		}
		log.Fatal().Msg(err.Error())
		return // unreachable
	}

}
