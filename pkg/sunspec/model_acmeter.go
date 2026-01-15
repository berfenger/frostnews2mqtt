package sunspec

const (
	SUNS_MODEL_LENGTH_ACMETER = 105 * 2 // 105 registers
)

type ACMeterModelData struct {
	modelData
}

func ReadModelACMeter(model ReadableModel) (*ACMeterModelData, error) {
	if model.id < SUNS_MODEL_ID_ACMETER_MIN || model.id > SUNS_MODEL_ID_ACMETER_MAX {
		return nil, ErrInvalidModelId
	}
	data, err := model.Read()
	if err != nil {
		return nil, err
	}
	if len(data.data) != SUNS_MODEL_LENGTH_ACMETER {
		return nil, ErrInvalidModelLength
	}
	return &ACMeterModelData{modelData: data}, nil
}

func (data *ACMeterModelData) GetCurrentPowerFlowWatt() float64 {
	totalRealPower := data.readRegister(16)
	totalRealPowerSF := data.readRegister(20)

	return applySFint16(int16(totalRealPower), totalRealPowerSF)
}

func (data *ACMeterModelData) GetGridFrequency() float64 {

	freq := data.readRegisters(14, 2)
	return applySF(freq[0], freq[1])
}

func (data *ACMeterModelData) GetTotalEnergyExported() float64 {

	rawTotalEnergyExported := data.readUint32(36)
	totWh_SF := data.readRegister(52)
	return applySFuint32(rawTotalEnergyExported, totWh_SF) / 1000
}

func (data *ACMeterModelData) GetTotalEnergyImported() float64 {

	rawTotalEnergyImported := data.readUint32(44)
	totWh_SF := data.readRegister(52)
	return applySFuint32(rawTotalEnergyImported, totWh_SF) / 1000
}

func (data *ACMeterModelData) GetPhaseAVoltage() float64 {
	phaseAVoltage := data.readRegister(6)
	phaseAVoltage_SF := data.readRegister(13)
	return applySF(phaseAVoltage, phaseAVoltage_SF)
}

func (data *ACMeterModelData) GetPhaseBVoltage() float64 {
	phaseAVoltage := data.readRegister(7)
	phaseAVoltage_SF := data.readRegister(13)
	return applySF(phaseAVoltage, phaseAVoltage_SF)
}

func (data *ACMeterModelData) GetPhaseCVoltage() float64 {
	phaseAVoltage := data.readRegister(8)
	phaseAVoltage_SF := data.readRegister(13)
	return applySF(phaseAVoltage, phaseAVoltage_SF)
}
