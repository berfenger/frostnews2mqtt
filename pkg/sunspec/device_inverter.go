package sunspec

import (
	"context"

	"github.com/berfenger/frostnews2mqtt/pkg/modbus"
)

type DeviceInverter struct {
	Common    ReadableModel
	Inverter  ReadableModel
	Status    ReadableModel
	Nameplate ReadableModel
	Mppt      ReadableModel
	Controls  *ControlsModelWriter
	Storage   *StorageModelWriter
}

func NewSunSpecDeviceInverter(ctx context.Context, client modbus.ModbusReaderWriter, deviceModels SunspecDeviceModels) (*DeviceInverter, error) {
	// search common model
	commonModel := deviceModels.FindModelById(SUNS_MODEL_ID_COMMON)

	// search int+sf inverter model
	inverterModel := deviceModels.FindModelByIdRange(SUNS_MODEL_ID_INVERTER_MIN, SUNS_MODEL_ID_INVERTER_MAX)

	// search status model
	statusModel := deviceModels.FindModelById(SUNS_MODEL_ID_STATUS)

	// search status model
	nameplateModel := deviceModels.FindModelById(SUNS_MODEL_ID_NAMEPLATE)

	// search MPPT model
	mpptModel := deviceModels.FindModelById(SUNS_MODEL_ID_MPPT)

	// search controls model
	controlsModel := deviceModels.FindModelById(SUNS_MODEL_ID_CONTROLS)
	var controls *ControlsModelWriter
	if controlsModel != nil {
		var err error
		controls, err = WritableModelControls(controlsModel.ToWritableModel(ctx, client))
		if err != nil {
			return nil, err
		}
	}

	// search storage model
	storageModel := deviceModels.FindModelById(SUNS_MODEL_ID_STORAGE)
	var storage *StorageModelWriter
	if storageModel != nil {
		var err error
		storage, err = WritableModelStorage(storageModel.ToWritableModel(ctx, client))
		if err != nil {
			return nil, err
		}
	}

	// check found models
	if commonModel != nil && inverterModel != nil && statusModel != nil {
		return &DeviceInverter{
			Common:    commonModel.ToReadableModel(ctx, client),
			Inverter:  inverterModel.ToReadableModel(ctx, client),
			Status:    statusModel.ToReadableModel(ctx, client),
			Nameplate: nameplateModel.ToReadableModel(ctx, client),
			Mppt:      mpptModel.ToReadableModel(ctx, client),
			Controls:  controls,
			Storage:   storage,
		}, nil
	} else {
		return nil, ErrAcMeterNotFound
	}
}

func (device *DeviceInverter) ReadCommonModel() (*CommonModelData, error) {
	return ReadModelCommon(device.Common)
}

func (device *DeviceInverter) ReadInverterModel() (*InverterModelData, error) {
	return ReadModelInverter(device.Inverter)
}

func (device *DeviceInverter) ReadStatusModel() (*StatusModelData, error) {
	return ReadModelStatus(device.Status)
}

func (device *DeviceInverter) ReadNameplateModel() (*NameplateModelData, error) {
	return ReadModelNameplate(device.Nameplate)
}

func (device *DeviceInverter) ReadMPPTModel() (*MPPTModelData, error) {
	return ReadModelMPPT(device.Mppt)
}

func (device *DeviceInverter) WritableControlsModel() *ControlsModelWriter {
	return device.Controls
}

func (device *DeviceInverter) WritableStorageModel() *StorageModelWriter {
	return device.Storage
}
