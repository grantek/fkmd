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
	//"strings"

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

	//fkmd options
	rominfo := flag.Bool("rominfo", false, "Print ROM info")
	//readrom := flag.Bool("readrom", false, "Read and output ROM")
	//writerom := flag.Bool("writerom", false, "(Flash cart only) Write ROM data to flash")
	//readram := flag.Bool("readram", false, "Read and output RAM")
	//writeram := flag.Bool("writeram", false, "Write supplied RAM data to cartridge")
	//autoname := flag.Bool("autoname", false, "Read ROM name and generate filenames to save ROM/RAM data")
	//romfile := flag.String("romfile", "", "File to save or read ROM data")
	//ramfile := flag.String("ramfile", "", "File to save or read RAM data")
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

	/*
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
	*/

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

	var d = &gbcf.Gbcf{}
	d.SetOptions(options)
	//var mdc memcart.MemCart
	//mdc, err = d.MemCart()

	err = d.Connect()
	if err != nil {
		elog.Println("Error opening serial port: ", err)
		os.Exit(-1)
	} else {
		defer d.Disconnect()
	}

	/*
		if *readram {
			//ReadRam(mdc, *ramfile, *autoname)
			elog.Println("readram not implemented")
		}

		if *writeram {
			//WriteRam(mdc, *ramfile)
			elog.Println("writeram not implemented")
		}

		if *readrom {
			//ReadRom(mdc, *romfile, *autoname)
			elog.Println("readrom not implemented")
		}

		if *writerom {
			//WriteRom(mdc, *romfile)
			elog.Println("writerom not implemented")
		}
	*/

	if *rominfo {
		ds, err := d.ReadDeviceStatus()
		if err != nil {
			elog.Printf("ReadDeviceStatus: %v", err)
		}
		fmt.Println("Device status:")
		b, err := json.MarshalIndent(ds, "", "  ")
		if err != nil {
			fmt.Println("MarshallIndent: ", err)
		}
		fmt.Println(string(b))
		ds, err = d.ReadStatus()
		if err != nil {
			elog.Println(err)
		}
		fmt.Println("Full status:")
		b, err = json.MarshalIndent(ds, "", "  ")
		if err != nil {
			fmt.Println("MarshallIndent: ", err)
		}
		fmt.Println(string(b))
	}
}
