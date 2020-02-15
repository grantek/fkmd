package gbcf

import "testing"

func TestCRC(t *testing.T) {
	p := Packet{}
	p.bytes[0] = 0x55
	p.bytes[1] = 0x04
	c := p.generate_crc16()
	if c != 0xA3C1 {
		t.Errorf("CRC(DATA, STATUS, NREAD_ID): got %x, want 0xA3C1", c)
	}
	p.bytes[2] = 0x01
	c = p.generate_crc16()
	if c != 0x9936 {
		t.Errorf("CRC(DATA, STATUS, READ_ID): got %x, want 0x9936", c)
	}
	p.bytes[70] = 0xFF
	p.bytes[71] = 0xFF
	p.bytes[2] = 0x00
	c = p.generate_crc16()
	if c != 0xA3C1 {
		t.Errorf("CRC(DATA, STATUS, NREAD_ID, ..., 0xFFFF): got %x, want 0xA3C1", c)
	}
	p.bytes[2] = 0x01
	c = p.generate_crc16()
	if c != 0x9936 {
		t.Errorf("CRC(DATA, STATUS, READ_ID, ..., 0xFFFF): got %x, want 0x9936", c)
	}
}
