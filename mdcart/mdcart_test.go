package mdcart

import (
	//"encoding/hex"
	//"errors"
	"flag"
	"fmt"
	//"io"
	"os"
	"testing"
)

var mf *os.File

func usage() {
	flag.PrintDefaults()
	os.Exit(-1)
}

func setup(m *testing.M) int {
	var (
		err error
		fi  os.FileInfo
		f   *os.File
	)
	mockfile := flag.String("mockfile", "", "expected ROM dump from device")
	flag.Parse()
	fi, err = os.Stat(*mockfile)
	if err != nil {
		panic(err)
	}
	if fi.Size() < 512 {
		panic(fmt.Sprintf("Small file for MockMemBank: minimum 512, file \"%s\" detected as %d bytes", mockfile, fi.Size()))
	}
	f, err = os.OpenFile(*mockfile, os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}

	if err != nil {
		fmt.Println("Error opening mock file: ", err)
		return -1
	} else {
		defer f.Close()
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
