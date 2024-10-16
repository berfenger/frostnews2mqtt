package sunspec_modbus

import (
	"math"
	"slices"
	"time"

	"github.com/simonvetter/modbus"
)

type ModbusClient struct {
	client     *modbus.ModbusClient
	instrument []ModbusInstrument
}

type ModbusInstrument struct {
	RecordTime func(fnName string, readTime time.Duration)
}

func (reader ModbusClient) readString(address uint16, size uint16) (string, error) {
	bytes, err := reader.readRawBytes(address, size, modbus.HOLDING_REGISTER)
	if err != nil {
		return "", err
	}
	f := slices.Index(bytes, 0x00)
	if f >= 0 {
		return string(bytes[:f]), nil
	}
	return string(bytes), nil
}

func (reader ModbusClient) applySF(number uint16, sf uint16) float64 {
	return float64(number) * math.Pow(10, float64(int16(sf)))
}

func (reader ModbusClient) applySFInv(number uint16, sf uint16) float64 {
	return float64(number) / math.Pow(10, float64(int16(sf)))
}

func (reader ModbusClient) applySFint16(number int16, sf uint16) float64 {
	return float64(number) * math.Pow(10, float64(int16(sf)))
}

func (reader ModbusClient) applySFuint32(number uint32, sf uint16) float64 {
	return float64(number) * math.Pow(10, float64(int16(sf)))
}

func (reader ModbusClient) applySFfloat64Inv(number float64, sf uint16) float64 {
	return number / math.Pow(10, float64(int16(sf)))
}

func (reader ModbusClient) readRegister(addr uint16, regType modbus.RegType) (uint16, error) {
	defer RecordTimer("ReadRegister", reader.instrument)()
	return reader.client.ReadRegister(addr, regType)
}

func (reader ModbusClient) readRegisters(addr uint16, quantity uint16, regType modbus.RegType) ([]uint16, error) {
	defer RecordTimer("ReadRegisters", reader.instrument)()
	return reader.client.ReadRegisters(addr, quantity, regType)
}

func (reader ModbusClient) readUint32(addr uint16, regType modbus.RegType) (uint32, error) {
	defer RecordTimer("ReadUint32", reader.instrument)()
	return reader.client.ReadUint32(addr, regType)
}

func (reader ModbusClient) readRawBytes(addr uint16, quantity uint16, regType modbus.RegType) ([]byte, error) {
	defer RecordTimer("ReadRawBytes", reader.instrument)()
	return reader.client.ReadRawBytes(addr, quantity, regType)
}

func (reader ModbusClient) writeRegister(addr uint16, value uint16) error {
	defer RecordTimer("WriteRegister", reader.instrument)()
	return reader.client.WriteRegister(addr, value)
}

func (reader ModbusClient) writeRegisters(addr uint16, values []uint16) error {
	defer RecordTimer("WriteRegisters", reader.instrument)()
	return reader.client.WriteRegisters(addr, values)
}

func RecordTimer(name string, instrument []ModbusInstrument) func() {
	if instrument == nil {
		return func() {}
	}

	start := time.Now()
	return func() {
		duration := time.Since(start)
		for i := range instrument {
			instrument[i].RecordTime(name, duration)
		}
	}
}
