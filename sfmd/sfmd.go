package main

import (
	//"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/grantek/fkmd/krikzz_fkmd"
	"github.com/grantek/fkmd/mdcart"
	"github.com/grantek/fkmd/memcart"
	"github.com/jacobsa/go-serial/serial"
)

var (
	elog *log.Logger //Always output to stderr
	ilog *log.Logger //Verbose output
	dlog *log.Logger //Debug output
)

func usage() {
	fmt.Println("sfmd usage:")
	flag.PrintDefaults()
	os.Exit(-1)
}

//md specific
func ReadRom(mdc memcart.MemCart, romfile string, autoname bool) {
	var (
		romname   string
		romsize   int64
		blocksize int64 = 32768
		f         *os.File
		n         int64 //counter for outer read in bytes
		m         int   //counter for inner read in bytes
		err       error
		mdr       memcart.MemBank
	)
	if autoname {
		romname, err = mdcart.GetRomName(mdc)
		if err != nil {
			panic(err)
		}
		re := regexp.MustCompile("  *")
		romname = re.ReplaceAllString(romname, " ")
		romname = strings.Title(strings.ToLower(strings.TrimSpace(romname)))
		romfile = fmt.Sprintf("%s.bin", romname)
	}

	err = mdc.SwitchBank(0)
	if err != nil {
		panic(err)
	}
	mdr = mdc.CurrentBank()
	if romfile == "-" {
		f = os.Stdout
	} else {
		f, err = os.Create(romfile)
		if err != nil {
			panic(err)
		}
		ilog.Println("Opened", romfile, "for writing")
		defer f.Close()
	}

	romsize = mdr.Size()
	mdr.Seek(0, io.SeekStart)
	buf := make([]byte, blocksize)
	for n = 0; n < romsize; n += blocksize {
		dlog.Printf("Bytes read: %d", n)
		m, err = mdr.Read(buf)
		if err != nil {
			panic(err)
		}
		if int64(m) < blocksize {
			elog.Printf("Short read at %d, expected romsize %d", n+int64(m), romsize)
			break
		}
		f.Write(buf[0:m])
	}
	ilog.Printf("Finished reading, bytes read: %d", n)
}

func ReadRam(mdc memcart.MemCart, ramfile string, autoname bool) {
	var (
		err  error
		f    *os.File
		n, m int
		mdr  memcart.MemBank
	)

	err = mdc.SwitchBank(1)
	if err != nil {
		panic(err)
	}

	mdr = mdc.CurrentBank()
	if mdr == nil {
		panic("Current Bank is nil")
	}

	if ramfile == "" {
		ramfile = "ram.out"
	}

	if ramfile == "-" {
		f = os.Stdout
	} else {
		f, err = os.Create(ramfile)
		if err != nil {
			panic(err)
		}
		ilog.Println("Opened ", ramfile, " for writing")
		defer f.Close()
	}

	ramsize := mdr.Size()
	buf := make([]byte, ramsize)

	for n < int(ramsize) {
		m, err = mdr.Read(buf[n:])
		n += m
		if err != nil || m == 0 {
			panic(fmt.Sprintf("Short RAM read, read %d bytes", n+m))
		}
	}
	ilog.Printf("Read %d bytes", n)
	f.Write(buf)
	ilog.Printf("Ok")
}

func WriteRam(mdc memcart.MemCart, ramfile string) {
	var (
		f         *os.File
		n         int
		ram, ram2 []byte
		err       error
	)
	if mdc.NumBanks() < 2 {
		panic("RAM not detected on cartridge for writing")
	}
	mdc.SwitchBank(1)
	mdr := mdc.CurrentBank()

	if ramfile == "-" {
		f = os.Stdin
	} else {
		f, err = os.Open(ramfile)
		if err != nil {
			panic(err)
		}
		ilog.Println("Opened", ramfile, "for reading")
		defer f.Close()
	}

	ram, err = ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	n, err = mdr.Write(ram)
	if err != nil {
		panic(err)
	}
	ilog.Printf("Wrote %d bytes", n)
	if n < len(ram) {
		elog.Printf("WARNING: wrote %d bytes, input is %d bytes.\n", n, len(ram))
	}
	if int64(n) < mdr.Size() {
		elog.Printf("WARNING: wrote %d bytes, cartridge RAM is %d bytes.\n", n, len(ram))
	}
	ilog.Println("Verify...")
	mdr.Seek(0, io.SeekStart)
	for i := 0; i < n; i++ {
		//TODO: check word length here
		if ram[i] != ram2[i] {
			panic(fmt.Sprintf("Failed verification at byte %d", i))
		}
	}
	ilog.Printf("Verified %d bytes", n)
}

func WriteRom(mdc memcart.MemCart, romfile string) error {
	var (
		romsize  int64
		blocklen int64 = 4096
		f        *os.File
		i        int64
		err      error
		filebuf  []byte
		fblen    int64
	)

	if romfile == "-" {
		f = os.Stdin
		romfile = "stdin"
	} else {
		f, err = os.Open(romfile)
		if err != nil {
			return err
		}
		ilog.Println("Opened", romfile, "for reading")
		defer f.Close()
	}

	filebuf, err = ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	if f != os.Stdin {
		f.Close()
	}

	romsize = int64(len(filebuf))
	ilog.Println("Read %d bytes from file", len(filebuf))

	if romsize%2 == 1 {
		elog.Printf("WARNING: file size in bytes is odd", mdcart.MAX_ROM_SIZE, romsize)
		filebuf = append(filebuf, 0)
		romsize++
	}

	if romsize > mdcart.MAX_ROM_SIZE {
		elog.Printf("WARNING: Max ROM data size is %x, cropping input\n", mdcart.MAX_ROM_SIZE, romsize)
		romsize = mdcart.MAX_ROM_SIZE
	}

	fblen = romsize
	if romsize < 0x8000 {
		return errors.New("File size < 32KiB, pad with 0xFF if required")
	}

	if romsize%0x10000 != 0 {
		romsize = (romsize/0x10000)*0x10000 + 0x10000
	}

	mdc.SwitchBank(0)
	mdr := mdc.CurrentBank()

	//Going to rely on Write() performing block erasure
	ilog.Println("Flash write...")
	for i = 0; i < fblen; i += blocklen {
		if i+blocklen > fblen {
			blocklen = fblen - i
		}
		//TODO: n, err
		mdr.Write(filebuf[i : i+blocklen])
		dlog.Printf("Bytes written: %d", i)
	}

	ilog.Println("Flash verify...")
	rom2 := make([]byte, romsize)

	mdr.Seek(0, io.SeekStart)
	for i = 0; i < fblen; i += blocklen {
		if i+blocklen > fblen {
			blocklen = fblen - i
		}
		mdr.Read(rom2[i : i+blocklen])
		dlog.Printf("Bytes read: %d", i)
	}

	for i = 0; i < fblen; i++ {
		if rom2[i] != filebuf[i] {
			return errors.New(fmt.Sprintf("Verify error at %x", i))
		}
	}

	ilog.Println("OK")
	return nil
}

func main() {
	var (
		err error
	)

	//options
	port := flag.String("port", "/dev/ttyUSB0", "serial port to use (/dev/ttyUSB0, etc)")

	//serial options, shouldn't be needed
	/*
		baud := flag.Uint("baud", 9600, "Baud rate")
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

	baud := new(uint)
	*baud = 9600
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
	*chartimeout = 200
	minread := new(uint)
	*minread = 0

	//fkmd options
	rominfo := flag.Bool("rominfo", false, "Print ROM info")
	readrom := flag.Bool("readrom", false, "Read and output ROM")
	writerom := flag.Bool("writerom", false, "(Flash cart only) Write ROM data to flash")
	readram := flag.Bool("readram", false, "Read and output RAM")
	writeram := flag.Bool("writeram", false, "Write supplied RAM data to cartridge")
	autoname := flag.Bool("autoname", false, "Read ROM name and generate filenames to save ROM/RAM data")
	romfile := flag.String("romfile", "", "File to save or read ROM data")
	ramfile := flag.String("ramfile", "", "File to save or read RAM data")
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

	var d = &krikzz_fkmd.Fkmd{}
	d.SetOptions(options)
	var mdc memcart.MemCart
	mdc, err = d.MemCart()

	if err != nil {
		elog.Println("Error opening serial port: ", err)
		os.Exit(-1)
	} else {
		defer d.Disconnect()
	}

	if *readram {
		ReadRam(mdc, *ramfile, *autoname)
	}

	if *writeram {
		WriteRam(mdc, *ramfile)
	}

	if *readrom {
		ReadRom(mdc, *romfile, *autoname)
	}

	if *writerom {
		WriteRom(mdc, *romfile)
	}

	if *rominfo {
		gotromname, _ := mdcart.GetRomName(mdc)
		fmt.Println(gotromname)
	}
}
