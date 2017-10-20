package mdcart

import (
	"errors"
	"fmt"
	"github.com/grantek/fkmd/krikzz_fkmd"
	"github.com/grantek/fkmd/memcart"
	"io"
)

const (
	MAP_2M  int = 1
	MAP_3M  int = 2
	MAP_SSF int = 3

	ROM_HDR_OFFSET int = 0
	ROM_HDR_LEN    int = 512
	NAME_LEN       int = 48

	MAX_ROM_SIZE int64 = 0x400000 //4MiB
)

func GetRomRegion(rom_hdr []byte) string {
	if len(rom_hdr) < 0x1f2 {
		return "X"
	}
	val := rom_hdr[0x1f0]
	if val != rom_hdr[0x1f1] && rom_hdr[0x1f1] != 0x20 && rom_hdr[0x1f1] != 0 {
		return "W"
	}

	switch val {

	case 'F':
		fallthrough
	case 'C':
		return "W"

	case 'U':
		fallthrough
	case 'W':
		fallthrough
	case '4':
		fallthrough
	case 4:
		return "U"

	case 'J':
		fallthrough
	case 'B':
		fallthrough
	case '1':
		fallthrough
	case 1:
		return "J"

	case 'E':
		fallthrough
	case 'A':
		fallthrough
	case '8':
		fallthrough
	case 8:
		return "E"

	}

	return "X"
}

func GetRomName(mdc memcart.MemCart) (string, error) {
	var (
		n   int
		err error
		mdr memcart.MemBank
	)
	err = mdc.SwitchBank(0)
	if err != nil {
		return "", err
	}

	mdr = mdc.GetCurrentBank()
	mdr.Seek(0, io.SeekStart)
	buf := make([]byte, 512)
	n, err = mdr.Read(buf)
	if n < 512 {
		return "", errors.New("short read")
	}
	if err != nil {
		return "", err
	}

	return GetRomNameFromHeader(buf)
}

func GetRomNameFromHeader(buf []byte) (string, error) {
	var (
		namestring string
		err        error
	)
	if len(buf) < 512 {
		return "", errors.New(fmt.Sprintf("Short ROM header, expected 512, got %d", len(buf)))
	}
	namestring, err = searchRomName(buf[0x120:])
	if err != nil {
		namestring, err = searchRomName(buf[0x150:])
	}

	if err != nil {
		namestring = "Unknown"
	}

	namestring = (fmt.Sprintf("%s (%s)", namestring, GetRomRegion(buf)))

	return namestring, nil
}

func searchRomName(rom_hdr []byte) (string, error) {
	// rom_hdr is expected to be pre-offset at a search position
	// finds name up to 48 bytes length
	var (
		found bool = false
		i     int
		name  []byte
		//j int
		//fname []byte //formatted name to dedupe spaces
		namestring string
	)

	name = make([]byte, NAME_LEN)
	//fname = make([]byte, NAME_LEN)
	copy(name, rom_hdr)

	for i = NAME_LEN - 1; i >= 0; i-- {
		if name[i] != 0 {
			if name[i] != 0x20 {
				break
			} else {
				name[i] = 0
			}
		}
	}

NameLoop:
	for i = 0; i < NAME_LEN; i++ {
		switch {
		case name[i] == 0:
			break NameLoop

		case name[i] == ' ':
			continue

		case name[i] == '/' ||
			name[i] == ':':
			name[i] = '-'
			found = true
			continue NameLoop

		case name[i] == '!' ||
			name[i] == '(' ||
			name[i] == ')' ||
			name[i] == '_' ||
			name[i] == '-' ||
			name[i] == '.' ||
			name[i] == '[' ||
			name[i] == ']' ||
			name[i] == '|' ||
			name[i] == '&' ||
			name[i] == 0x27 ||
			name[i] >= '0' && name[i] <= '9' ||
			name[i] >= 'A' && name[i] <= 'Z' ||
			name[i] >= 'a' && name[i] <= 'z':
			found = true
			continue NameLoop

		default:
			return "", errors.New("Illegal characters in ROM name")
		}
	}
	if found {
		namestring = fmt.Sprintf("%s", name[:i])
	}
	return namestring, nil
}
