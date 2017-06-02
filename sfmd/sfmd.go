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

	"github.com/grantek/fkmd/cart"
	"github.com/grantek/fkmd/device"
	"github.com/grantek/fkmd/krikzz_fkmd"
	"github.com/jacobsa/go-serial/serial"
)

func usage() {
	fmt.Println("fkmd usage:")
	flag.PrintDefaults()
	os.Exit(-1)
}

//md specific
/*
func ReadRom(d *krikzz_fkmd.Fkmd, romfile string, autoname bool) {
	var (
		romname   string
		romsize   int
		blocksize int = 32768
		f         *os.File
		err       error
	)
	if autoname {
		romname, _ = cart.GetRomName(d)
		re := regexp.MustCompile("  *")
		romname = re.ReplaceAllString(romname, " ")
		romname = strings.Title(strings.ToLower(strings.TrimSpace(romname)))
		romfile = fmt.Sprintf("%s.bin", romname)
	}
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

	romsize = cart.GetRomSize(d)
	d.Seek(0, io.SeekStart)
	buf := make([]byte, blocksize)
	for i := 0; i < romsize; i += blocksize {
		d.Read(buf)
		f.Write(buf)
		if f != os.Stdout {
			fmt.Printf(".")
		}
	}
	fmt.Println()
}
*/

func ReadRom(d *krikzz_fkmd.Fkmd, romfile string, autoname bool) {
	if autoname {
		romfile = "autoname"
	}

}

func ReadRam(d *krikzz_fkmd.Fkmd, ramfile string, autoname bool) {
	var (
		romname string
		ramsize int
		f       *os.File
		err     error
		n       int
	)
	ramsize = cart.GetRamSize(d)
	if ramsize == 0 {
		fmt.Println("RAM not detected for reading")
		return
	}
	if autoname {
		romname, _ = cart.GetRomName(d)
		re := regexp.MustCompile("  *")
		romname = re.ReplaceAllString(romname, " ")
		romname = strings.Title(strings.ToLower(strings.TrimSpace(romname)))
		ramfile = fmt.Sprintf("%s.srm", romname)
	}
	if ramfile == "-" {
		f = os.Stdout
	} else {
		f, err = os.Create(ramfile)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if f != os.Stdout {
		fmt.Println("Opened", ramfile, "for writing")
		defer f.Close()
	}

	d.RamEnable()
	defer d.RamDisable()
	d.Seek(0x200000, io.SeekStart)
	buf := make([]byte, ramsize)

	n, err = d.Read(buf)
	if n < ramsize {
		panic(errors.New("short RAM read"))
	}
	if err != nil {
		panic(err)
	}
	f.Write(buf)
	if err != nil {
		panic(err)
	}
	if f != os.Stdout {
		fmt.Println("OK")
	}
}

func WriteRam(d *krikzz_fkmd.Fkmd, ramfile string) error {
	var (
		ramsize int
		f       *os.File
		i, j, n int
		err     error
		v       byte
	)

	ramsize = cart.GetRamSize(d)
	if ramsize == 0 {
		return errors.New("RAM not detected for writing")
	}
	if ramfile == "-" {
		f = os.Stdin
	} else {
		f, err = os.Open(ramfile)
		if err != nil {
			return err
		}
	}

	if f != os.Stdin {
		fmt.Println("Opened", ramfile, "for reading")
		defer f.Close()
	}

	buf := make([]byte, ramsize)
	nextbyte := make([]byte, 1)
	for i = 0; i < ramsize; {
		j, err = f.Read(buf)
		i += j
		if err != nil {
			return err
		}
		if j == 0 {
			return errors.New(fmt.Sprintf("error: read %d bytes from ramfile \"%s\", need %d", i, ramfile, ramsize))
		}
	}
	j, err = f.Read(nextbyte)
	if err == nil || j > 0 {
		return errors.New(fmt.Sprintf("error: read data beyond %d bytes from ramfile \"%s\": %x", ramsize, ramfile, nextbyte[0]))
	}

	d.RamEnable()
	defer d.RamDisable()
	d.Seek(0x200000, io.SeekStart)

	n, err = d.Write(buf)
	if err != nil {
		panic(err)
	}
	fmt.Println("Verify...")
	buf2 := make([]byte, ramsize)
	d.Seek(0x200000, io.SeekStart)
	n, err = d.Read(buf2)
	if n < ramsize {
		panic(errors.New("short RAM read"))
	}
	if err != nil {
		panic(err)
	}
	for i, v = range buf {
		if buf2[i] != v {
			return errors.New(fmt.Sprintf("Failed verification at byte %d", i))
		}
	}
	fmt.Println("ok")
	return nil
}

func WriteRom(d *krikzz_fkmd.Fkmd, romfile string) error {
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
		fmt.Printf("Warning: file size in bytes is odd\n", cart.MAX_ROM_SIZE, romsize)
		filebuf = append(filebuf, 0)
		romsize++
	}

	if romsize > cart.MAX_ROM_SIZE {
		fmt.Printf("Warning: Max ROM data size is %x cropping input\n", cart.MAX_ROM_SIZE, romsize)
		romsize = cart.MAX_ROM_SIZE
	}

	fblen = romsize
	if romsize < 0x8000 {
		return errors.New("Error: file size < 32KiB, pad with zeroes if required")
	}

	if romsize%0x10000 != 0 {
		romsize = (romsize/0x10000)*0x10000 + 0x10000
	}

	fmt.Println("Flash erase...")
	d.FlashResetBypass()

	for i = 0; i < romsize; i += 65536 {
		d.FlashErase(i)
		fmt.Printf("*")
	}
	fmt.Printf("\n")

	d.FlashUnlockBypass()
	d.Seek(0, io.SeekStart)
	fmt.Println("Flash write...")
	for i = 0; i < fblen; i += blocklen {
		if i+blocklen > fblen {
			blocklen = fblen - i
		}
		d.FlashWrite(filebuf[i : i+blocklen])
		fmt.Printf("*")
	}
	d.FlashResetBypass()
	fmt.Printf("\n")

	fmt.Println("Flash verify...")
	rom2 := make([]byte, romsize)

	d.Seek(0, io.SeekStart)
	for i = 0; i < fblen; i += blocklen {
		if i+blocklen > fblen {
			blocklen = fblen - i
		}
		d.Read(rom2[i : i+blocklen])
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

	/*
		if *even && *odd {
			fmt.Println("can't specify both even and odd parity")
			usage()
		}

		parity := serial.PARITY_NONE

		if *even {
			parity = serial.PARITY_EVEN
		} else if *odd {
			parity = serial.PARITY_ODD
		}
	*/

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

	var d = krikzz_fkmd.New()
	var mdc device.MemCart
	d.SetOptions(options)
	mdc, err = d.Connect()

	if err != nil {
		fmt.Println("Error opening serial port: ", err)
		os.Exit(-1)
	} else {
		defer d.Disconnect()
	}

	if *rominfo {
		fmt.Println("Not implemented")
		/*
			s, _ := cart.GetRomName(d)
			fmt.Println("ROM name:", s)
			fmt.Println("ROM size:", cart.GetRomSize(d))
			ramsize := cart.GetRamSize(d)
			if ramsize > 0 {
				fmt.Println("RAM available: yes")
				fmt.Println("RAM size:", ramsize)
			} else {
				fmt.Println("RAM available: no")
			}
		*/
	}

	var blocksize int64
	var f *os.File
	var n int

	if *readrom {
		if *romfile == "" {
			*romfile = "rom.out"
		}
		if *romfile == "-" {
			f = os.Stdout
		} else {
			f, err = os.Create(*romfile)
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		if f != os.Stdout {
			fmt.Println("Opened", romfile, "for writing")
			defer f.Close()
		}

		mdc.SwitchBank(0)
		var mdr device.MemBank
		mdr, err = mdc.GetCurrentBank()

		romsize := mdr.GetSize()
		buf := make([]byte, blocksize)
		for i := int64(0); i < romsize; i += blocksize {
			d.Read(buf)
			f.Write(buf)
			if f != os.Stdout {
				fmt.Printf(".")
			}
		}

		fmt.Println()

	}

	if *readram {
		if *romfile == "" {
			*romfile = "rom.out"
		}
		if *romfile == "-" {
			f = os.Stdout
		} else {
			f, err = os.Create(*romfile)
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		if f != os.Stdout {
			fmt.Println("Opened", romfile, "for writing")
			defer f.Close()
		}

		mdc.SwitchBank(1)
		mdr, err := mdc.GetCurrentBank()

		ramsize := mdr.GetSize()
		buf := make([]byte, ramsize)
		bytesread := 0
		for bytesread < int(ramsize) {
			n, err = mdr.Read(buf)
			bytesread += n
			if err != nil || bytesread == 0 {
				panic("short RAM read")
			}
		}

		fmt.Println()

	}

	if *writeram {
		fmt.Println("Not implemented")
	}

	if *writerom {
		fmt.Println("Not implemented")
	}

	var i int
	i, _ = mdc.NumBanks()
	fmt.Println(i)
}
