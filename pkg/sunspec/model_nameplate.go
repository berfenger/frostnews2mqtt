package sunspec

const (
	SUNS_MODEL_LENGTH_NAMEPLATE = 26 * 2 // 26 registers
)

type NameplateModelData struct {
	modelData
}

func ReadModelNameplate(model ReadableModel) (*NameplateModelData, error) {
	if model.id != SUNS_MODEL_ID_NAMEPLATE {
		return nil, ErrInvalidModelId
	}
	data, err := model.Read()
	if err != nil {
		return nil, err
	}
	if len(data.data) != SUNS_MODEL_LENGTH_NAMEPLATE {
		return nil, ErrInvalidModelLength
	}
	return &NameplateModelData{modelData: data}, nil
}

func (data *NameplateModelData) MaxRatedACPowerWatt() uint32 {
	regs := data.readRegisters(1, 2)
	return uint32(applySF(regs[0], regs[1]))
}
