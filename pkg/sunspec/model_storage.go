package sunspec

import (
	"math"

	"github.com/berfenger/frostnews2mqtt/pkg/modbus"
)

const (
	SUNS_MODEL_LENGTH_STORAGE = 24 * 2 // 24 registers
)

type StorageModelData struct {
	modelData
}

func ReadModelStorage(model ReadableModel) (*StorageModelData, error) {
	if model.id != SUNS_MODEL_ID_STORAGE {
		return nil, ErrInvalidModelId
	}
	data, err := model.Read()
	if err != nil {
		return nil, err
	}
	if len(data.data) != SUNS_MODEL_LENGTH_STORAGE {
		return nil, ErrInvalidModelLength
	}
	return &StorageModelData{modelData: data}, nil
}

func (data *StorageModelData) StorageCapacity() int {
	wChaMax := data.readRegister(0)
	wChaMaxSF := data.readRegister(16)
	return int(applySF(wChaMax, wChaMaxSF))
}

func (data *StorageModelData) StateOfCharge() float64 {
	soc := applySF(data.readRegister(6), data.readRegister(20))
	// if state == off, soc = 0
	if data.ChargeStatus() == StorageChargeStatusOff {
		soc = 0
	}
	return soc
}

func (data *StorageModelData) MaxCapacityWatt() uint32 {
	maxCap := applySF(data.readRegister(0), data.readRegister(17))
	return uint32(math.Round(maxCap))
}

func (data *StorageModelData) CurrentCapacityWatt() uint32 {
	return uint32(math.Round(data.StateOfCharge() / 100 * float64(data.MaxCapacityWatt())))
}

func (data *StorageModelData) ChargeStatus() StorageChargeStatus {
	return StorageChargeStatus(data.readRegister(9))
}

func (data *StorageModelData) ChargeStatusString() string {
	return data.ChargeStatus().String()
}

func WritableModelStorage(model WritableModel) (*StorageModelWriter, error) {
	if model.id != SUNS_MODEL_ID_STORAGE {
		return nil, ErrInvalidModelId
	}
	if model.length != SUNS_MODEL_LENGTH_STORAGE/2 {
		return nil, ErrInvalidModelLength
	}
	return &StorageModelWriter{WritableModel: model}, nil
}

type StorageModelWriter struct {
	WritableModel
}

func (storage *StorageModelWriter) SetRawStorageChargeControl(maxDischargeRatePercent float64, maxChargeRatePercent float64, controlDischarge bool, controlCharge bool, rvrtTimeSeconds int32) error {

	inoutSF, err := storage.client.ReadRegister(storage.baseAddr+23, modbus.HOLDING_REGISTER)
	if err != nil {
		return err
	}
	control := uint16(0)
	if controlDischarge {
		control = control | 0x02
	}
	if controlCharge {
		control = control | 0x01
	}

	outWRte := int16(applySFfloat64Inv(maxDischargeRatePercent, inoutSF))
	inWRte := int16(applySFfloat64Inv(maxChargeRatePercent, inoutSF))

	err = storage.WriteRegisters(10, []uint16{uint16(outWRte), uint16(inWRte)})
	if err != nil {
		return err
	}
	err = storage.WriteRegister(3, control)
	if err != nil {
		return err
	}
	if rvrtTimeSeconds >= 0 {
		err = storage.WriteRegister(13, uint16(rvrtTimeSeconds))
		if err != nil {
			return err
		}
	}
	return nil
}

func (storage *StorageModelWriter) SetStorageControl(
	minChargePowerWatt int32,
	maxChargePowerWatt int32,
	minDischargePowerWatt int32,
	maxDischargePowerWatt int32,
	revertTimeSeconds uint32) error {

	data, err := storage.Read()
	if err != nil {
		return err
	}

	rawCapacity := data.StorageCapacity()
	capacity := float64(rawCapacity)

	var outWRte float64 = 100
	var inWRte float64 = 100
	controlOut := false
	controlIn := false
	rvrtTime := int32(revertTimeSeconds)
	if minChargePowerWatt >= 0 {
		outWRte = -(float64(minChargePowerWatt) / capacity) * 100
		controlOut = true
	}
	if maxChargePowerWatt >= 0 {
		inWRte = (float64(maxChargePowerWatt) / capacity) * 100
		controlIn = true
	}
	if minDischargePowerWatt >= 0 {
		inWRte = -(float64(minDischargePowerWatt) / capacity) * 100
		controlIn = true
	}
	if maxDischargePowerWatt >= 0 {
		outWRte = (float64(maxDischargePowerWatt) / capacity) * 100
		controlOut = true
	}

	return storage.SetRawStorageChargeControl(outWRte, inWRte, controlOut, controlIn, rvrtTime)
}

func (storage *StorageModelWriter) DisableStorageControl() error {
	return storage.SetRawStorageChargeControl(100, 100, false, false, -1)
}

func (controls *StorageModelWriter) Read() (*StorageModelData, error) {
	return ReadModelStorage(controls.ReadableModel)
}
