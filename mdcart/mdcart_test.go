package mdcart

import (
	//"encoding/hex"
	//"errors"
	"flag"
	"fmt"
	"github.com/grantek/fkmd/memcart_mock"
	//"io"
	"os"
	"testing"
)

var mf *os.File
var mmc *memcart_mock.MockMemCart

func usage() {
	flag.PrintDefaults()
	os.Exit(-1)
}

func setup(m *testing.M) int {
	var (
		err  error
		fi   os.FileInfo
		f    *os.File
		mrom *memcart_mock.MockMemBank
	)
	testfile := flag.String("testfile", "", "expected ROM dump from device")
	flag.Parse()
	if *testfile == "" {
		usage()
	}
	fi, err = os.Stat(*testfile)
	if err != nil {
		panic(err)
	}
	if fi.Size() < 512 {
		panic(fmt.Sprintf("Small file for MockMemBank: minimum 512, file \"%s\" detected as %d bytes", *testfile, fi.Size()))
	}
	f, err = os.OpenFile(*testfile, os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	} else {
		defer f.Close()
	}
	mmc = new(memcart_mock.MockMemCart)
	mrom, err = memcart_mock.NewMemBank("mdrom", f, fi.Size())

	if err != nil {
		panic(err)
	}
	mmc.AddBank(mrom)

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(setup(m))
}

func TestGetRomName(t *testing.T) {
	var err error
	var romname string
	romname, err = GetRomName(mmc)

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(fmt.Sprintf("romname: %s", romname))
}
