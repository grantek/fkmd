// Game Boy cart flasher documented by jrodrigo.net/cart-flasher and www.reinerziegler.de/readplus.htm
// Original PC driver software from sourceforge.net/projects/gbcf
package gbcf

import (
	"errors"
	"fmt"
	"time"
	//"github.com/grantek/fkmd/gbcart"
	"github.com/howeyc/crc16/crc16"
	"github.com/jacobsa/go-serial/serial"
	"io"
)

/* leftover from const.h
#define USB 0
#define SERIAL 1

// strings for version information
#ifdef Q_WS_X11
#define SYSTEM "LINUX"
#define DEVELOPED "GCC 4.1.1 + QT 4.3.2"
#endif

#ifdef Q_WS_WIN
#define SYSTEM "WINDOWS"
#define DEVELOPED "Dev-C++ 4.9.9.2 + QT 4.3.2"
#endif

*/

type Packet struct {
	//ptype byte,
	//command byte,
	//subcommand byte,
	//algo byte,
	//mbc byte,
	//frame byte[FRAMESIZE],
	//crc16 uint16,
	bytes [PACKETSIZE]byte
}

/*
generate_crc16:
original source defines its own crc16 function.
- the predefined table is CRC16-CCITT-FALSE
- the function hashes the bytes of the packet and returns a short
*/
func (p *Packet) generate_crc16() error {
	// TODO: return validate error
	c := crc16.ChecksumCCITTFalse(p.bytes[:PACKETSIZE-2])
	p.bytes[PACKETSIZE-2] = c / 256
	p.bytes[PACKETSIZE-1] = c % 256
	return nil
}

func (p *Packet) Serialise() ([]byte, error) {
	if err := p.generate_crc16(); err != nil {
		return nil, err
	}
	b := make([]byte, PACKETSIZE)
	if n := copy(b, p.bytes); n != PACKETSIZE {
		return nil, fmt.Errorf("Serialise: copied %d of %d bytes", n, PACKETSIZE)
	}
	return b, nil
}

const (
	// enum cchars
	ACK  byte = 0xAA
	NAK  byte = 0xF0
	END  byte = 0x0F
	DATA byte = 0x55

	SERIAL_TIMEOUT = 3 * time.Second
	DELETE_TIMEOUT = 60 * time.Second
	PACKETSIZE     = 72
	FRAMESIZE      = 64
	//AUTOSIZE       = -1 // unused
	//PORTS_COUNT    = 4 // not used in protocol
	//VER            = "1.1" // not used in protocol

	//enum error_t
	TIMEOUT     = -1
	FILEERROR_O = -2
	FILEERROR_W = -3
	FILEERROR_R = -4
	SEND_ERROR  = -5
	BAD_PACKET  = -6
	BAD_PARAMS  = -7
	PORT_ERROR  = -8
	WRONG_SIZE  = -9

	// packet types
	CONFIG      = 0x00
	NORMAL_DATA = 0x01
	LAST_DATA   = 0x02
	ERASE       = 0x03
	STATUS      = 0x04

	// filler subcommand used in *_DATA
	RESERVED = 0x00

	// subcommands used in STATUS
	NREAD_ID = 0x00 // read device information only
	READ_ID  = 0x01 // read device+cartridge information

	// subcommands used in CONFIG
	RROM = 0x00
	RRAM = 0x01
	WROM = 0x02
	WRAM = 0x03

	// subcommands used in ERASE
	EFLA = 0x00
	ERAM = 0x01

	// enum alg_t
	ALG16 = 0x00
	ALG12 = 0x01

	// enum dap_t
	LONGER   = 0x00
	DEFAULT  = 0x01
	DATAPOLL = 0x02
	TOGGLE   = 0x03

	// speed_type
	LOW      = 0x00
	STANDARD = 0x01
	HIGH     = 0x02

	// enum mbc_t
	MBCAUTO = 0x00
	MBC1    = 0x01
	MBC2    = 0x02
	MBC3    = 0x03
	ROMONLY = 0x04
	MBC5    = 0x05
	RUMBLE  = 0x06
)

type Gbcf struct {
	fd  io.ReadWriteCloser
	opt serial.OpenOptions
}

func (d *Gbcf) SetOptions(options serial.OpenOptions) {
	d.opt = options
}

//Perform initialisation and return a MemCart
func (d *Gbcf) MemCart() (memcart.MemCart, error) {
	//var gbc gbcart
	if err := d.Connect(); err != nil {
		return nil, err
	}
	/*if err := d.Handshake(); err != nil {
	return nil, err

	*/

	return nil, nil //d.Gbcart()
}

//Just open the serial device for low-level debugging
func (d *Gbcf) Connect() error {
	f, err := serial.Open(d.opt)
	d.fd = f
	return err
}

// Note: original driver always sends fixed PACKETSIZE packets
func (d *Gbcf) sendPacket(p *Packet) error {
	b := p.Serialise()
	n, err := d.fd.Write(cmd)
	if err {
		return err
	}
	if n != len(b) {
		return fmt.Errorf("short write: sent %d of %d", n, len(b))
	}
	return nil
}

func (d *Gbcf) ReadDeviceStatus() {
	p := Packet{}
	p.SetType(DATA)
	p.SetCommand(STATUS)
	p.SetSubcommand(NREAD_ID)
	d.sendPacket(p)
}

// Receive a character when a control character is expected
func (d *Gbcf) receive_char() (byte, error) {
	var b [1]byte
	n, err := d.fd.Read(b)
	if err != nil {
		return 0, err
	}
	if n < len(b) {
		return 0, fmt.Errorf("short read: sent %d of %d", n, len(b))
	}
	return b[0], nil
}

/* from fkmd:
//Perform some initialisation and verify device ID, as per vendor driver
//Requires Connect()
func (d *Gbcf) Handshake() error {
	//my flashkit device ID is 257, which matches this logic
	id, err := d.GetID()
	if err != nil {
		return err
	}

	if (id&0xff) == (id>>8) && id != 0 {
		//vendor driver doesn't close and reopen
		d.fd.Close()
		//port.WriteTimeout = 2000;
		//port.ReadTimeout = 2000;
		d.opt.InterCharacterTimeout = 2000
		f, err := serial.Open(d.opt)
		if err != nil {
			return err
		}
		d.fd = f
		//need to redo GetID after reopen
		_, err = d.GetID()
		if err != nil {
			return err
		}
		err = d.SetDelay(0)
		return err
	}
	return errors.New(fmt.Sprintf("Unknown device ID %d", id))
}
*/
