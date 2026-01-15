package modbus

import (
	"encoding/binary"

	"github.com/simonvetter/modbus"
)

func bytesToUint32s(endianness modbus.Endianness, wordOrder modbus.WordOrder, in []byte) (out []uint32) {
	var u32 uint32

	for i := 0; i < len(in); i += 4 {
		switch endianness {
		case modbus.BIG_ENDIAN:
			if wordOrder == modbus.HIGH_WORD_FIRST {
				u32 = binary.BigEndian.Uint32(in[i : i+4])
			} else {
				u32 = binary.BigEndian.Uint32(
					[]byte{in[i+2], in[i+3], in[i+0], in[i+1]})
			}
		case modbus.LITTLE_ENDIAN:
			if wordOrder == modbus.LOW_WORD_FIRST {
				u32 = binary.LittleEndian.Uint32(in[i : i+4])
			} else {
				u32 = binary.LittleEndian.Uint32(
					[]byte{in[i+2], in[i+3], in[i+0], in[i+1]})
			}
		}

		out = append(out, u32)
	}

	return
}

func bytesToUint16(endianness modbus.Endianness, in []byte) (out uint16) {
	switch endianness {
	case modbus.BIG_ENDIAN:
		out = binary.BigEndian.Uint16(in)
	case modbus.LITTLE_ENDIAN:
		out = binary.LittleEndian.Uint16(in)
	}

	return
}

func bytesToUint32(endianness modbus.Endianness, wordOrder modbus.WordOrder, bytes []byte) uint32 {
	return bytesToUint32s(endianness, wordOrder, bytes)[0]
}
