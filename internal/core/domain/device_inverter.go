package domain

import "github.com/berfenger/frostnews2mqtt/pkg/sunspec"

type InverterInfo struct {
	Manufacturer      string
	Model             string
	Version           string
	Serial            string
	MaxRatedPowerWatt uint32
	HasStorage        bool
	HasVendorProfile  bool
}

type InverterState struct {
	CabinetTemperature   float64
	OperatingState       sunspec.InverterStatus
	OperatingStateStr    string
	VendorOperatingState uint16
	SunspecEvent1        uint32
	SunspecDeviceEvents  sunspec.DeviceEvents
	SunspecVendorEvent1  uint32
	SunspecVendorEvent2  uint32
	SunspecVendorEvent3  uint32
	SunspecVendorEvent4  uint32
}

type VendorInverterState struct {
	VendorOperatingStateStr string
	VendorDeviceEvents      sunspec.DeviceEvents
}

type InverterPowerFlow struct {
	ACPowerWatt               float64
	PVPowerWatt               float64
	BatteryChargePowerWatt    float64
	BatteryDischargePowerWatt float64
	BatteryDCPowerFlowWatt    float64
}

type InverterPowerLimit struct {
	Enabled           bool
	Percent           float64
	RevertTimeSeconds uint32
}

type StorageState struct {
	StateOfCharge       float64
	MaxCapacityWatt     uint32
	CurrentCapacityWatt uint32
	ChargeStatus        sunspec.StorageChargeStatus
	ChargeStatusStr     string
}

type StorageControlParams struct {
	MinChargePowerWatt    int32
	MaxChargePowerWatt    int32
	MinDischargePowerWatt int32
	MaxDischargePowerWatt int32
	RevertTimeSeconds     uint32
}

type InverterDevice interface {
	Open() error
	Close() error
	GetInfo() (*InverterInfo, error)
	GetState() (*InverterState, error)
	GetPowerFlow() (*InverterPowerFlow, error)

	SetPowerLimit(powerLimit InverterPowerLimit) error
	GetPowerLimit() (*InverterPowerLimit, error)

	HasStorage() (bool, error)
	SupportsPowerControl() (bool, error)
	SetStorageControl(params StorageControlParams) error
	SetStorageForceChargePower(watts uint16, revertTimeSeconds int32) error
	SetStorageForceDischargePower(watts uint16, revertTimeSeconds int32) error
	DisableStorageControl() error
	GetStorageState() (*StorageState, error)
}

type VendorInverterDevice interface {
	InverterDevice
	GetVendorState() (*VendorInverterState, error)
}
