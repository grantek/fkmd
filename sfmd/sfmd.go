package main

import (
	//"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/grantek/fkmd/krikzz_fkmd"
	"github.com/grantek/fkmd/mdcart"
	"github.com/grantek/fkmd/memcart"
	"github.com/jacobsa/go-serial/serial"
)

func usage() {
	fmt.Println("fkmd usage:")
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
	mdr = mdc.GetCurrentBank()
	if romfile == "-" {
		f = os.Stdout
	} else {
		f, err = os.Create(romfile)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if f != os.Stdout {
		fmt.Println("Opened", romfile, "for writing")
		defer f.Close()
	}

	romsize = mdr.GetSize()
	mdr.Seek(0, io.SeekStart)
	buf := make([]byte, blocksize)
	for i := int64(0); i < romsize; i += blocksize {
		mdr.Read(buf)
		f.Write(buf)
		if f != os.Stdout {
			fmt.Printf(".")
		}
	}
	fmt.Println()
}

func ReadRam(mdc memcart.MemCart, ramfile string, autoname bool) {
	var (
		err error
		f   *os.File
		n   int
		mdr memcart.MemBank
	)

	err = mdc.SwitchBank(1)
	if err != nil {
		panic(err)
	}

	mdr = mdc.GetCurrentBank()
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
	}

	if f != os.Stdout {
		fmt.Println("Opened", ramfile, "for writing")
		defer f.Close()
	}

	ramsize := mdr.GetSize()
	buf := make([]byte, ramsize)
	bytesread := 0
	for bytesread < int(ramsize) {
		n, err = mdr.Read(buf[bytesread:])
		bytesread += n
		if err != nil || bytesread == 0 {
			panic("short RAM read")
		}
	}

	f.Write(buf)

	fmt.Println()
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
	mdr := mdc.GetCurrentBank()

	if ramfile == "-" {
		f = os.Stdin
	} else {
		f, err = os.Open(ramfile)
		if err != nil {
			panic(err)
		}
	}
	if f != os.Stdin {
		fmt.Println("Opened", ramfile, "for reading")
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
	fmt.Printf("Wrote %d bytes", n)
	if n < len(ram) {
		fmt.Printf("WARNING: wrote %d bytes, input is %d bytes.\n", n, len(ram))
	}
	if int64(n) < mdr.GetSize() {
		fmt.Printf("WARNING: wrote %d bytes, cartridge RAM is %d bytes.\n", n, len(ram))
	}
	fmt.Println("Verify...")
	mdr.Seek(0, io.SeekStart)
	for i := 0; i < n; i++ {
		if ram[i] != ram2[i] {
			panic(fmt.Sprintf("Failed verification at byte %d", i))
		}
	}
	fmt.Printf("Verified %d bytes", n)
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
	}

	filebuf, err = ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	if f != os.Stdin {
		f.Close()
	}

	romsize = int64(len(filebuf))

	fmt.Printf("Read %d bytes from %s\n", romsize, romfile)
	if romsize%2 == 1 {
		fmt.Printf("Warning: file size in bytes is odd\n", mdcart.MAX_ROM_SIZE, romsize)
		filebuf = append(filebuf, 0)
		romsize++
	}

	if romsize > mdcart.MAX_ROM_SIZE {
		fmt.Printf("Warning: Max ROM data size is %x, cropping input\n", mdcart.MAX_ROM_SIZE, romsize)
		romsize = mdcart.MAX_ROM_SIZE
	}

	fblen = romsize
	if romsize < 0x8000 {
		return errors.New("Error: file size < 32KiB, pad with 0xFF if required")
	}

	if romsize%0x10000 != 0 {
		romsize = (romsize/0x10000)*0x10000 + 0x10000
	}

	mdc.SwitchBank(0)
	mdr := mdc.GetCurrentBank()

	//Going to rely on Write() performing block erasure
	fmt.Println("Flash write...")
	for i = 0; i < fblen; i += blocklen {
		if i+blocklen > fblen {
			blocklen = fblen - i
		}
		mdr.Write(filebuf[i : i+blocklen])
		fmt.Printf("*")
	}
	fmt.Printf("\n")

	fmt.Println("Flash verify...")
	rom2 := make([]byte, romsize)

	mdr.Seek(0, io.SeekStart)
	for i = 0; i < fblen; i += blocklen {
		if i+blocklen > fblen {
			blocklen = fblen - i
		}
		mdr.Read(rom2[i : i+blocklen])
		fmt.Printf(".")
	}
	fmt.Printf("\n")

	for i = 0; i < fblen; i++ {
		if rom2[i] != filebuf[i] {
			return errors.New(fmt.Sprintf("Verify error at %x", i))
		}
	}

	fmt.Println("OK")
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

	flag.Parse()

	if *port == "" {
		fmt.Println("Must specify port")
		usage()
	}

	if *readram && *writeram {
		fmt.Println("Can't read and write cartridge RAM in one invocation")
		usage()
	}

	if *readrom && *writerom {
		fmt.Println("Can't read and write cartridge ROM in one invocation")
		usage()
	}

	if (*romfile != "" || *ramfile != "") && *autoname {
		fmt.Println("Can't supply file names when autoname is used")
		usage()
	}

	if (*readrom || *writerom) && (*romfile == "" && !*autoname) {
		fmt.Println("No ROM file name supplied")
		usage()
	}

	if (*readram || *writeram) && (*ramfile == "" && !*autoname) {
		fmt.Println("No RAM file name supplied")
		usage()
	}

	if !*readrom && !*writerom && !*readram && !*writeram && !*rominfo {
		fmt.Println("No action specified")
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
		fmt.Println("Error opening serial port: ", err)
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
