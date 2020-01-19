// Game Boy cart flasher documented by jrodrigo.net/cart-flasher and www.reinerziegler.de/readplus.htm
// Original PC driver software from sourceforge.net/projects/gbcf
package gbcf

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	//"github.com/grantek/fkmd/gbcart"
	"github.com/grantek/fkmd/memcart"
	"github.com/howeyc/crc16"
	"github.com/jacobsa/go-serial/serial"
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
	//control byte,
	//command byte,
	//subcommand byte,
	//algo byte,
	//mbc byte,
	//frame byte[FRAMESIZE],
	//crc16 uint16,
	bytes [PACKETSIZE]byte
}

// setControl sets the control byte for the packet type.
// It relies on the stringer tool to generate the list of valid values.
func (p *Packet) setControl(cb ControlByte) error {
	// generated function returns "%T(%d)" for unknown values
	if strings.HasPrefix(cb.String(), "ControlByte") {
		return fmt.Errorf("Invalid control character for packet: %X", cb)
	}
	p.bytes[0] = byte(cb)
	return nil
}

// setCommand sets the command byte for packets of type DATA.
// It relies on the stringer tool to generate the list of valid values.
func (p *Packet) setCommand(cb CommandByte) error {
	// generated function returns "%T(%d)" for unknown values
	if strings.HasPrefix(cb.String(), "CommandByte") {
		return fmt.Errorf("Invalid command character for packet: %X", cb)
	}
	p.bytes[1] = byte(cb)
	return nil
}

// setSubcommand sets the subcommand byte for packets of type DATA.
func (p *Packet) setSubcommand(scb SubcommandByte) error {
	// Subcommand values have a rango of 0 to N-1 for commands with N subcommands.
	var n int
	cb := CommandByte(p.bytes[1])
	switch cb {
	case CONFIG:
		n = 4
	case NORMAL_DATA:
		n = 1
	case LAST_DATA:
		n = 1
	case ERASE:
		n = 2
	case STATUS:
		n = 2
	default:
		n = 1 // allow zeroing of packet
	}
	if int(scb) >= n {
		return fmt.Errorf("Subcommand value %d is invalid for %s.", scb, cb.String())
	}
	p.bytes[2] = byte(scb)
	return nil
}

/*
generate_crc16:
original source defines its own crc16 function.
- the predefined table is CRC16-CCITT-FALSE
- the function hashes the bytes of the packet and returns a short
*/
func (p *Packet) generate_crc16() error {
	// TODO: add validation and return error if invalid
	c := crc16.ChecksumCCITTFalse(p.bytes[:PACKETSIZE-2])
	p.bytes[PACKETSIZE-2] = byte(c / 256)
	p.bytes[PACKETSIZE-1] = byte(c % 256)
	return nil
}

func (p *Packet) check_packet() error {
	//TODO: move this to separate validation logic
	if ControlByte(p.bytes[0]) != DATA {
		return errors.New("Packet is not marked as DATA type.")
	}
	c := crc16.ChecksumCCITTFalse(p.bytes[:PACKETSIZE-2])
	if p.bytes[PACKETSIZE-2] != byte(c/256) ||
		p.bytes[PACKETSIZE-1] != byte(c%256) {
		return errors.New("CRC error in received packet.")
	}
	return nil
}

func (p *Packet) Serialise() ([]byte, error) {
	if err := p.generate_crc16(); err != nil {
		return nil, err
	}
	b := make([]byte, PACKETSIZE)
	if n := copy(b, p.bytes[:]); n != PACKETSIZE {
		return nil, fmt.Errorf("Serialise: copied %d of %d bytes", n, PACKETSIZE)
	}
	return b, nil
}

//go:generate stringer -type=ControlByte
// ControlByte uses generated stringer for value validation
type ControlByte byte

const (
	ACK  ControlByte = 0xAA
	NAK  ControlByte = 0xF0
	END  ControlByte = 0x0F
	DATA ControlByte = 0x55
)

const (
	SERIAL_TIMEOUT = 3 * time.Second
	DELETE_TIMEOUT = 60 * time.Second
	PACKETSIZE     = 72
	FRAMESIZE      = 64
	//AUTOSIZE       = -1 // unused
	//PORTS_COUNT    = 4 // not used in protocol
	//VER            = "1.1" // not used in protocol
)
const (
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
)

//go:generate stringer -type=CommandByte
// CommandByte uses generated stringer for value validation
type CommandByte byte

const (
	CONFIG      CommandByte = 0x00
	NORMAL_DATA CommandByte = 0x01
	LAST_DATA   CommandByte = 0x02
	ERASE       CommandByte = 0x03
	STATUS      CommandByte = 0x04
)

// SubcommandByte is manually validated by the setSubcommand function
type SubcommandByte byte

const (
	// filler subcommand used in *_DATA
	RESERVED SubcommandByte = 0x00

	// subcommands used in STATUS
	NREAD_ID SubcommandByte = 0x00 // read device information only
	READ_ID  SubcommandByte = 0x01 // read device+cartridge information

	// subcommands used in CONFIG
	RROM SubcommandByte = 0x00
	RRAM SubcommandByte = 0x01
	WROM SubcommandByte = 0x02
	WRAM SubcommandByte = 0x03

	// subcommands used in ERASE
	EFLA SubcommandByte = 0x00
	ERAM SubcommandByte = 0x01
)

const (
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
	b, err := p.Serialise()
	if err != nil {
		return err
	}
	n, err := d.fd.Write(p.bytes[:])
	if err != nil {
		return err
	}
	if n != len(b) {
		return fmt.Errorf("short write: sent %d of %d", n, len(b))
	}
	return nil
}

func (d *Gbcf) ReadDeviceStatus() error {
	p := &Packet{}
	if err := p.setControl(DATA); err != nil {
		return err
	}
	if err := p.setCommand(STATUS); err != nil {
		return err
	}
	if err := p.setSubcommand(NREAD_ID); err != nil {
		return err
	}
	if err := d.sendPacket(p); err != nil {
		return err
	}
	p, err := d.receive_packet()
	if err != nil {
		return err
	}
	if err := p.check_packet(); err != nil {
		return err
	}
	fmt.Println("Received packet:")
	fmt.Printf("%v\n", p.bytes)
	return nil
}

/*
// Receive a character when a control character is expected
func (d *Gbcf) receive_char() (byte, error) {
	b := make([]byte, 1)
	n, err := d.fd.Read(b)
	if err != nil {
		return 0, err
	}
	if n < len(b) {
		return 0, fmt.Errorf("short read: read %d of %d", n, len(b))
	}
	return b[0], nil
}
*/

// Receive a Packet
func (d *Gbcf) receive_packet() (*Packet, error) {
	p := &Packet{}
	n, err := d.fd.Read(p.bytes[:1])
	if err != nil {
		return nil, err
	}
	if n < 1 {
		return nil, errors.New("Failed to read first byte of packet.")
	}
	if ControlByte(p.bytes[0]) != DATA {
		return p, nil
	}
	n, err = d.fd.Read(p.bytes[1:])
	if err != nil {
		return nil, err
	}
	if n < PACKETSIZE-1 {
		return nil, fmt.Errorf("Short packet: read %d bytes.", n+1)
	}
	return p, nil
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
