package sunspec

import (
	"github.com/berfenger/frostnews2mqtt/pkg/modbus"
)

const (
	SUNS_MODEL_LENGTH_CONTROLS = 24 * 2 // 24 registers
)

type ControlsModelData struct {
	modelData
}

func ReadModelControls(model ReadableModel) (*ControlsModelData, error) {
	if model.id != SUNS_MODEL_ID_CONTROLS {
		return nil, ErrInvalidModelId
	}
	data, err := model.Read()
	if err != nil {
		return nil, err
	}
	if len(data.data) != SUNS_MODEL_LENGTH_CONTROLS {
		return nil, ErrInvalidModelLength
	}
	return &ControlsModelData{modelData: data}, nil
}

func (data *ControlsModelData) PowerLimitEnabled() bool {
	return data.readRegister(7) == 1
}

func (data *ControlsModelData) PowerLimitPercent() float64 {
	sf := data.readRegister(21)
	return applySF(data.readRegister(3), sf)
}

func (data *ControlsModelData) RevertTimeSeconds() uint32 {
	return uint32(data.readRegister(5))
}

func WritableModelControls(model WritableModel) (*ControlsModelWriter, error) {
	if model.id != SUNS_MODEL_ID_CONTROLS {
		return nil, ErrInvalidModelId
	}
	if model.length != SUNS_MODEL_LENGTH_CONTROLS/2 {
		return nil, ErrInvalidModelLength
	}
	return &ControlsModelWriter{WritableModel: model}, nil
}

type ControlsModelWriter struct {
	WritableModel
}

func (controls *ControlsModelWriter) SetPowerLimit(enable bool, percent float64, revertTimeSeconds uint32) error {
	// write 0 to WMaxLim_Ena. A new value won't be accepted without this step
	err := controls.WriteRegister(7, uint16(0))
	if err != nil {
		return err
	}
	if enable {
		// get scale factor to write percent
		sf, err := controls.client.ReadRegister(controls.baseAddr+21, modbus.HOLDING_REGISTER)
		if err != nil {
			return err
		}
		c := uint16(1)
		// build and write data array [WMaxLimPct, WMaxLimPct_WinTms, WMaxLimPct_RvrtTms, WMaxLimPct_RmpTms, WMaxLim_Ena]
		data := []uint16{uint16(applySFInv(uint16(percent), sf)), 0, uint16(revertTimeSeconds), 0, c}
		return controls.WriteRegisters(3, data)
	} else {
		return err
	}
}

func (controls *ControlsModelWriter) Read() (*ControlsModelData, error) {
	return ReadModelControls(controls.ReadableModel)
}
