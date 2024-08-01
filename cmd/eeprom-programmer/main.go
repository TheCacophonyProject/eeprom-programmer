package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/TheCacophonyProject/tc2-hat-controller/eeprom"
	"github.com/alexflint/go-arg"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

//LogLevel string      `arg:"-l, --log-level" default:"info" help:"Set the logging level (debug, info, warn, error)"`

type Args struct {
	MainPCBVersion  string `arg:"--main" help:"Main PCB version"`
	PowerPCBVersion string `arg:"--power" help:"Main PCB version"`
	TouchPCBVersion string `arg:"--touch" help:"Main PCB version"`
	MicPCBVersion   string `arg:"--mic" help:"Main PCB version"`
	AudioOnly       bool   `arg:"--audio-only" help:"Device only for audio recordings, no lepton module"`
}

var version = "<not set>"

func (Args) Version() string {
	return version
}

func procArgs() Args {
	args := Args{}
	arg.MustParse(&args)
	return args
}

func main() {
	if err := runMain(); err != nil {
		log.Fatal(err)
	}
}

func runMain() error {
	args := procArgs()

	/*
		parts := strings.Split(args.PCBVersion, ".")
		if len(parts) != 3 {
			return fmt.Errorf("invalid hardware version '%s'", args.PCBVersion)
		}

		major, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return err
		}
		minor, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return err
		}
		patch, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return err
		}
	*/

	mainPcbVersion, err := eeprom.NewSemVer(args.MainPCBVersion)
	if err != nil {
		return err
	}
	powerPcbVersion, err := eeprom.NewSemVer(args.PowerPCBVersion)
	if err != nil {
		return err
	}
	touchPcbVersion, err := eeprom.NewSemVer(args.TouchPCBVersion)
	if err != nil {
		return err
	}
	micPcbVersion, err := eeprom.NewSemVer(args.MicPCBVersion)
	if err != nil {
		return err
	}

	eepromData := &eeprom.EepromDataV2{
		Version:       2,
		MainPCB:       *mainPcbVersion,
		PowerPCB:      *powerPcbVersion,
		TouchPCB:      *touchPcbVersion,
		MicrophonePCB: *micPcbVersion,
		AudioOnly:     args.AudioOnly,
		ID:            eeprom.GenerateRandomID(),
		Time:          time.Now().Truncate(time.Second),
	}

	data := eepromData.WriteData()

	log.Println("Initializing I2C")
	if _, err := host.Init(); err != nil {
		return err
	}
	bus, err := i2creg.Open("")
	if err != nil {
		return err
	}

	log.Printf("Writing EEPROM data: %+v", eepromData)
	pageLength := 16 // Can only read one page on the eeprom chip at a time
	dataLength := len(data)
	for i := 0; i < dataLength; i += pageLength {
		writeLen := min(pageLength, dataLength-i)
		pageWriteData := data[i : i+writeLen]
		d := append([]byte{byte(i)}, pageWriteData...)
		err = bus.Tx(eeprom.EEPROM_ADDRESS, d, nil)
		if err != nil {
			log.Println("Error writing EEPROM: ", err)
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}

	log.Println("EEPROM data written to.")
	eepromDataLength := len(data)
	readData := []byte{}
	for i := 0; i < eepromDataLength; i += pageLength {
		readLen := min(pageLength, eepromDataLength-i)
		r := make([]byte, readLen)
		err := bus.Tx(eeprom.EEPROM_ADDRESS, []byte{byte(i)}, r)
		if err != nil {
			return err
		}
		readData = append(readData, r...)
		time.Sleep(10 * time.Millisecond)
	}

	if !bytes.Equal(data, readData) {
		return fmt.Errorf("data read from eeprom doesn't match data written to")
	}

	return nil
}
