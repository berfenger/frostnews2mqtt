package sunspec

const (
	SUNS_MODEL_LENGTH_INVERTER = 50 * 2 // 50 registers
)

type InverterModelData struct {
	modelData
}

func ReadModelInverter(model ReadableModel) (*InverterModelData, error) {
	if model.id < SUNS_MODEL_ID_INVERTER_MIN || model.id > SUNS_MODEL_ID_INVERTER_MAX {
		return nil, ErrInvalidModelId
	}
	data, err := model.Read()
	if err != nil {
		return nil, err
	}
	if len(data.data) != SUNS_MODEL_LENGTH_INVERTER {
		return nil, ErrInvalidModelLength
	}
	return &InverterModelData{modelData: data}, nil
}

func (data *InverterModelData) CabinetTemperature() float64 {
	temp := data.readRegister(31)
	tempSF := data.readRegister(35)
	return applySF(temp, tempSF)
}

func (data *InverterModelData) OperatingState() InverterStatus {
	return InverterStatus(data.readRegister(36))
}

func (data *InverterModelData) OperatingStateString() string {
	return data.OperatingState().String()
}

func (data *InverterModelData) VendorOperatingState() uint16 {
	return data.readRegister(37)
}

func (data *InverterModelData) Events1() uint32 {
	return data.readUint32(38)
}

func (data *InverterModelData) Events1Parsed() DeviceEvents {
	return InverterSunspecEvent(data.Events1())
}

func (data *InverterModelData) VendorDefinedEvents1() uint32 {
	return data.readUint32(40)
}

func (data *InverterModelData) VendorDefinedEvents2() uint32 {
	return data.readUint32(42)
}

func (data *InverterModelData) VendorDefinedEvents3() uint32 {
	return data.readUint32(44)
}

func (data *InverterModelData) VendorDefinedEvents4() uint32 {
	return data.readUint32(46)
}

func (data *InverterModelData) ACPowerWatt() float64 {
	acpower := data.readRegisters(12, 2)
	return applySFint16(int16(acpower[0]), acpower[1])
}
