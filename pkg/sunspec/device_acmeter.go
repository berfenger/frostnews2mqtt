package sunspec

import (
	"context"

	"github.com/berfenger/frostnews2mqtt/pkg/modbus"
)

type DeviceACMeter struct {
	common  ReadableModel
	acMeter ReadableModel
}

func NewSunSpecDeviceACMeter(ctx context.Context, client modbus.ModbusReaderWriter, deviceModels SunspecDeviceModels) (*DeviceACMeter, error) {
	// search common model
	commonModel := deviceModels.FindModelById(SUNS_MODEL_ID_COMMON)

	// search acMeter model
	acMeterModel := deviceModels.FindModelByIdRange(SUNS_MODEL_ID_ACMETER_MIN, SUNS_MODEL_ID_ACMETER_MAX)

	// check found models
	if commonModel != nil && acMeterModel != nil {
		return &DeviceACMeter{
			common:  commonModel.ToReadableModel(ctx, client),
			acMeter: acMeterModel.ToReadableModel(ctx, client),
		}, nil
	} else {
		return nil, ErrAcMeterNotFound
	}
}

func (device *DeviceACMeter) ReadCommonModel() (*CommonModelData, error) {
	return ReadModelCommon(device.common)
}

func (device *DeviceACMeter) ReadACMeterModel() (*ACMeterModelData, error) {
	return ReadModelACMeter(device.acMeter)
}
