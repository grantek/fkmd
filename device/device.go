package device

import (
	"io"
)

//type CartIODevice interface {
//    io.ReadWriteSeeker
//    io.Closer

//ReadWord(addr int64) (uint16, error)
//WriteByte(addr int, data byte) (err error)
//WriteWord(addr int64, data uint16) (err error)

//Connect() error
//Disconnect() error
//RamEnable() error
//RamDisable() error
//GetID() (int, error)
//}

type MemCart interface {
	NumBanks() (int, error)
	GetCurrentBank() (MemBank, error)
	SwitchBank(int) error
}

type MemBank interface {
	io.ReadWriteSeeker
	//GetName() (string, error)
}
