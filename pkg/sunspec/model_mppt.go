package sunspec

const (
	SUNS_MODEL_LENGTH_MPPT        = 8 * 2  // base 8 registers
	SUNS_MODEL_LENGTH_MPPT_MODULE = 20 * 2 // registers per MPPT module
)

type MPPTModelData struct {
	modelData
}

type MPPTModuleData struct {
	Id        uint16
	IdString  string
	DCCurrent float64
	DCVoltage float64
	DCPower   float64
}

func ReadModelMPPT(model ReadableModel) (*MPPTModelData, error) {
	if model.id != SUNS_MODEL_ID_MPPT {
		return nil, ErrInvalidModelId
	}
	data, err := model.Read()
	if err != nil {
		return nil, err
	}
	if (len(data.data)-SUNS_MODEL_LENGTH_MPPT)%SUNS_MODEL_LENGTH_MPPT_MODULE != 0 {
		return nil, ErrInvalidModelLength
	}
	return &MPPTModelData{modelData: data}, nil
}

func (data *MPPTModelData) MPPTModulesData() []MPPTModuleData {
	modules := []MPPTModuleData{}
	sf := data.readRegisters(0, 3)
	numModules := int(data.readRegister(6))
	for i := range numModules {
		baseAddr := uint16(8 + 20*uint16(i))
		dcPower := applySF(data.readRegister(baseAddr+11), sf[2])
		if int16(dcPower) == -1 {
			dcPower = 0
		}
		module := MPPTModuleData{
			Id:        data.readRegister(baseAddr),
			IdString:  data.readString(baseAddr+1, 16),
			DCCurrent: applySF(data.readRegister(baseAddr+9), sf[0]),
			DCVoltage: applySF(data.readRegister(baseAddr+10), sf[1]),
			DCPower:   dcPower,
		}
		modules = append(modules, module)
	}
	return modules
}
