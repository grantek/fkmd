// Game Boy cart flasher documented by jrodrigo.net/cart-flasher and www.reinerziegler.de/readplus.htm
// Original PC driver software from sourceforge.net/projects/gbcf
package gbcf

import (
	"errors"
	"fmt"
	"time"
	//"github.com/grantek/fkmd/gbcart"
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

const (
	// enum cchars
	ACK  = 0xAA
	NAK  = 0xF0
	END  = 0x0F
	DATA = 0x55

	SERIAL_TIMEOUT = 3 * time.Second
	DELETE_TIMEOUT = 60 * time.Second
	PACKETSIZE     = 72
	FRAMESIZE      = 64
	AUTOSIZE       = -1
	PORTS_COUNT    = 4
	VER            = "1.1"

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

	RESERVED = 0x00
	NREAD_ID = 0x00
	READ_ID  = 0x01

	// operations
	RROM = 0x00
	RRAM = 0x01
	WROM = 0x02
	WRAM = 0x03
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
func (d *Gbcf) send_packet([]byte) error {
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
