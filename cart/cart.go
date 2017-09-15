package cart

//These functions currently work on the kfkmd directly, but should be made to work on a bank/memcart interface

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

func GetRomName(d *krikzz_fkmd.Fkmd) (string, error) {
	var (
		n          int
		err        error
		namestring string
	)

	d.Seek(0, io.SeekStart)
	buf := make([]byte, 512)
	n, err = d.Read(buf)
	if n < 512 {
		return "", errors.New("short read")
	}
	if err != nil {
		return "", err
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

func RamAvailable(d *krikzz_fkmd.Fkmd) bool {
	var (
		first_word uint16
		tmp        uint16
		err        error
	)

	d.RamEnable()
	first_word, err = d.ReadWord(0x200000)
	d.WriteWord(0x200000, uint16(first_word^0xffff))
	tmp, err = d.ReadWord(0x200000)
	if err != nil {
		panic(err)
	}
	d.WriteWord(0x200000, first_word)
	tmp ^= 0xffff
	if (first_word & 0x00ff) != (tmp & 0x00ff) { //Save RAM is 8-bit so we don't care what the second byte of the word is
		return false
	}

	return true
}

func GetRamSize(d *krikzz_fkmd.Fkmd) int {
	var (
		ram_size       int64
		first_word     uint16
		first_word_tmp uint16
		tmp            uint16
		tmp2           uint16
		ram_type       uint16 = 0x00ff
		err            error
	)

	//This commented-out write was in the original code
	//Device.writeWord(0xA13000, 0x0001); //RamDisable()

	if RamAvailable(d) == false { //RAM is banskswitched in here?
		return 0
	}

	first_word, err = d.ReadWord(0x200000)

	for ram_size = 256; ram_size < 0x100000; ram_size *= 2 {
		tmp, err = d.ReadWord(0x200000 + ram_size)
		d.WriteWord(0x200000+ram_size, tmp^0xffff)
		tmp2, err = d.ReadWord(0x200000 + ram_size)
		first_word_tmp, err = d.ReadWord(0x200000)
		if err != nil {
			panic(err)
		}
		d.WriteWord(0x200000+ram_size, tmp)
		tmp2 ^= 0xffff
		if (tmp & 0xff) != (tmp2 & 0xff) {
			break
		}
		if (first_word & ram_type) != (first_word_tmp & ram_type) {
			break
		}
	}

	//Save RAM is 8-bit on a 16-bit system, this returns the real size in bytes
	return int(ram_size / 2)

}

func checkRomSize(d *krikzz_fkmd.Fkmd, base_addr int, max_len int) int {
	var (
		eq          bool
		base_offset int = 0x8000
		offset      int = 0x8000
		i           int
		v           byte
		sector0     []byte
		sector      []byte
	)
	sector0 = make([]byte, 256)
	sector = make([]byte, 256)

	d.RamDisable()
	d.Seek(int64(base_addr), io.SeekStart)
	d.Read(sector0)

	for {
		d.Seek(int64(base_addr+offset), io.SeekStart)
		d.Read(sector)

		eq = true
		for i, v = range sector0 {
			if sector[i] != v {
				eq = false
			}
		}
		if eq == true {
			break
		}

		offset *= 2
		if offset >= max_len {
			break
		}
	}
	if offset == base_offset {
		return 0
	}
	return offset
}

// todo: seems to leave RAM in inconsistent bankswitch state
func GetRomSize(d *krikzz_fkmd.Fkmd) (romsize int) {
	var (
		v            byte
		i            int
		max_rom_size int
	)
	sector0 := make([]byte, 512)
	sector := make([]byte, 512)
	var ram bool = false
	var extra_rom bool = false

	//
	if RamAvailable(d) { //RAM enable
		ram = true
		extra_rom = true
		d.RamDisable()
		d.Seek(0x200000, io.SeekStart)
		d.Read(sector0)
		d.Seek(0x200000, io.SeekStart)
		d.Read(sector)
		for i, v = range sector0 {
			if sector[i] != v {
				extra_rom = false
			}
		}
		if extra_rom == true {
			extra_rom = false
			d.Seek(0x200000+0x10000, io.SeekStart)
			d.Read(sector) //wtf? logic from original driver
			d.RamEnable()
			d.Seek(0x200000, io.SeekStart)
			d.Read(sector)
			for i, v = range sector0 {
				if sector[i] != v {
					extra_rom = true
				}
			}
		}
	}

	if ram == true && extra_rom == false {
		max_rom_size = 0x200000
	} else {
		max_rom_size = 0x400000
	}

	//check ROM size based on unused address pin, causing ROM to appear to repeat
	//this detects ROM sizes as small as 64kiB (the 32kiB case looks intended for base_Addr > 0)
	romsize = checkRomSize(d, 0, max_rom_size)

	//search for "end of ROM" blank space
	//if the base address doesn't match the first checkRomSize iteration at 32k, then it will just keep going to max_len???
	//maybe the max_len args should be smaller
	if romsize == 0x400000 {
		romsize = 0x200000
		rs2 := checkRomSize(d, 0x200000, 0x200000)
		if rs2 == 0x200000 {
			rs2 = checkRomSize(d, 0x300000, 0x100000)
			if rs2 >= 0x80000 {
				rs2 = 0x200000
			} else {
				rs2 = 0x100000
			}
		}
		if rs2 >= 0x80000 {
			romsize += rs2
		}
	}

	return
}
