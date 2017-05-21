package krikzz_fkmd

import (
	"errors"
	"fmt"
	"github.com/grantek/fkmd/device"
	"github.com/jacobsa/go-serial/serial"
	"io"
)

const (
	CMD_ADDR   byte = 0
	CMD_LEN    byte = 1
	CMD_RD     byte = 2
	CMD_WR     byte = 3
	CMD_RY     byte = 4
	CMD_DELAY  byte = 5
	PAR_MODE8  byte = 16
	PAR_DEV_ID byte = 32
	PAR_SINGE  byte = 64
	PAR_INC    byte = 128

	WRITE_BLOCK_SIZE int64 = 65536

	RAM_ADDR int64 = 0x20000
)

type Fkmd struct {
	fd  io.ReadWriteCloser
	opt serial.OpenOptions
}

func New() *Fkmd {
	var d Fkmd
	return &d
}

func (d *Fkmd) SetOptions(options serial.OpenOptions) {
	d.opt = options
}

func (d *Fkmd) Connect() (device.MemCart, error) {
	f, err := serial.Open(d.opt)
	d.fd = f
	if err != nil {
		return nil, err
	}

	//my flashkit device ID is 257, which matches this logic
	id, err := d.GetID()
	if (id&0xff) == (id>>8) && id != 0 {
		//original code doesn't close and reopen
		d.fd.Close()
		//port.WriteTimeout = 2000;
		//port.ReadTimeout = 2000;
		d.opt.InterCharacterTimeout = 2000
		f, err := serial.Open(d.opt)
		d.fd = f
		if err != nil {
			return nil, err
		}
		//need to redo GetID after reopen
		_, err = d.GetID()
		err = d.SetDelay(0)
	}
	var mdc MDCart
	mdc.d = d
	return &mdc, err
}

func (d *Fkmd) Disconnect() error {
	e := d.fd.Close()
	if e != nil {
		d.fd = nil
	}
	return e
}

func (d *Fkmd) GetID() (int, error) {
	var id int = 0
	cmd := make([]byte, 1)
	data := make([]byte, 2)

	cmd[0] = CMD_RD | PAR_SINGE | PAR_DEV_ID
	n, err := d.fd.Write(cmd)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, errors.New(fmt.Sprintf("error: short write: %d of %d bytes", n, 1))
	}

	n, err = d.fd.Read(data)
	if err != nil {
		return 0, err
	}
	if n < 2 {
		return 0, errors.New(fmt.Sprintf("error: short read: %d of %d bytes", n, 2))
	}

	id = int(data[0]) << 8
	id |= int(data[1])

	return id, nil
}

func (d *Fkmd) SetDelay(val byte) error {
	cmd := make([]byte, 2)
	cmd[0] = CMD_DELAY
	cmd[1] = val

	_, err := d.fd.Write(cmd)

	return err
}

func (d *Fkmd) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		return 0, errors.New("SeekCurrent not implemented")
	case io.SeekEnd: //TODO: figure out what to return
		return -1, errors.New("SeekEnd not implemented")
	}

	buf := make([]byte, 6)
	addr := offset / 2

	buf[0] = CMD_ADDR
	buf[1] = byte(addr >> 16)
	buf[2] = CMD_ADDR
	buf[3] = byte(addr >> 8)
	buf[4] = CMD_ADDR
	buf[5] = (byte)(addr)

	d.fd.Write(buf)

	return offset, nil
}

//odd offset OR len(p) will break these functions
func (d *Fkmd) Read(p []byte) (n int, err error) {
	var (
		req_len int = len(p) //total bytes left
		rd_len  int          //current read chunk size
		i       int          //chunk loop's readloop offset
		read    int          //bytes read in current readloop iteration
	)
	n = 0 //total bytes read
	err = nil
	if req_len%2 == 1 {
		return 0, errors.New("Fkmd.Read: odd-sized read not implemented")
	}
	/*
	   mod := 0
	   if req_len % 2 == 1 {
	       mod = 1
	   }
	*/

	for req_len > 0 {
		rd_len = req_len
		if rd_len > 65536 {
			rd_len = 65536
		}
		cmd := make([]byte, 5)
		cmd[0] = CMD_LEN
		cmd[1] = byte(rd_len / 2 >> 8)
		cmd[2] = CMD_LEN
		cmd[3] = byte(rd_len / 2)
		cmd[4] = CMD_RD | PAR_INC

		_, err = d.fd.Write(cmd)

		for i = 0; i < rd_len; { //loop to protect against short reads
			read, err = d.fd.Read(p[n : n+rd_len-i])
			i += read
			n += read
			if err != nil {
				return n, err
			}
			if read == 0 {
				return n, errors.New("zero-byte read")
			}
		}
		req_len -= rd_len
	}

	/*
	   if mod == 1 {
	       high := make([]byte, 1)
	       read, err = d.fd.Read(p[n:n+rd_len-i])
	       d.fd.Read(high)
	   }
	*/
	return n, err
}

//the input address to this is a byte offset, the output is a word, so increment addr by 2 to read sequentially
//device's internal seek position remains at start of word
func (d *Fkmd) ReadWord(addr int64) (uint16, error) {
	var val uint16 = 0
	var err error

	addr /= 2
	cmd := make([]byte, 7)
	buf := make([]byte, 2)

	cmd[0] = CMD_ADDR
	cmd[1] = byte(addr >> 16)
	cmd[2] = CMD_ADDR
	cmd[3] = byte(addr >> 8)
	cmd[4] = CMD_ADDR
	cmd[5] = byte(addr)

	cmd[6] = CMD_RD | PAR_SINGE

	_, err = d.fd.Write(cmd)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}

	_, err = d.fd.Read(buf)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}

	val = uint16(buf[0]) << 8
	val |= uint16(buf[1])

	return val, nil

}

// we can only write to the lower byte of a word-aligned address
// since cartridge RAM is 8-bit
func (d *Fkmd) WriteByte(addr int, data byte) (err error) {
	cmd := make([]byte, 8)
	addr /= 2

	cmd[0] = CMD_ADDR
	cmd[1] = byte(addr >> 16)
	cmd[2] = CMD_ADDR
	cmd[3] = byte(addr >> 8)
	cmd[4] = CMD_ADDR
	cmd[5] = byte(addr)

	cmd[6] = CMD_WR | PAR_SINGE | PAR_MODE8
	cmd[7] = data

	d.fd.Write(cmd)
	return
}

func (d *Fkmd) WriteWord(addr int64, data uint16) (err error) {
	cmd := make([]byte, 9)
	addr /= 2

	cmd[0] = CMD_ADDR
	cmd[1] = byte(addr >> 16)
	cmd[2] = CMD_ADDR
	cmd[3] = byte(addr >> 8)
	cmd[4] = CMD_ADDR
	cmd[5] = byte(addr)

	cmd[6] = CMD_WR | PAR_SINGE
	cmd[7] = byte(data >> 8)
	cmd[8] = byte(data)

	d.fd.Write(cmd)
	return
}

func (d *Fkmd) Write(p []byte) (n int, err error) {
	var (
		req_len int = len(p) //total bytes left
		wr_len  int          //current write chunk size
		wrote   int          //bytes written in current writeloop iteration
	)
	err = nil

	for req_len > 0 {
		wr_len = req_len
		if wr_len > WRITE_BLOCK_SIZE {
			wr_len = WRITE_BLOCK_SIZE
		}
		cmd := make([]byte, 5)
		cmd[0] = CMD_LEN
		cmd[1] = byte(wr_len / 2 >> 8)
		cmd[2] = CMD_LEN
		cmd[3] = byte(wr_len / 2)
		cmd[4] = CMD_WR | PAR_INC

		_, err = d.fd.Write(cmd)
		if err != nil {
			return 0, err
		}
		wrote, err = d.fd.Write(p[n : n+wr_len])
		n += wrote
		if err != nil {
			return n, err
		}
		req_len -= wr_len
	}
	return n, err
}

// Erase 64KiB from addr (32K words)
func (d *Fkmd) FlashErase(addr int64) error {
	if addr%WRITE_BLOCK_SIZE > 0 {
		return errors.New(fmt.Sprintf("Flash erase request is not aligned to a %iKiB block", WRITE_BLOCK_SIZE/1024))
	}
	cmd := make([]byte, 8*8)
	addr /= 2

	for i := 0; i < len(cmd); i += 8 {
		cmd[0+i] = CMD_ADDR
		cmd[1+i] = byte(addr >> 16)
		cmd[2+i] = CMD_ADDR
		cmd[3+i] = byte(addr >> 8)
		cmd[4+i] = CMD_ADDR
		cmd[5+i] = byte(addr)

		cmd[6+i] = CMD_WR | PAR_SINGE | PAR_MODE8
		cmd[7+i] = 0x30
		addr += 4096 //jump 8kib, for a total of 64kib covered
	}

	d.WriteWord(0x555*2, 0xaa) //write 0x00aa to 0xaaa
	d.WriteWord(0x2aa*2, 0x55) //write 0x0055 to 0x554
	d.WriteWord(0x555*2, 0x80) //write 0x0080 to 0xaaa
	d.WriteWord(0x555*2, 0xaa) //write 0x00aa to 0xaaa
	d.WriteWord(0x2aa*2, 0x55) //write 0x0055 to 0x554
	//writeWord(addr, 0x30); //comment from original code

	_, err := d.fd.Write(cmd)
	d.FlashRY()

	return err
}

func (d *Fkmd) FlashRY() error {
	cmd := make([]byte, 2)
	buf := make([]byte, 2)
	cmd[0] = CMD_RY
	cmd[1] = CMD_RD | PAR_SINGE

	d.fd.Write(cmd)
	d.fd.Read(buf)

	return nil
}

func (d *Fkmd) FlashUnlockBypass() {
	d.WriteByte(0x555*2, 0xaa)
	d.WriteByte(0x2aa*2, 0x55)
	d.WriteByte(0x555*2, 0x20)
}

func (d *Fkmd) FlashResetBypass() {
	d.WriteWord(0, 0xf0)
	d.WriteByte(0, 0x90)
	d.WriteByte(0, 0x00)
}

// expected use is to perform full erase, seek to 0, then run this with chunks of data until complete
func (d *Fkmd) FlashWrite(buf []byte) error {
	if len(buf)%2 == 1 {
		return errors.New("odd write lengths not supported")
	}
	buf_len := len(buf) / 2
	cmd := make([]byte, buf_len*6)
	offset := 0

	for i := 0; i < (buf_len * 6); i += 6 {
		cmd[0+i] = CMD_WR | PAR_SINGE | PAR_MODE8
		cmd[1+i] = 0xA0
		cmd[2+i] = CMD_WR | PAR_SINGE | PAR_INC
		cmd[3+i] = buf[offset]
		cmd[4+i] = buf[offset+1]
		cmd[5+i] = CMD_RY

		offset += 2
	}

	_, err := d.fd.Write(cmd)

	return err
}

func (d *Fkmd) RamEnable() error {
	err := d.SetDelay(1)
	if err != nil {
		return err
	}
	err = d.WriteWord(0xA13000, 0xffff)
	return err
}

func (d *Fkmd) RamDisable() error {
	err := d.WriteWord(0xA13000, 0x0000)
	if err != nil {
		return err
	}
	err = d.SetDelay(0)
	return err
}

////////////////MDRom (MemBank)

type MDRom struct {
	d          *Fkmd
	addressCur int64
	size       int64
}

func (m *MDRom) Read(p []byte) (n int, err error) {
	/*
	   n = 0
	   err = m.d.RamDisable() //appropriate?
	   if err != nil {
	       return
	   }
	*/
	n, err = m.d.Read(p)
	m.addressCur += int64(n)
	return
}

func (m *MDRom) Seek(offset int64, whence int) (newoffset int64, err error) {
	var (
		romsize int64
	)
	switch whence {
	case io.SeekCurrent:
		offset += m.addressCur
	case io.SeekEnd:
		romsize = m.d.GetRomSize()
		if offset > romsize {
			return romsize - m.addressCur - 1, errors.New(fmt.Sprintf("Trying to seek %i from io.SeekEnd of ROM with detected length %i", offset, romsize))
		}
		offset = m.d.GetRomSize() - offset
	}

	newoffset, err = m.d.Seek(offset, io.SeekStart)
	m.addressCur = newoffset
	return
}

//should allow incremental writing, erasing pages as they're reached?
//if not on a page boundary, assume the rest of the page has been erased
func (m *MDRom) Write(p []byte) (n int, err error) {
	var (
		writelen  int
		chunksize int
	)
	writelen = len(p)
	for n <= writelen {
		chunksize = writelen - n
		if chunksize > WRITE_BLOCK_SIZE {
			chunksize = WRITE_BLOCK_SIZE
		}
		if m.addressCur%WRITE_BLOCK_SIZE == 0 {
			md.d.FlashErase(m.addressCur)
			md.d.Seek(m.addressCur)
		}
		chunksize = chunksize - m.addressCur%WRITE_BLOCK_SIZE
		err = m.d.FlashWrite(p[n : n+chunksize])

		if err != nil {
			n += chunksize
			m.addressCur += int64(n)
		} else {
			panic()
		}

	}

	return
}

func (m *MDRom) GetName() (string, error) {
	return "mdrom", nil
}

///////////////mdram (MemBank)
type MDRam struct {
	d          *Fkmd
	addressCur int64
}

func (m *MDRam) Read(p []byte) (n int, err error) {
	/*
	   n = 0
	   if m.addressCur == 0 {
	       err = m.d.RamEnable()
	   }
	   if err != nil {
	       return
	   }
	*/
	n, err = m.d.Read(p)
	m.addressCur += int64(n)
	return
}

func (m *MDRam) Write(p []byte) (n int, err error) {
	n, err = m.d.Write(p)
	m.addressCur += int64(n)
	return
}

func (m *MDRam) Seek(offset int64, whence int) (newoffset int64, err error) {
	var (
		ramsize int64
	)
	switch whence {
	case io.SeekCurrent:
		offset += m.addressCur
	case io.SeekEnd:
		ramsize = m.d.GetRamSize()
		if offset > ramsize {
			return ramsize - m.addressCur - 1, errors.New(fmt.Sprintf("Trying to seek %i from io.SeekEnd of RAM with detected length %i", offset, ramsize))
		}
		offset = m.d.GetRamSize() - offset
	}

	newoffset, err = m.d.Seek(offset+RAM_ADDR, io.SeekStart)
	m.addressCur = newoffset
	return
}

func (m *MDRam) GetName() (string, error) {
	return "mdram", nil
}

///////////////mdcart (MemCart)
type MDCart struct {
	d            *Fkmd
	ramAvailable bool
	currentBank  device.MemBank
}

func (mdc *MDCart) NumBanks() (int, error) {
	if mdc.ramAvailable {
		return 2, nil
	}
	return 1, nil
}

func (mdc *MDCart) GetCurrentBank() (device.MemBank, error) {
	return mdc.currentBank, nil
}

//always explicitly seeks to 0
func (mdc *MDCart) SwitchBank(reqbank int) error {
	if reqbank == 0 {
		var m MDRom
		m.d = mdc.d
		addr, err := m.d.Seek(0, io.SeekStart)
		m.addressCur = addr
		mdc.currentBank = &m
		return err
	}
	if reqbank == 1 {
		if !mdc.ramAvailable {
			return errors.New("req ram, no ram")
		}
		var m MDRam
		m.d = mdc.d
		addr, err := m.d.Seek(0, io.SeekStart)
		m.addressCur = addr
		mdc.currentBank = &m
		return err
		//return errors.New("ram not implemented in MDCart")
	}
	return nil
}

///////////////from cart.go

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

func GetRomName(d *Fkmd) (string, error) {
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

func (d *Fkmd) RamAvailable() bool {
	var (
		first_word uint16
		tmp        uint16
		err        error
	)

	d.RamEnable()
	first_word, err = d.ReadWord(RAM_ADDR)
	d.WriteWord(RAM_ADDR, uint16(first_word^0xffff))
	tmp, err = d.ReadWord(RAM_ADDR)
	if err != nil {
		panic(err)
	}
	d.WriteWord(RAM_ADDR, first_word)
	tmp ^= 0xffff
	if (first_word & 0x00ff) != (tmp & 0x00ff) { //Save RAM is 8-bit so we don't care what the second byte of the word is
		return false
	}

	return true
}

func (d *Fkmd) GetRamSize() int64 {
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

	if d.RamAvailable() == false { //RAM is banskswitched in here?
		return 0
	}

	first_word, err = d.ReadWord(RAM_ADDR)

	for ram_size = 256; ram_size < 0x100000; ram_size *= 2 {
		tmp, err = d.ReadWord(RAM_ADDR + ram_size)
		d.WriteWord(RAM_ADDR+ram_size, tmp^0xffff)
		tmp2, err = d.ReadWord(RAM_ADDR + ram_size)
		first_word_tmp, err = d.ReadWord(RAM_ADDR)
		if err != nil {
			panic(err)
		}
		d.WriteWord(RAM_ADDR+ram_size, tmp)
		tmp2 ^= 0xffff
		if (tmp & 0xff) != (tmp2 & 0xff) {
			break
		}
		if (first_word & ram_type) != (first_word_tmp & ram_type) {
			break
		}
	}

	//Save RAM is 8-bit on a 16-bit system, this returns the real size in bytes
	return ram_size / 2

}

func checkRomSize(d *Fkmd, base_addr int64, max_len int64) int64 {
	var (
		eq          bool
		base_offset int64 = 0x8000
		offset      int64 = 0x8000
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
		d.Seek(base_addr+offset, io.SeekStart)
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

// Should be called with RAM disabled, will return with RAM disabled and address
// cursor inconsistent
func (d *Fkmd) GetRomSize() (romsize int64) {
	var (
		v            byte
		i            int
		max_rom_size int64
	)
	sector0 := make([]byte, 512)
	sector := make([]byte, 512)
	var ram bool = false
	var extra_rom bool = false

	defer d.RamDisable()
	if d.RamAvailable() { //RAM enable
		ram = true
		extra_rom = true
		d.RamDisable()
		d.Seek(RAM_ADDR, io.SeekStart)
		d.Read(sector0)
		d.Seek(RAM_ADDR, io.SeekStart)
		d.Read(sector)
		for i, v = range sector0 {
			if sector[i] != v {
				extra_rom = false
			}
		}
		if extra_rom == true {
			extra_rom = false
			d.Seek(RAM_ADDR+0x10000, io.SeekStart)
			d.Read(sector) //wtf? logic from original driver
			d.RamEnable()
			d.Seek(RAM_ADDR, io.SeekStart)
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
