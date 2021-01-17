// Game Boy cart flasher documented by jrodrigo.net/cart-flasher and www.reinerziegler.de/readplus.htm
// Original PC driver software from sourceforge.net/projects/gbcf
package gbcf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	//"github.com/grantek/fkmd/gbcart"
	"github.com/grantek/fkmd/memcart"
	"github.com/jacobsa/go-serial/serial"
)

// array of cart types - source GB CPU Manual
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

var RomSizes = map[byte]string{
	0x00: "32KB",
	0x01: "64KB",
	0x02: "128KB",
	0x03: "256KB",
	0x04: "512KB",
	0x05: "1MB",
	0x06: "2MB",
	0x07: "4MB", // Not in original code, not sure if device will detect this.
	0x52: "1.1MB",
	0x53: "1.2MB",
	0x54: "1.5MB",
}

var RamSizes = map[byte]string{
	0x00: "0KB",
	0x01: "2KB",
	0x02: "8KB",
	0x03: "32KB",
	0x04: "128KB",
}

var RomSizeBytes = map[byte]int{
	0x00: 32768,
	0x01: 65536,
	0x02: 131072,
	0x03: 262144,
	0x04: 524288,
	0x05: 524288,
	0x06: 2097152,
	0x07: 4194304,
	0x52: 1179648,
	0x53: 1310720,
	0x54: 1572864,
}

var RamSizeBytes = map[byte]int{
	0x00: 0,
	0x01: 2048,
	0x02: 8192,
	0x03: 32768,
	0x04: 131072,
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
)

const (
	//enum error_t (signed char)
	//TODO: verbose errors
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

// DeviceCartInfo stores the cart-related parts of STATUS(READ_ID)
type DeviceCartInfo struct {
	// Flash chip data.
	ManufacturerID byte
	ChipID         byte
	BBL            bool // Boot Block Locked
	// Info about loaded game.
	LogoCorrect   bool
	CGB           bool // Color Game Boy
	SGB           bool // Super Game Boy
	ROMSize       byte //[6]byte
	RAMSize       byte //[6]byte
	CRC16         uint16
	TypeID        byte   //typ          [30]byte
	GameNameBytes []byte //[17]byte
}

func (dci *DeviceCartInfo) GBCartInfo() *GBCartInfo {
	g := &GBCartInfo{}
	g.Manufacturer = manufacturers[dci.ManufacturerID]
	g.LogoCorrect = dci.LogoCorrect
	g.ChipID = dci.ChipID
	g.CartType = carts[dci.TypeID]
	g.BBL = dci.BBL
	g.CGB = dci.CGB
	g.SGB = dci.SGB
	g.CRC16 = dci.CRC16
	runes := make([]rune, 0, len(dci.GameNameBytes))
	for _, b := range dci.GameNameBytes {
		if !strconv.IsPrint(rune(b)) {
			break
		}
		runes = append(runes, rune(b))
	}
	g.GameNamePrintable = string(runes)
	g.ROMSize = RomSizes[dci.ROMSize]
	g.RAMSize = RamSizes[dci.RAMSize]
	return g
}

// FirmwareVersion represents the device version sent by STATUS(NREAD_ID)
type FirmwareVersion struct {
	// BCD, formatted by original code as ("%d%d.%d%d", v11,v12,v21,v22)
	Ver11 byte
	Ver12 byte
	Ver21 byte
	Ver22 byte
}

type GBCF struct {
	fd  io.ReadWriteCloser
	opt serial.OpenOptions
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

func (d *GBCF) GBCartInfo() (*GBCartInfo, error) {
	_, dci, err := d.ReadStatus()
	if err != nil {
		return nil, err
	}
	return dci.GBCartInfo(), nil
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

func (d *GBCF) ReadDeviceStatus() (*FirmwareVersion, error) {
	pc := &PacketConfig{
		Control:    DATA,
		Command:    STATUS,
		Subcommand: NREAD_ID,
		Algorithm:  ALG16,
		MBC:        MBCAUTO,
	}
	p, err := pc.Packet()
	if err != nil {
		return nil, err
	}
	if err := d.SendPacket(p); err != nil {
		return nil, err
	}
	p, err = d.ReceivePacket()
	if err != nil {
		return nil, err
	}
	if err := p.Check(); err != nil {
		return nil, err
	}
	return p.DeviceStatusShort(), nil
}

// readRAM reads all of RAM up to len(b), and returns an error if b is not
// completely filled.
func (d *GBCF) ReadRAM(b []byte) error {
	want := len(b)
	pgc := 1
	switch {
	case want == 2*1024:
	case want > 0 && want%(8*1024) == 0:
		pgc = want / (8 * 1024)
	default:
		return fmt.Errorf("readRAM: invalid buffer size %d bytes, should be 2KiB or N*8KiB", want)
	}
	pc := &PacketConfig{
		Control:    DATA,
		Command:    CONFIG,
		Subcommand: RRAM,
		Algorithm:  ALG16,
		MBC:        MBCAUTO,
		PageCount:  pgc,
	}
	p, err := pc.Packet()
	if err != nil {
		return err
	}
	if err := d.SendPacket(p); err != nil {
		return err
	}
	fin := false
	n := 0
	for fin == false {
		page := (n / FRAMESIZE) / 128 // 8kiB RAM page / 64B packet payload
		packet := (n / FRAMESIZE) % 128
		p, err = d.ReceivePacket()
		if err != nil {
			// TODO: original code has 10 retries on serial TIMEOUT.
			return err
		}
		switch pt := p.Control(); pt {
		case DATA:
		case END:
			if n == want {
				fmt.Printf("DEBUG: got END with full buffer at %d\n", n)
				return nil
			}
			fallthrough
		default:
			return fmt.Errorf("readRAM: unexpected control byte  %q: got %d bytes, want %d", pt, n, want)
		}
		if err := p.Check(); err != nil {
			return err
		}
		cm := p.Command()
		if cm != NORMAL_DATA && cm != LAST_DATA {
			return fmt.Errorf("readRAM: unexpected command byte %q: got %d bytes, want %d", cm, n, want)
		}
		pk := int(p.bytes[3])
		pg := int(p.bytes[4])*256 + int(p.bytes[5])
		if packet != pk || page != pg {
			return fmt.Errorf("readRAM: packet out of sequence: got %d,%d bytes, want %d,%d", pk, pg, packet, page)
		}
		if n+FRAMESIZE == want {
			if want == 2*1024 {
				d.SendControl(END)
				fin = true
			}
			if cm == LAST_DATA {
				fmt.Printf("DEBUG: got LAST_DATA at %d\n", n)
				fin = true
			} else {
				fmt.Printf("DEBUG: filled buffer with NORMAL_DATA command at %d\n", n)
				d.SendControl(ACK)
			}
		} else {
			if packet == 127 {
				//TODO: check if a different response is needed
				d.SendControl(ACK)
			} else {
				d.SendControl(ACK)
			}
		}
		copy(b[n:n+FRAMESIZE], p.Frame())
		n = n + FRAMESIZE
		fmt.Printf("DEBUG: buffer: %d page: %d packet: %d pk: %d pg: %d\n", n, page, packet, pk, pg)
	}
	return nil
}

func (d *GBCF) ReadStatus() (*FirmwareVersion, *DeviceCartInfo, error) {
	pc := &PacketConfig{
		Control:    DATA,
		Command:    STATUS,
		Subcommand: READ_ID,
		Algorithm:  ALG16,
		MBC:        MBCAUTO,
	}
	p, err := pc.Packet()
	if err != nil {
		return nil, nil, err
	}
	if err := d.SendPacket(p); err != nil {
		return nil, nil, err
	}
	p, err = d.ReceivePacket()
	if err != nil {
		return nil, nil, err
	}
	if err := p.Check(); err != nil {
		return nil, nil, err
	}
	fw, dci := p.DeviceStatusLong()
	return fw, dci, nil
}

/*
// Receive a character when a control character is expected
func (d *GBCF) ReceiveControl() (byte, error) {
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

// ReceivePacket receives a Packet
func (d *GBCF) ReceivePacket() (*Packet, error) {
	p := &Packet{}
	n, err := d.fd.Read(p.bytes[:1])
	if err != nil {
		return nil, err
	}
	if n < 1 {
		return nil, errors.New("Failed to read first byte of packet.")
	}
	if p.bytes[0] < 0 {
		return nil, fmt.Errorf("Received error value in ControlByte: %d", p.bytes[0])
	}
	// Non-DATA packets only send one byte over serial.
	if ControlByte(p.bytes[0]) != DATA {
		return p, nil
	}
	// Read rest of DATA packet.
	n, err = d.fd.Read(p.bytes[1:])
	if err != nil {
		return nil, err
	}
	if n < PACKETSIZE-1 {
		return nil, fmt.Errorf("Short packet: read %d bytes.", n+1)
	}
	return p, nil
}

//SendControl sends a control byte.
func (d *GBCF) SendControl(cb ControlByte) error {
	_, err := d.fd.Write([]byte{byte(cb)})
	return err
}

// SendPacket sends a packet.
func (d *GBCF) SendPacket(p *Packet) error {
	b, err := p.Bytes()
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

func (d *GBCF) SetOptions(options serial.OpenOptions) {
	d.opt = options
}

// WriteRAM writes b to cartridge RAM
func (d *GBCF) WriteRAM(b []byte) error {
	have := len(b)
	pgc := 1
	switch {
	case have == 2*1024:
	case have > 0 && have%(8*1024) == 0:
		pgc = have / (8 * 1024)
	default:
		return fmt.Errorf("WriteRAM: invalid buffer size %d bytes, should be 2KiB or N*8KiB", have)
	}
	pc := &PacketConfig{
		Control:    DATA,
		Command:    CONFIG,
		Subcommand: WRAM,
		Algorithm:  ALG16,
		MBC:        MBCAUTO,
		PageCount:  pgc,
	}
	p, err := pc.Packet()
	if err != nil {
		return err
	}
	if err := d.SendPacket(p); err != nil {
		return err
	}
	// TODO: original code has 10 retries
	p, err = d.ReceivePacket()
	if err != nil {
		return err
	}
	cb := p.Control()
	if cb != ACK {
		return fmt.Errorf("WriteRAM: Unexpected response ControlByte to WRAM: %s", p.Control().String())
	}
	fin := false
	n := 0
	for fin == false {
		page := uint16((n / FRAMESIZE) / 128) // 8kiB RAM page / 64B packet payload
		packet := uint8((n / FRAMESIZE) % 128)
		c := NORMAL_DATA
		if n+FRAMESIZE >= len(b) {
			c = LAST_DATA
			fin = true
		}
		pc = &PacketConfig{
			Control:     DATA,
			Command:     c,
			Subcommand:  RESERVED,
			PacketIndex: packet,
			PageIndex:   page,
		}
		p, err = pc.Packet()
		if err != nil {
			return err
		}
		n = n + p.Pack(b[n:n+FRAMESIZE])
		fmt.Printf("DEBUG: buffer: %d page: %d packet: %d\n", n, page, packet)
		if err := d.SendPacket(p); err != nil {
			return err
		}
		p, err = d.ReceivePacket()
		if err != nil {
			return err
		}
		if cb := p.Control(); cb != ACK {
			return fmt.Errorf("WriteRAM: Unexpected response ControlByte to sent data: %s", cb.String())
		}
	}
	return nil
}

// GBCartInfo is human-readable version of DeviceCartInfo
type GBCartInfo struct {
	Manufacturer      string
	ChipID            byte
	BBL               bool
	LogoCorrect       bool
	CGB               bool
	SGB               bool
	ROMSize           string
	RAMSize           string
	CRC16             uint16
	CartType          string
	GameNamePrintable string
}

type GBCartRAM struct {
	d          *GBCF
	addressCur int64
	size       int64
}

func (m *GBCartRAM) AlwaysWritable() bool {
	return true
}

func (m *GBCartRAM) Name() string {
	return "gbcartram"
}

func (m *GBCartRAM) Read(p []byte) (n int, err error) {
	// wrap readRAM bytes with a reader
	return 0, nil
}

func (m *GBCartRAM) Size() int64 {
	return m.size
}

type Packet struct {
	bytes [PACKETSIZE]byte
}

// Bytes returns a copy of the Packet bytes, with CRC filled in.
func (p *Packet) Bytes() ([]byte, error) {
	c := p.CRC16()
	p.bytes[PACKETSIZE-2] = byte(c / 256)
	p.bytes[PACKETSIZE-1] = byte(c % 256)
	b := make([]byte, PACKETSIZE)
	if n := copy(b, p.bytes[:]); n != PACKETSIZE {
		return nil, fmt.Errorf("Serialise: copied %d of %d bytes", n, PACKETSIZE)
	}
	return b, nil
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

// Pack fills a DATA packet for writing data to the device.
func (p *Packet) Pack(b []byte) int {
	return copy(p.bytes[6:6+FRAMESIZE], b)
}

// Check performs a CRC16 on a DATA packet (other control messages that
// don't carry data don't have a check sum)
func (p *Packet) Check() error {
	if p.Control() != DATA {
		return errors.New("Packet is not marked as DATA packet.")
	}
	c := p.CRC16()
	if p.bytes[PACKETSIZE-2] != byte(c/256) ||
		p.bytes[PACKETSIZE-1] != byte(c%256) {
		return errors.New("CRC error in received packet.")
	}
	return nil
}

// Command returns the command byte from the packet as a CommandByte.
func (p *Packet) Command() CommandByte {
	return CommandByte(p.bytes[1])
}

// Control returns the control byte from the packet as a ControlByte.
func (p *Packet) Control() ControlByte {
	return ControlByte(p.bytes[0])
}

func (p *Packet) DeviceStatusLong() (*FirmwareVersion, *DeviceCartInfo) {
	fw := p.DeviceStatusShort()
	dci := &DeviceCartInfo{}
	dci.ManufacturerID = p.bytes[4]
	dci.ChipID = p.bytes[5]
	dci.BBL = p.bytes[6]&0x01 == 0x01
	dci.LogoCorrect = p.bytes[8] == 1
	gnb := p.bytes[9:25]
	i := bytes.IndexByte(gnb, 0x00)
	if i >= 0 {
		gnb = gnb[:i]
	}
	dci.GameNameBytes = make([]byte, len(gnb))
	copy(dci.GameNameBytes, gnb)
	dci.CGB = p.bytes[24] == 0x80
	dci.SGB = p.bytes[27] == 0x03
	dci.TypeID = p.bytes[28]
	dci.ROMSize = p.bytes[29]
	dci.RAMSize = p.bytes[30]
	dci.CRC16 = 256*uint16(p.bytes[35]) + uint16(p.bytes[36])
	return fw, dci
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

// Frame returns a slice (not a copy) of the underlying []byte that
// refers to the packet's data frame.
func (p *Packet) Frame() []byte {
	return p.bytes[6 : 6+FRAMESIZE]
}

type PacketConfig struct {
	Control     ControlByte
	Command     CommandByte
	Subcommand  SubcommandByte
	Algorithm   byte   // *READ_ID, RRAM?
	MBC         byte   // *READ_ID, RRAM?
	PageCount   int    // RRAM
	PacketIndex uint8  // WRAM
	PageIndex   uint16 // WRAM
}

// Packet validates the PacketConfig and returns a Packet.
// DATA packets only: use SendControl for others.
func (pc *PacketConfig) Packet() (*Packet, error) {
	p := &Packet{}
	// generated functions return "%T(%d)" for unknown values
	if strings.HasPrefix(pc.Control.String(), "ControlByte") {
		return nil, fmt.Errorf("Invalid control character for packet: %X", pc.Control)
	}
	p.bytes[0] = byte(pc.Control)
	if strings.HasPrefix(pc.Command.String(), "CommandByte") {
		return nil, fmt.Errorf("Invalid command character for packet: %X", pc.Command)
	}
	p.bytes[1] = byte(pc.Command)
	// Subcommand values have a range of 0 to N-1 for commands with N subcommands.
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
	if int(pc.Subcommand) >= n {
		return nil, fmt.Errorf("Subcommand value %d is invalid for %s.", pc.Subcommand, cb.String())
	}
	p.bytes[2] = byte(pc.Subcommand)
	// TODO: nonzero Algorithm, MBC for STATUS
	switch pc.Command {
	case CONFIG:
		p.bytes[6] = byte((pc.PageCount - 1) / 256)
		p.bytes[7] = byte((pc.PageCount - 1) % 256)
	case NORMAL_DATA:
		fallthrough
	case LAST_DATA:
		p.bytes[3] = pc.PacketIndex
		p.bytes[4] = uint8(pc.PageIndex / 256)
		p.bytes[5] = uint8(pc.PageIndex % 256)
	}
	return p, nil
}
