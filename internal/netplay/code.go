// Package netplay connects players directly, with no central server. One
// player's game hosts the match and is the referee for everyone else.
package netplay

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// Address is a host's IPv4 address and TCP port.
type Address struct {
	IP   [4]byte
	Port uint16
}

func (a Address) String() string {
	return fmt.Sprintf("%d.%d.%d.%d:%d", a.IP[0], a.IP[1], a.IP[2], a.IP[3], a.Port)
}

// Crockford base32, chosen because it has no easily confused symbols.
const codeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

const codeSymbols = 16

// ErrBadCode reports a room code that can't be read.
var ErrBadCode = errors.New("that room code doesn't look right")

// EncodeCode packs an address and access token into a room code.
func EncodeCode(a Address, token [4]byte) string {
	var raw [10]byte
	copy(raw[:4], a.IP[:])
	binary.BigEndian.PutUint16(raw[4:6], a.Port)
	copy(raw[6:], token[:])

	var out strings.Builder
	var acc uint32
	bits, written := 0, 0
	for _, b := range raw {
		acc = acc<<8 | uint32(b)
		bits += 8
		for bits >= 5 {
			bits -= 5
			if written > 0 && written%4 == 0 {
				out.WriteByte('-')
			}
			out.WriteByte(codeAlphabet[acc>>bits&31])
			written++
		}
	}
	return out.String()
}

// DecodeCode reads a room code, forgiving case, spacing and look-alike
// characters so that a code read aloud or retyped still works.
func DecodeCode(code string) (Address, [4]byte, error) {
	var a Address
	var token [4]byte
	clean := strings.NewReplacer(" ", "", "-", "", "\n", "", "\r", "", "\t", "",
		"O", "0", "I", "1", "L", "1").Replace(strings.ToUpper(code))
	if len(clean) != codeSymbols {
		return a, token, ErrBadCode
	}
	var raw []byte
	var acc uint32
	bits := 0
	for _, c := range []byte(clean) {
		v := strings.IndexByte(codeAlphabet, c)
		if v < 0 {
			return a, token, ErrBadCode
		}
		acc = acc<<5 | uint32(v)
		bits += 5
		if bits >= 8 {
			bits -= 8
			raw = append(raw, byte(acc>>bits))
		}
	}
	copy(a.IP[:], raw[:4])
	a.Port = binary.BigEndian.Uint16(raw[4:6])
	copy(token[:], raw[6:10])
	return a, token, nil
}
