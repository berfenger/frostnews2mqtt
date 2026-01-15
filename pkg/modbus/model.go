package modbus

type RegType uint

const (
	HOLDING_REGISTER RegType = 0
	INPUT_REGISTER   RegType = 1
)

type ModbusReaderWriter interface {
	Open() (err error)
	Close() (err error)
	ReadString(address uint16, size uint16) (string, error)
	ReadRegister(addr uint16, regType RegType) (uint16, error)
	ReadRegisters(addr uint16, quantity uint16, regType RegType) ([]uint16, error)
	ReadUint32(addr uint16, regType RegType) (uint32, error)
	ReadRawBytes(addr uint16, quantity uint16, regType RegType) ([]byte, error)
	WriteRegister(addr uint16, value uint16) error
	WriteRegisters(addr uint16, values []uint16) error
	BytesToUint32s(in []byte) []uint32
	BytesToUint32(in []byte) uint32
	BytesToUint16(in []byte) uint16
}
