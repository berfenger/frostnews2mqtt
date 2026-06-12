package modbus

import (
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"syscall"
	"time"

	"github.com/simonvetter/modbus"
	"go.uber.org/zap"
)

type ModbusReaderWriterClient struct {
	client           *modbus.ModbusClient
	unitId           uint8
	endianess        modbus.Endianness
	wordOrder        modbus.WordOrder
	instrumentations []ModbusInstrumentation
}

func NewModbusTCPReaderWriterClient(deviceName string, ip string, port uint, unitId uint8, timeout time.Duration,
	logger *zap.Logger, instrumentations []ModbusInstrumentation) (ModbusReaderWriter, error) {

	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     fmt.Sprintf("tcp://%s:%d", ip, port),
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}

	// set device address
	if unitId > 0 {
		err = client.SetUnitId(unitId)
		if err != nil {
			return nil, err
		}
	}

	return &ModbusReaderWriterClient{
		client:           client,
		unitId:           unitId,
		endianess:        modbus.BIG_ENDIAN,
		wordOrder:        modbus.HIGH_WORD_FIRST,
		instrumentations: CreateModbusInstrumentation(deviceName, unitId, logger, instrumentations),
	}, nil
}

func (reader ModbusReaderWriterClient) ReadString(address uint16, size uint16) (string, error) {
	bytes, err := reader.ReadRawBytes(address, size, HOLDING_REGISTER)
	if err != nil {
		return "", err
	}
	f := slices.Index(bytes, 0x00)
	if f >= 0 {
		return string(bytes[:f]), nil
	}
	return string(bytes), nil
}

func (reader ModbusReaderWriterClient) ReadRegister(addr uint16, regType RegType) (uint16, error) {
	defer RecordModbusTimer("ReadRegister", addr, 1, reader.unitId, reader.instrumentations)()
	return reader.client.ReadRegister(addr, modbus.RegType(regType))
}

func (reader ModbusReaderWriterClient) ReadRegisters(addr uint16, quantity uint16, regType RegType) ([]uint16, error) {
	defer RecordModbusTimer("ReadRegisters", addr, quantity, reader.unitId, reader.instrumentations)()
	return reader.client.ReadRegisters(addr, quantity, modbus.RegType(regType))
}

// nolint unused suppressed for future use
func (reader ModbusReaderWriterClient) ReadUint32(addr uint16, regType RegType) (uint32, error) {
	defer RecordModbusTimer("ReadUint32", addr, 2, reader.unitId, reader.instrumentations)()
	return reader.client.ReadUint32(addr, modbus.RegType(regType))
}

func (reader ModbusReaderWriterClient) ReadRawBytes(addr uint16, quantity uint16, regType RegType) ([]byte, error) {
	defer RecordModbusTimer("ReadRawBytes", addr, quantity, reader.unitId, reader.instrumentations)()
	return reader.client.ReadRawBytes(addr, quantity, modbus.RegType(regType))
}

func (reader ModbusReaderWriterClient) WriteRegister(addr uint16, value uint16) error {
	defer RecordModbusTimer("WriteRegister", addr, 1, reader.unitId, reader.instrumentations)()
	return reader.client.WriteRegister(addr, value)
}

func (reader ModbusReaderWriterClient) WriteRegisters(addr uint16, values []uint16) error {
	defer RecordModbusTimer("WriteRegisters", addr, uint16(len(values)), reader.unitId, reader.instrumentations)()
	return reader.client.WriteRegisters(addr, values)
}

func (reader ModbusReaderWriterClient) BytesToUint32s(in []byte) []uint32 {
	return bytesToUint32s(reader.endianess, reader.wordOrder, in)
}

func (reader ModbusReaderWriterClient) BytesToUint32(in []byte) uint32 {
	return bytesToUint32(reader.endianess, reader.wordOrder, in)
}

func (reader ModbusReaderWriterClient) BytesToUint16(in []byte) uint16 {
	return bytesToUint16(reader.endianess, in)
}

func (reader ModbusReaderWriterClient) Open() (err error) {
	return reader.client.Open()
}
func (reader ModbusReaderWriterClient) Close() (err error) {
	return reader.client.Close()
}

// ensure interface compliance
var _ ModbusReaderWriter = (*ModbusReaderWriterClient)(nil)

type AutoReconnectModbusReaderWriterClient struct {
	*ModbusReaderWriterClient
	logger *zap.Logger
}

func NewAutoReconnectModbusTCPReaderWriterClient(deviceName string, ip string, port uint, unitId uint8, timeout time.Duration,
	logger *zap.Logger, instrumentations []ModbusInstrumentation) (ModbusReaderWriter, error) {

	baseClient, err := NewModbusTCPReaderWriterClient(deviceName, ip, port, unitId, timeout, logger, instrumentations)
	if err != nil {
		return nil, err
	}

	return &AutoReconnectModbusReaderWriterClient{
		ModbusReaderWriterClient: baseClient.(*ModbusReaderWriterClient),
		logger:                   logger,
	}, nil
}

func isBrokenPipe(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, io.EOF) ||
		err.Error() == "read: connection reset by peer" ||
		err.Error() == "write: broken pipe" ||
		err.Error() == "EOF"
}

func (r *AutoReconnectModbusReaderWriterClient) reconnectAndRetry(operation func() error) error {
	err := operation()
	if err != nil && isBrokenPipe(err) {
		r.logger.Warn("broken connection detected, attempting reconnection")
		//nolint errcheck
		r.client.Close()
		if reopenErr := r.client.Open(); reopenErr != nil {
			r.logger.Error("error reopening connection", zap.Error(reopenErr))
			return err
		}
		return operation()
	}
	return err
}

func (r *AutoReconnectModbusReaderWriterClient) ReadRegister(addr uint16, regType RegType) (result uint16, err error) {
	err = r.reconnectAndRetry(func() error {
		var e error
		result, e = r.ModbusReaderWriterClient.ReadRegister(addr, regType)
		return e
	})
	return
}

func (r *AutoReconnectModbusReaderWriterClient) ReadRegisters(addr uint16, quantity uint16, regType RegType) (result []uint16, err error) {
	err = r.reconnectAndRetry(func() error {
		var e error
		result, e = r.ModbusReaderWriterClient.ReadRegisters(addr, quantity, regType)
		return e
	})
	return
}

func (r *AutoReconnectModbusReaderWriterClient) ReadUint32(addr uint16, regType RegType) (result uint32, err error) {
	err = r.reconnectAndRetry(func() error {
		var e error
		result, e = r.ModbusReaderWriterClient.ReadUint32(addr, regType)
		return e
	})
	return
}

func (r *AutoReconnectModbusReaderWriterClient) ReadRawBytes(addr uint16, quantity uint16, regType RegType) (result []byte, err error) {
	err = r.reconnectAndRetry(func() error {
		var e error
		result, e = r.ModbusReaderWriterClient.ReadRawBytes(addr, quantity, regType)
		return e
	})
	return
}

func (r *AutoReconnectModbusReaderWriterClient) WriteRegister(addr uint16, value uint16) error {
	return r.reconnectAndRetry(func() error {
		return r.ModbusReaderWriterClient.WriteRegister(addr, value)
	})
}

func (r *AutoReconnectModbusReaderWriterClient) WriteRegisters(addr uint16, values []uint16) error {
	return r.reconnectAndRetry(func() error {
		return r.ModbusReaderWriterClient.WriteRegisters(addr, values)
	})
}

// ensure interface compliance
var _ ModbusReaderWriter = (*AutoReconnectModbusReaderWriterClient)(nil)
