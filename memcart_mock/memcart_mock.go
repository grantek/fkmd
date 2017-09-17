package memcart_mock

import (
	"bytes"
	"github.com/grantek/fkmd/memcart"
	"os"
	"ioutil"
	"errors"
)

type MockMemCart struct {
	banks       []MockMemBank
	currentbank int
}

func (mc *MockMemCart) NumBanks() int {
	return len(mc.banks)
}

func (mc *MockMemCart) GetCurrentBank() MemBank {
	return mc.banks[currentbank]
}

func (mc *MockMemCart) SwitchBank(n int) error {
	if n < 0 || n >= len(mc.banks) {
		return errors.New(fmt.Sprintf("Requested bank %d does not exist"))
	}
	currentbank = n
	return nil
}

type MockMemBank struct {
	f      os.File
	string name
	int64  size
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

func NewMemBank(string name, io.Reader data)(memcart.MemBank, error){
	var (
		mb memcart.MockMemBank
		err error
		b []byte
	)

	err, b = ioutil.ReadAll(data)
	if err != nil {
		return nil, err
	}
	mb.f = bytes.NewBuffer(b)
}
