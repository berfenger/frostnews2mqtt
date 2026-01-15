package sunspec

import (
	"context"
	"errors"
	"slices"

	"github.com/berfenger/frostnews2mqtt/pkg/modbus"
	"github.com/berfenger/frostnews2mqtt/pkg/util/logutil"
	"go.uber.org/zap"
)

const (
	SUNS_BASE_ADDR_1           uint16 = 40000
	SUNS_BASE_ADDR_2           uint16 = 40001
	SUNS_BASE_ADDR_3           uint16 = 50000
	SUNS_DEVICE_WELL_KNOWN_ID  string = "SunS" // 0x53756e53
	SUNS_MODEL_ID_COMMON       uint16 = 1
	SUNS_MODEL_ID_INVERTER_MIN uint16 = 101
	SUNS_MODEL_ID_INVERTER_MAX uint16 = 103
	SUNS_MODEL_ID_NAMEPLATE    uint16 = 120
	SUNS_MODEL_ID_STATUS       uint16 = 122
	SUNS_MODEL_ID_CONTROLS     uint16 = 123
	SUNS_MODEL_ID_STORAGE      uint16 = 124
	SUNS_MODEL_ID_MPPT         uint16 = 160
	SUNS_MODEL_ID_ACMETER_MIN  uint16 = 201
	SUNS_MODEL_ID_ACMETER_MAX  uint16 = 204
	SUNS_MODEL_ID_END          uint16 = 0xFFFF
)

var SUNS_BASE_ADDRESSES []uint16 = []uint16{SUNS_BASE_ADDR_1, SUNS_BASE_ADDR_2, SUNS_BASE_ADDR_3}

var (
	ErrInvalidModelLength = errors.New("invalid sunspec model length")
	ErrInvalidModelId     = errors.New("invalid sunspec model id")
)

type ModelHeader struct {
	id       uint16
	baseAddr uint16
	length   uint16
}

func (model ModelHeader) isEndBlock() bool {
	return model.id == SUNS_MODEL_ID_END
}

func (model ModelHeader) ToReadableModel(ctx context.Context, client modbus.ModbusReaderWriter) ReadableModel {
	return ReadableModel{
		ModelHeader: model,
		client:      client,
		ctx:         ctx,
	}
}

func (model ModelHeader) ToWritableModel(ctx context.Context, client modbus.ModbusReaderWriter) WritableModel {
	return WritableModel{
		ReadableModel: model.ToReadableModel(ctx, client),
	}
}

type ReadableModel struct {
	ModelHeader
	client modbus.ModbusReaderWriter
	ctx    context.Context
}

func (model ReadableModel) Read() (modelData, error) {

	bytes, err := model.client.ReadRawBytes(model.baseAddr, model.length*2, modbus.HOLDING_REGISTER)
	if err != nil {
		logutil.Error(model.ctx, "failed to read SunSpec model",
			zap.Uint16("model_id", model.id),
			zap.Error(err))
		return modelData{}, err
	}

	return modelData{
		client: model.client,
		header: model.ModelHeader,
		data:   bytes,
	}, nil
}

type modelData struct {
	client modbus.ModbusReaderWriter
	header ModelHeader
	data   []byte
}

func (model *modelData) readRegister(address uint16) uint16 {
	return model.client.BytesToUint16(model.data[address*2 : address*2+2])
}

func (model *modelData) readRegisters(address uint16, count uint16) []uint16 {
	result := make([]uint16, count)
	for i := range result {
		result[i] = model.client.BytesToUint16(model.data[address*2+uint16(i)*2 : address*2+uint16(i)*2+2])
	}
	return result
}

func (model *modelData) readUint32(address uint16) uint32 {
	return model.client.BytesToUint32(model.data[address*2 : address*2+4])
}

// nolint unused suppressed for future use
func (model *modelData) readUint32s(address uint16, count uint16) []uint32 {
	return model.client.BytesToUint32s(model.data[address*2 : address*2+count*4])
}

func (model *modelData) readString(address uint16, size uint16) string {
	bytes := model.data[address*2 : address*2+size]

	f := slices.Index(bytes, 0x00)
	if f >= 0 {
		return string(bytes[:f])
	}
	return string(bytes)
}

type WritableModel struct {
	ReadableModel
}

func (model *WritableModel) WriteRegister(addr uint16, value uint16) error {
	return model.client.WriteRegister(model.baseAddr+addr, value)
}

func (model *WritableModel) WriteRegisters(addr uint16, values []uint16) error {
	return model.client.WriteRegisters(model.baseAddr+addr, values)
}
