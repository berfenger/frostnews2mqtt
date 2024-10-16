package sunspec_modbus

type ACMeterInfo struct {
	Manufacturer string
	Model        string
	Version      string
	Serial       string
}

type ACMeterPowerFlow struct {
	// Current AC power flow. Positive = import. Negative = export
	CurrentPowerFlowWatt float64
	// Current import AC power
	CurrentImportPowerWatt float64
	// Current export AC power
	CurrentExportPowerWatt float64
	// Lifetime exported energy in kWh
	TotalEnergyExportedKWh float64
	// Lifetime imported energy in kWh
	TotalEnergyImportedKWh float64
	// Grid frequency
	Frequency float64
	// First grid phase voltage
	PhaseAVoltage float64
}

type ACMeterModbusReader interface {
	Open() error
	Close() error
	Validate() error
	GetInfo() (*ACMeterInfo, error)
	GetCurrentPowerFlowWatt() (float64, error)
	GetPowerFlow() (*ACMeterPowerFlow, error)
}
