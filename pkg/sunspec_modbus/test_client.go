package sunspec_modbus

func CreateTestACMeterModbusReader() (ACMeterModbusReader, error) {
	return TestACMeterModbusReader{}, nil
}

func CreateTestInverterModbusReader() (InverterModbusReader, error) {

	return TestInverterModbusReader{}, nil
}

// ACMeter

type TestACMeterModbusReader struct {
}

func (reader TestACMeterModbusReader) Open() error {
	return nil
}

func (reader TestACMeterModbusReader) Close() error {
	return nil
}

func (reader TestACMeterModbusReader) Validate() error {
	return nil
}

func (reader TestACMeterModbusReader) GetInfo() (*ACMeterInfo, error) {
	return &ACMeterInfo{
		Manufacturer: "Frostnews",
		Model:        "Smart Meter TS 100A-1",
		Version:      "1.2",
	}, nil
}

func (reader TestACMeterModbusReader) GetCurrentPowerFlowWatt() (float64, error) {
	return -1250, nil
}

func (reader TestACMeterModbusReader) GetPowerFlow() (*ACMeterPowerFlow, error) {

	return &ACMeterPowerFlow{
		CurrentPowerFlowWatt:   -1250,
		CurrentImportPowerWatt: 0,
		CurrentExportPowerWatt: 1250,
		TotalEnergyExportedKWh: 2770.34,
		TotalEnergyImportedKWh: 550.22,
		Frequency:              50,
		PhaseAVoltage:          234.24,
	}, nil
}

// Inverter

type TestInverterModbusReader struct {
}

func (inv TestInverterModbusReader) Open() error {
	return nil
}

func (inv TestInverterModbusReader) Close() error {
	return nil
}

func (inv TestInverterModbusReader) Validate() error {
	return nil
}

func (inv TestInverterModbusReader) GetInfo() (*InverterInfo, error) {
	return &InverterInfo{
		Manufacturer:      "Frostnews",
		Model:             "Primo GEN24 4.0",
		Version:           "1.30.7-1",
		MaxRatedPowerWatt: 4000,
	}, nil
}
func (inv TestInverterModbusReader) GetState() (*InverterState, error) {

	return &InverterState{
		CabinetTemperature: 51.7,
		OperatingState:     4,
		OperatingStateStr:  InverterStatusToString(4),
	}, nil
}

func (inv TestInverterModbusReader) SetPowerLimit(powerLimit InverterPowerLimit) error {
	return nil
}
func (inv TestInverterModbusReader) GetPowerLimit() (*InverterPowerLimit, error) {
	return &InverterPowerLimit{
		Enabled:           false,
		Percent:           100,
		RevertTimeSeconds: 30,
	}, nil
}

func (inv TestInverterModbusReader) GetPowerFlow() (*InverterPowerFlow, error) {
	return &InverterPowerFlow{
		ACPowerWatt:               320.2,
		PVPowerWatt:               920.3,
		BatteryChargePowerWatt:    572.45,
		BatteryDischargePowerWatt: 0,
		BatteryDCPowerFlowWatt:    -572.45,
	}, nil
}

func (inv TestInverterModbusReader) SupportsPowerControl() (bool, error) {
	return true, nil
}

func (inv TestInverterModbusReader) HasStorage() (bool, error) {
	return true, nil
}

func (inv TestInverterModbusReader) SetStorageChargeControl(maxDischargeRatePercent float64, maxChargeRatePercent float64, controlDischarge bool, controlCharge bool, rvrtTimeSeconds int32) error {
	return nil
}

func (inv TestInverterModbusReader) SetStorageControl(params StorageControlParams) error {
	return nil
}

func (inv TestInverterModbusReader) DisableStorageControl() error {
	return inv.SetStorageChargeControl(100, 100, false, false, -1)
}

func (inv TestInverterModbusReader) SetStorageForceChargePower(watts uint16, revertTimeSeconds int32) error {

	return inv.SetStorageControl(StorageControlParams{
		MinChargePowerWatt:    int32(watts),
		MaxChargePowerWatt:    -1,
		MinDischargePowerWatt: -1,
		MaxDischargePowerWatt: -1,
		RevertTimeSeconds:     uint32(revertTimeSeconds),
	})
}

func (inv TestInverterModbusReader) SetStorageForceDischargePower(watts uint16, revertTimeSeconds int32) error {

	return inv.SetStorageControl(StorageControlParams{
		MinChargePowerWatt:    -1,
		MaxChargePowerWatt:    -1,
		MinDischargePowerWatt: int32(watts),
		MaxDischargePowerWatt: -1,
		RevertTimeSeconds:     uint32(revertTimeSeconds),
	})
}

func (inv TestInverterModbusReader) GetStorageState() (*StorageState, error) {
	return &StorageState{
		StateOfCharge:       23.5,
		MaxCapacityWatt:     5260,
		CurrentCapacityWatt: 1236,
		ChargeStatus:        StorageChargeStatusCharging,
		ChargeStatusStr:     StorageChargeStatusToString(StorageChargeStatusCharging),
	}, nil
}
