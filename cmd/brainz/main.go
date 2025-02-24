package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/yunginnanet/fdtdi-ads1256/pkg/ads1256"
	"github.com/yunginnanet/fdtdi-ads1256/pkg/ft232h"
	ft232h2 "github.com/ardnew/ft232h"
	"os"
	"time"
)

var log zerolog.Logger

func init() {
	cw := zerolog.ConsoleWriter{Out: os.Stdout}
	log = zerolog.New(cw).With().Timestamp().Logger()
}

func flags() (ftindex int, cs uint, drdy uint, pwdn uint) {
	fti := flag.Int("FT232H", 0, "FT232H Index")
	csi := flag.Uint("CS", 0x10, "Chip Select (SPI, Digital)")
	dri := flag.Uint("DRDY", 0x01, "Data Ready (GPIO)")
	pwi := flag.Uint("PWDN", 0x40, "Power Down (GPIO)")
	flag.Parse()
	return *fti, *csi, *dri, *pwi
}

func main() {
	ftindex, cs, drdy, pwdn := flags()

	spi, err := ft232h.ConnectFT232h(ft232h.ByIndex(ftindex))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to FT232H")
	}

	log.Info().Any("info", spi.Info()).
		Msgf("connected to FT232H: %s", spi)

	log.Info().Msg("initializing SPI")

	spiCfg := spi.FT232H.SPI.GetConfig()
	spiCfg.Clock = 1700000
	spiCfg.CS = ft232h2.C(cs)
	spiCfg.Mode = 0x00000001
	spiCfg.ActiveLow = false

	spi.SetPWDN(pwdn)
	spi.SetDRDY(drdy)
	spi.SetCSPin(cs)

	log.Debug().Any("config", spiCfg).Msg("initializing SPI")
	if err = spi.SPI.Config(spiCfg); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize SPI")
	}

	adc := ads1256.NewADS1256(spi)

	cfg := ads1256.DefaultConfig()
	// cfg.ClkOut = ads1256.AdconCL
	// cfg.AutoCal = true
	cfg.BufferEn = true
	cfg.DataRate = ads1256.DRATE_DR_50_SPS
	cfg.PGA = ads1256.ADCON_PGA_1

	log.Debug().Any("config", cfg).Msg("initializing ADS1256")
	if err = adc.Initialize(cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize ADS1256")
	}

	log.Info().Msg("initialized ADS1256")

	time.Sleep(100 * time.Millisecond)

	var regs map[ads1256.Register]byte

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	go func() {
		_, _ = fmt.Scanln()
		println("cancelling")
		cancel()
	}()

fl:
	for {
		select {
		case <-ctx.Done():
			break fl
		default:
		}

		/*		if err = adc.WaitDRDY(); err != nil {
				log.Error().Err(err).Msg("failed to wait for ADS1256 DRDY")
				break fl
			}*/

		if regs, err = adc.ReadAllRegisters(); err != nil {
			log.Error().Err(err).Msg("failed to read ADS1256 registers")
			break fl
		}

		log.Info().Any("values", regs).Msg("ADS1256 Registers")

	}

	if err = adc.Close(); err != nil {
		log.Fatal().Err(err).Msg("failed to close ADS1256")
	}

	log.Info().Msg("closed ADS1256")
}
