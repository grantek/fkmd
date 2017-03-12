package krikzz_fkmd

import (
    "fmt"
    "io"
    "errors"
//    "github.com/grantek/fkmd/device"
	"github.com/jacobsa/go-serial/serial"
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

func (d *Fkmd) Connect() error {
	f, err := serial.Open(d.opt)
	d.fd = f
	if err != nil {
		return err
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
			return err
		}
		//need to redo GetID after reopen
		_, err = d.GetID()
		err = d.SetDelay(0)
	}
	return err
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

func (d *Fkmd) Write(p []byte) (offset int, err error) {
	var (
		n       int = 0      //total bytes written
		req_len int = len(p) //total bytes left
		wr_len  int          //current write chunk size
		wrote   int          //bytes written in current writeloop iteration
	)
	err = nil

	for req_len > 0 {
		wr_len = req_len
		if wr_len > 65536 {
			wr_len = 65536
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
		wrote, err = d.fd.Write(p[offset+n : offset+n+wr_len])
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
    if addr % WRITE_BLOCK_SIZE > 0 {
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

func (d *Fkmd)RamEnable() error {
    err := d.SetDelay(1)
    if err != nil {
        return err
    }
	err = d.WriteWord(0xA13000, 0xffff)
    return err
}

func (d *Fkmd)RamDisable() error {
    err := d.WriteWord(0xA13000, 0x0000)
    if err != nil {
        return err
    }
	err = d.SetDelay(0)
    return err
}

type Mdrom struct {
    d *Fkmd
    addr int64
}

func (m *Mdrom)Read(p []byte) (n int, err error) {
    n = 0
    err = m.d.RamDisable() //appropriate?
    if err != nil {
        return
    }
    return m.d.Read(p)
}

func (m *Mdrom) Seek(offset int64, whence int) (newoffset int64, err error) {
	switch whence {
	case io.SeekCurrent:
		offset += m.addr
    case io.SeekEnd:
		return -1, errors.New("SeekEnd not implemented")
	}

    newoffset, err = m.d.Seek(offset, whence)
    m.addr = newoffset
    return newoffset, err
}

func (m *Mdrom) Write(p []byte) (offset int, err error) {
    var (
        writelen int64
    )
    writelen = int64(len(p))
    if writelen % WRITE_BLOCK_SIZE > 0 {
        fmt.Println(fmt.Sprintf("Attempting to write less than %iKiB, flash erase will be larger than write", WRITE_BLOCK_SIZE/1024))
    }
    return 0, nil

}
