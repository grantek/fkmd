package memcart

import (
	"io"
)

type MemCart interface {
	NumBanks() int
	CurrentBank() MemBank
	SwitchBank(int) error
}

type MemBank interface {
	io.ReadWriteSeeker

	Name() string         // Implementation-specific name.
	Size() int64          // Size in bytes.
	AlwaysWritable() bool // RAM is always writable, ROM is sometimes writable.
}
