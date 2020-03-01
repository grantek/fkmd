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
// FirmwareVersion represents the device version sent by STATUS(NREAD_ID)
type FirmwareVersion struct {
	/* digits describing version of device soft */
	Ver11 byte
	Ver12 byte
	Ver21 byte
	Ver22 byte
}

// DeviceCartInfo stores the cart-related parts of STATUS(READ_ID)
type DeviceCartInfo struct {
	/* flash chip data */
	ManufacturerID byte
	ChipID         byte
	BBL            byte // Boot Block Locked
	/* info about loaded game */
	LogoCorrect byte
	CGB         byte
	SGB         byte
	ROMSize     byte //[6]byte
	RAMSize     byte //[6]byte
	CRC16       uint16
	TypeID      byte   //typ          [30]byte
	GameName    string //[17]byte
}

// GBCartInfo is human-readable version of DeviceCartInfo
type GBCartInfo struct {
	Manufacturer string
	ChipID       byte
	BBL          bool
	LogoCorrect  bool
	CGB          bool
	SGB          bool
	ROMSize      string
	RAMSize      string
	CRC16        uint16
	CartType     string
	GameName     string
}

/* array of cart types - source GB CPU Manual */
var carts = map[byte]string{
	0x00: "ROM ONLY",
	0x01: "ROM+MBC1",
	0x02: "ROM+MBC1+RAM",
	0x03: "ROM+MBC1+RAM+BATT",
	0x05: "ROM+MBC2",
	0x06: "ROM+MBC2+BATTERY",
	0x08: "ROM+RAM",
	0x09: "ROM+RAM+BATTERY",
	0x11: "ROM+MBC3",
	0x0b: "ROM+MMMO1",
	0x0c: "ROM+MMMO1+SRAM",
	0x0d: "ROM+MMMO1+SRAM+BATT",
	0x0f: "ROM+MBC3+TIMER+BATT",
	0x10: "ROM+MBC3+TIMER+RAM+BAT",
	0x12: "ROM+MBC3+RAM",
	0x13: "ROM+MBC3+RAM+BATT",
	0x19: "ROM+MBC5",
	0x1a: "ROM+MBC5+RAM",
	0x1b: "ROM+MBC5+RAM+BATT",
	0x1c: "ROM+MBC5+RUMBLE",
	0x1d: "ROM+MBC5+RUMBLE+SRAM",
	0x1e: "ROM+MBC5+RUMBLE+SRAM+BATT",
	0x1f: "Pocket Camera",
	0xfd: "Bandai TAMA5",
	0xfe: "Hudson HuC-3",
}

var romSizes = map[byte]string{
	0x00: "32KB",
	0x01: "64KB",
	0x02: "128KB",
	0x03: "256KB",
	0x04: "512KB",
	0x05: "1MB",
	0x06: "2MB",
	0x52: "1.1MB",
	0x53: "1.2MB",
	0x54: "1.5MB",
}

var ramSizes = map[byte]string{
	0x00: "0KB",
	0x01: "2KB",
	0x02: "8KB",
	0x03: "32KB",
	0x04: "128KB",
}

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

// CRC16 returns the CRC16 of a packet.
// Original source defines its own crc16 function.
// - the predefined table matches CRC16-CCITT-FALSE
// - the initial CRC is 0x0000, as in CRC16-CCITT
// - the function hashes the bytes of the packet and returns a short
func (p *Packet) CRC16() uint16 {
	var c uint16
	for _, v := range p.bytes[:PACKETSIZE-2] {
		c = (c << 8) ^ crc16Table[byte(c>>8)^v]
	}
	return c
}

// Check performs some validation on a received packet.
func (p *Packet) Check() error {
	//TODO: move this to separate validation logic
	if ControlByte(p.bytes[0]) != DATA {
		return errors.New("Packet is not marked as DATA type.")
	}
	c := p.CRC16()
	if p.bytes[PACKETSIZE-2] != byte(c/256) ||
		p.bytes[PACKETSIZE-1] != byte(c%256) {
		return errors.New("CRC error in received packet.")
	}
	return nil
}

func (p *Packet) Serialise() ([]byte, error) {
	c := p.CRC16()
	p.bytes[PACKETSIZE-2] = byte(c / 256)
	p.bytes[PACKETSIZE-1] = byte(c % 256)
	b := make([]byte, PACKETSIZE)
	if n := copy(b, p.bytes[:]); n != PACKETSIZE {
		return nil, fmt.Errorf("Serialise: copied %d of %d bytes", n, PACKETSIZE)
	}
	return b, nil
}

// DeviceStatusShort parses a STATUS(NREAD_ID) packet into a *FirmwareVersion.
func (p *Packet) DeviceStatusShort() *FirmwareVersion {
	fw := &FirmwareVersion{}
	fw.Ver11 = p.bytes[2] / 16
	fw.Ver12 = p.bytes[2] % 16
	fw.Ver21 = p.bytes[3] / 16
	fw.Ver22 = p.bytes[3] % 16
	return fw
}

func (p *Packet) DeviceStatusLong() (*FirmwareVersion, *DeviceCartInfo) {
	fw := p.DeviceStatusShort()
	dci := &DeviceCartInfo{}
	dci.ManufacturerID = p.bytes[4]
	dci.ChipID = p.bytes[5]
	dci.BBL = p.bytes[6] & 0x01
	dci.LogoCorrect = p.bytes[8]
	dci.GameName = string(p.bytes[9:25])
	if p.bytes[24] == 0x80 {
		dci.CGB = 1
	}
	if p.bytes[27] == 0x03 {
		dci.SGB = 1
	}
	dci.TypeID = p.bytes[28]
	dci.ROMSize = p.bytes[29]
	dci.RAMSize = p.bytes[30]
	dci.CRC16 = 256*uint16(p.bytes[35]) + uint16(p.bytes[36])
	return fw, dci
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
	//enum error_t (signed char)
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

	// enum dap_t, unreferenced in original code.
	// LONGER   = 0x00
	// DEFAULT  = 0x01
	// DATAPOLL = 0x02
	// TOGGLE   = 0x03 // Default

	// speed_type, appears to be local to original code.
	// LOW      = 0x00 // 125000 baud
	// STANDARD = 0x01 // 185000 baud, default
	// HIGH     = 0x02 // 375000 baud

	// enum mbc_t
	MBCAUTO = 0x00
	MBC1    = 0x01
	MBC2    = 0x02
	MBC3    = 0x03
	ROMONLY = 0x04
	MBC5    = 0x05
	RUMBLE  = 0x06
)

type GBCF struct {
	fd  io.ReadWriteCloser
	opt serial.OpenOptions
}

func (d *GBCF) SetOptions(options serial.OpenOptions) {
	d.opt = options
}

//Perform initialisation and return a MemCart
func (d *GBCF) MemCart() (memcart.MemCart, error) {
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
func (d *GBCF) Connect() error {
	f, err := serial.Open(d.opt)
	d.fd = f
	return err
}

func (d *GBCF) Disconnect() error {
	e := d.fd.Close()
	if e != nil {
		d.fd = nil
	}
	return e
}

func (d *GBCF) sendPacket(p *Packet) error {
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

func (d *GBCF) ReadDeviceStatus() (*FirmwareVersion, error) {
	p := &Packet{}
	if err := p.setControl(DATA); err != nil {
		return nil, err
	}
	if err := p.setCommand(STATUS); err != nil {
		return nil, err
	}
	if err := p.setSubcommand(NREAD_ID); err != nil {
		return nil, err
	}
	// mbc, algorithm = nil
	if err := d.sendPacket(p); err != nil {
		return nil, err
	}
	p, err := d.receivePacket()
	if err != nil {
		return nil, err
	}
	if err := p.Check(); err != nil {
		return nil, err
	}
	return p.DeviceStatusShort(), nil
}

func (d *GBCF) ReadStatus() (*FirmwareVersion, *DeviceCartInfo, error) {
	p := &Packet{}
	if err := p.setControl(DATA); err != nil {
		return nil, nil, err
	}
	if err := p.setCommand(STATUS); err != nil {
		return nil, nil, err
	}
	if err := p.setSubcommand(READ_ID); err != nil {
		return nil, nil, err
	}

	if err := d.sendPacket(p); err != nil {
		return nil, nil, err
	}
	p, err := d.receivePacket()
	if err != nil {
		return nil, nil, err
	}
	if err := p.Check(); err != nil {
		return nil, nil, err
	}
	fw, dci := p.DeviceStatusLong()
	return fw, dci, nil
}

func (d *GBCF) GBCartInfo() (*GBCartInfo, error) {
	_, dci, err := d.ReadStatus()
	if err != nil {
		return nil, err
	}
	return dci.GBCartInfo(), nil
}

func (dci *DeviceCartInfo) GBCartInfo() *GBCartInfo {
	g := &GBCartInfo{}
	g.Manufacturer = manufacturers[dci.ManufacturerID]
	g.LogoCorrect = dci.LogoCorrect != 0
	g.ChipID = dci.ChipID
	g.CartType = carts[dci.TypeID]
	g.BBL = dci.BBL != 0
	g.CGB = dci.CGB != 0
	g.SGB = dci.SGB != 0
	g.CRC16 = dci.CRC16
	g.GameName = dci.GameName[:strings.Index(dci.GameName, "\x00")]
	g.ROMSize = romSizes[dci.ROMSize]
	g.RAMSize = ramSizes[dci.RAMSize]
	return g
}

/*
// Receive a character when a control character is expected
func (d *GBCF) receive_char() (byte, error) {
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
func (d *GBCF) receivePacket() (*Packet, error) {
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
func (d *GBCF) Handshake() error {
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
