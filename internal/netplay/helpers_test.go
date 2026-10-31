package netplay

import (
	"encoding/binary"
	"net"
	"time"
)

// sendOversizedHello claims a frame far larger than the host accepts.
func sendOversizedHello(code string) error {
	addr, _, err := DecodeCode(code)
	if err != nil {
		return err
	}
	conn, err := net.DialTimeout("tcp", addr.String(), time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	w := newWire(conn, maxHostFrame)
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], 1<<30)
	if _, err := w.fw.Write(size[:]); err != nil {
		return err
	}
	return w.fw.Flush()
}
