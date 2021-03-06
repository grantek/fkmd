package memcart_mock

import (
	"errors"
	"fmt"
	"github.com/grantek/fkmd/memcart"
	"io"
	//"io/ioutil"
	//"os"
)

type MockMemCart struct {
	banks       []*MockMemBank
	currentbank int
}

func (mc *MockMemCart) NumBanks() int {
	return len(mc.banks)
}

func (mc *MockMemCart) AddBank(mb *MockMemBank) {
	mc.banks = append(mc.banks, mb)
}

func (mc *MockMemCart) GetCurrentBank() memcart.MemBank {
	return mc.banks[mc.currentbank]
}

func (mc *MockMemCart) SwitchBank(n int) error {
	if n < 0 || n >= len(mc.banks) {
		return errors.New(fmt.Sprintf("Requested bank %d does not exist"))
	}
	mc.currentbank = n
	return nil
}

type MockMemBank struct {
	f    io.ReadWriteSeeker
	name string
	size int64
}

func (d *MockMemBank) Read(p []byte) (n int, err error) {
	return d.f.Read(p)
}

func (d *MockMemBank) Write(p []byte) (n int, err error) {
	return d.f.Write(p)
}

func (d *MockMemBank) Seek(offset int64, whence int) (int64, error) {
	return d.f.Seek(offset, whence)
}

func (d *MockMemBank) GetName() string {
	return d.name
}

func (d *MockMemBank) GetSize() int64 {
	return d.size
}

func NewMemBank(name string, f io.ReadWriteSeeker, size int64) (*MockMemBank, error) {
	var mb MockMemBank
	mb.f = f
	mb.name = name
	mb.size = size
	return &mb, nil
}
