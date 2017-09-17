package mdcart

import (
	"encoding/hex"
	//    "errors"
	"flag"
	"fmt"
	"github.com/jacobsa/go-serial/serial"
	"io"
	"os"
	"testing"
)

var d *Device
var mf *os.File
var d_len int64

func CompReadWord(t *testing.T, addr int64) {
	var (
		err         error
		device_word uint16
		file_word   uint16
	)
	t.Logf("ReadWord: %x", addr)
	device_word, err = d.ReadWord(addr)
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	file_word_bytes := make([]byte, 2)
	mf.Seek(addr, io.SeekStart)
	_, err = mf.Read(file_word_bytes)
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	file_word = uint16(file_word_bytes[0]) << 8
	file_word |= uint16(file_word_bytes[1])

	if file_word != device_word {
		t.Log("expected %x, device returned %x", file_word, device_word)
		t.Fail()
	}
}

func CompSeekRead(t *testing.T, addr int64, readlen int) {
	d.Seek(addr, io.SeekStart)
	mf.Seek(addr, io.SeekStart)
	buf := make([]byte, readlen)
	buf2 := make([]byte, readlen)

	n, err := d.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Read", n, "bytes from device")
	t.Logf("\n%s", hex.Dump(buf))

	n, err = mf.Read(buf2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Read", n, "bytes from file")
	t.Logf("\n%s", hex.Dump(buf2))
	for i, v := range buf {
		if buf2[i] != v {
			t.Logf("Fail at offset %x", i)
			t.Fail()
		}
	}
}

func usage() {
	flag.PrintDefaults()
	os.Exit(-1)
}

func setup(m *testing.M) int {
	port := flag.String("port", "/dev/ttyUSB0", "serial port to use (/dev/ttyUSB0, etc)")
	mockfile := flag.String("mockfile", "", "expected ROM dump from device")
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

	flag.Parse()

	if *port == "" {
		fmt.Println("Must specify port")
		usage()
	}

	if *mockfile == "" {
		fmt.Println("Must specify port")
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

	d = New()
	d.SetOptions(options)
	err := d.Connect()

	if err != nil {
		fmt.Println("Error opening serial port: ", err)
		return -1
	} else {
		defer d.Disconnect()
	}

	mf, err = os.Open(*mockfile)

	if err != nil {
		fmt.Println("Error opening mock file: ", err)
		return -1
	} else {
		defer mf.Close()
	}

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(setup(m))
}

func TestGetID(t *testing.T) {
	id, err := d.GetID()
	if err != nil {
		fmt.Println(err)
		t.Fatal(err)
	}
	t.Log("ID:", id)
	if id != 257 {
		t.Fatal("ID", id, "not a known device ID")
	}
}

func TestSeekReadLow(t *testing.T) {
	addrs := []int64{0, 0x200, 0x200, 0x100, 0xFE, 0x100}
	rlens := []int{0x80, 0x10, 0x2, 0x4, 0x100, 0x100}
	for i, v := range addrs {
		CompSeekRead(t, v, rlens[i])
	}
}

func TestReadWordLow(t *testing.T) {
	addrs := []int64{0x0, 0x4, 0x2, 0x0, 0x10, 0x100, 0x102, 0x104, 0x106, 0x108, 0x10a, 0x10c, 0x10e, 0xFFE, 0x1000}

	for _, addr := range addrs {
		CompReadWord(t, addr)
	}
}

func TestFullRead(t *testing.T) {
	var (
		chunksize int = 0x10000
		n, n2     int
		romlen    int
		err, err2 error
		i         int
		v         byte
	)
	buf := make([]byte, chunksize)
	buf2 := make([]byte, chunksize)

	mf.Seek(0, io.SeekStart)
	d.Seek(0, io.SeekStart)
	for {
		n, err = mf.Read(buf)

		if n <= 0 {
			break
		}
		n2, err2 = d.Read(buf2[:n])
		if n2 < n {
			if err2 != nil {
				t.Log(err2)
			}
			t.Fatal("Short read from addr", romlen, "on device: wanted", n, "got", n2)
		}
		for i, v = range buf {
			if buf2[i] != v {
				t.Fatal("Data mismatch at addr", romlen+i, ", read", buf2[i], "want", v)
			}
		}
		romlen += n

		if err == io.EOF {
			break
		}
	}
	if romlen == 0 {
		t.Fatal("Read 0 bytes from mock file")
	}
	t.Log("Read and matched", romlen, "bytes")

}
