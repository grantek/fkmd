package main

import (
	//"encoding/hex"
	//"errors"
	"flag"
	"fmt"
	//"io"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	//"regexp"
	"strings"

	"github.com/grantek/fkmd/gbcf"
	"github.com/jacobsa/go-serial/serial"
)

var (
	elog *log.Logger //Always output to stderr
	ilog *log.Logger //Verbose output
	dlog *log.Logger //Debug output
)

func usage() {
	fmt.Println("sfgb usage:")
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	var (
		err error
	)

	//options
	port := flag.String("port", "/dev/ttyUSB0", "serial port to use (/dev/ttyUSB0, etc)")

	baud := flag.Uint("baud", 185000, "Baud rate")
	/*
		//serial options, shouldn't be needed
		even := flag.Bool("even", false, "enable even parity")
		odd := flag.Bool("odd", false, "enable odd parity")
		rs485 := flag.Bool("rs485", false, "enable RS485 RTS for direction control")
		rs485HighDuringSend := flag.Bool("rs485_high_during_send", false, "RTS signal should be high during send")
		rs485HighAfterSend := flag.Bool("rs485_high_after_send", false, "RTS signal should be high after send")
		stopbits := flag.Uint("stopbits", 1, "Stop bits")
		databits := flag.Uint("databits", 8, "Data bits")
		chartimeout := flag.Uint("chartimeout", 200, "Inter Character timeout (ms)")
		minread := flag.Uint("minread", 0, "Minimum read count")
	*/

	//baud := new(uint)
	//*baud = 115200
	//even := new(bool); *even = false
	//odd := new(bool); *odd = false
	parity := serial.PARITY_NONE
	rs485 := new(bool)
	*rs485 = false
	rs485HighDuringSend := new(bool)
	*rs485HighDuringSend = false
	rs485HighAfterSend := new(bool)
	*rs485HighAfterSend = false
	stopbits := new(uint)
	*stopbits = 1
	databits := new(uint)
	*databits = 8
	chartimeout := new(uint)
	*chartimeout = 3000
	minread := new(uint)
	*minread = 0

	//sfgb options
	rominfo := flag.Bool("rominfo", false, "Print ROM info")
	readrom := flag.Bool("readrom", false, "Read and save ROM")
	writerom := flag.Bool("writerom", false, "(Flash cart only) Write ROM data to flash")
	readram := flag.Bool("readram", false, "Read and save RAM")
	writeram := flag.Bool("writeram", false, "Write supplied RAM data to cartridge")
	autoname := flag.Bool("autoname", false, "Read ROM name and generate filenames to save ROM/RAM data")
	//mbc := flag.String("mbc", "auto", "")
	romfile := flag.String("romfile", "", "File to save or read ROM data (- for STDOUT/STDIN)")
	ramfile := flag.String("ramfile", "", "File to save or read RAM data (- for STDOUT/STDIN)")
	ramsize := flag.Int("ramsize", 0, "Size of RAM (0 to autodetect)")
	romsize := flag.Int("romsize", 0, "Size of ROM (0 to autodetect)")
	verbose := flag.Bool("verbose", false, "Output info logs to stderr")
	debug := flag.Bool("debug", false, "Output debug logs to stderr (implies verbose)")

	flag.Parse()

	elog = log.New(os.Stderr, "", log.Lshortfile)
	if *verbose || *debug {
		ilog = log.New(os.Stderr, "", log.Lshortfile)
	} else {
		ilog = log.New(ioutil.Discard, "", 0)
	}

	if *debug {
		dlog = log.New(os.Stderr, "", log.Lshortfile)
	} else {
		dlog = log.New(ioutil.Discard, "", 0)
	}
	if *port == "" {
		elog.Println("Must specify port")
		usage()
	}

	if *readram && *writeram {
		elog.Println("Can't read and write cartridge RAM in one invocation")
		usage()
	}

	if *readrom && *writerom {
		elog.Println("Can't read and write cartridge ROM in one invocation")
		usage()
	}

	if (*romfile != "" || *ramfile != "") && *autoname {
		elog.Println("Can't supply file names when autoname is used")
		usage()
	}

	if (*readrom || *writerom) && (*romfile == "" && !*autoname) {
		elog.Println("No ROM file name supplied")
		usage()
	}

	if (*readram || *writeram) && (*ramfile == "" && !*autoname) {
		elog.Println("No RAM file name supplied")
		usage()
	}

	if !*readrom && !*writerom && !*readram && !*writeram && !*rominfo {
		elog.Println("No action specified")
		usage()
	}

	options := serial.OpenOptions{
		PortName:               *port,
		BaudRate:               *baud,
		DataBits:               *databits,
		StopBits:               *stopbits,
		MinimumReadSize:        *minread,
		InterCharacterTimeout:  *chartimeout,
		ParityMode:             parity,
		Rs485Enable:            *rs485,
		Rs485RtsHighDuringSend: *rs485HighDuringSend,
		Rs485RtsHighAfterSend:  *rs485HighAfterSend,
	}

	var d = &gbcf.GBCF{}
	d.SetOptions(options)
	//var mdc memcart.MemCart
	//mdc, err = d.MemCart()

	err = d.Connect()
	if err != nil {
		elog.Print("Error opening serial port: ", err)
		os.Exit(-1)
	} else {
		defer d.Disconnect()
	}

	var dci *gbcf.DeviceCartInfo
	var gbci *gbcf.GBCartInfo
	if *rominfo || *autoname || (*ramsize == 0) || (*romsize == 0) {
		fv, err := d.ReadDeviceStatus()
		if err != nil {
			elog.Printf("ReadDeviceStatus: %v", err)
			os.Exit(-1)
		}
		if *rominfo {
			b, err := json.MarshalIndent(fv, "", "  ")
			if err != nil {
				dlog.Println("MarshallIndent: ", err)
			}
			dlog.Printf("Device status:\n%s\n", string(b))
			fmt.Printf("Device Firmware version: %d%d.%d%d\n", fv.Ver11, fv.Ver12, fv.Ver21, fv.Ver22)
		}

		_, dci, err = d.ReadStatus()
		if err != nil {
			elog.Println(err)
			os.Exit(-1)
		}
		if *rominfo {
			b, err := json.MarshalIndent(dci, "", "  ")
			if err != nil {
				dlog.Println("MarshallIndent: ", err)
			}
			dlog.Printf("Raw cart status:\n%s\n", string(b))

			gbci = dci.GBCartInfo()
			b, err = json.MarshalIndent(gbci, "", "  ")
			if err != nil {
				fmt.Println("MarshallIndent: ", err)
			}
			fmt.Printf("Cart status:\n%s\n", string(b))
		}
		if *romsize == 0 {
			*romsize = gbcf.RomSizeBytes[dci.ROMSize]
		}
		if *ramsize == 0 {
			*ramsize = gbcf.RamSizeBytes[dci.RAMSize]
		}
	}
	if *autoname {
		r := strings.NewReplacer("/", "_", " ", "_")
		cartname := r.Replace(strings.TrimSpace(string(gbci.GameNamePrintable)))
		if cartname == "" {
			cartname = "game"
		}
		suffix := "gb"
		if dci.CGB {
			suffix = "gbc"
		}
		*romfile = cartname + "." + suffix
		*ramfile = cartname + ".sav"
	}

	if *readram {
		if *ramsize == 0 {
			elog.Println("Cartridge RAM not detected (force attempt to read by setting explicit -ramsize).")
			os.Exit(1)
		}
		dlog.Printf("Using ramfile: %s\n", *ramfile)
		b := make([]byte, *ramsize)
		err = d.ReadRAM(b)
		if err != nil {
			elog.Println(err)
		}
		ioutil.WriteFile(*ramfile, b, 0644)
	}

	if *writeram {
		if *ramsize == 0 {
			elog.Println("Cartridge RAM not detected (force attempt to write by setting explicit -ramsize).")
			os.Exit(1)
		}
		dlog.Printf("Using ramfile: %s\n", *ramfile)
		b, err := ioutil.ReadFile(*ramfile)
		if err != nil {
			elog.Print(err)
			os.Exit(1)
		}
		have := len(b)
		if have > *ramsize {
			elog.Printf("ramfile (%d bytes) > ramsize (%d bytes), write will be limited to ramsize.", have, *ramsize)
			b = b[:*ramsize]
		}
		if have < *ramsize {
			elog.Printf("ramfile (%d bytes) < ramsize (%d bytes).", have, *ramsize)
		}
		err = d.WriteRAM(b)
		if err != nil {
			elog.Print(err)
			os.Exit(1)
		}
	}

	if *readrom {
		if *romsize == 0 {
			elog.Println("Cartridge ROM not detected (force attempt to read by setting explicit -romsize).")
			os.Exit(1)
		}
		dlog.Printf("Using romfile: %s\n", *ramfile)
		b := make([]byte, *romsize)
		err = d.ReadROM(b)
		if err != nil {
			elog.Println(err)
		}
		ioutil.WriteFile(*romfile, b, 0644)
	}

	if *writerom {
		//WriteRom(mdc, *romfile)
		elog.Println("writerom not implemented")
	}
}
