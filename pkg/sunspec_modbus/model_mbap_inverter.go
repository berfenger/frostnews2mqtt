package sunspec_modbus

import (
	"fmt"
)

// storage states
const (
	StorageChargeStatusOff         = 1
	StorageChargeStatusEmpty       = 2
	StorageChargeStatusDischarging = 3
	StorageChargeStatusCharging    = 4
	StorageChargeStatusFull        = 5
	StorageChargeStatusHolding     = 6
	StorageChargeStatusTest        = 7
)

// storage state strings
const (
	StorageChargeStatusOffStr         = "off"
	StorageChargeStatusEmptyStr       = "empty"
	StorageChargeStatusDischargingStr = "discharging"
	StorageChargeStatusChargingStr    = "charging"
	StorageChargeStatusFullStr        = "full"
	StorageChargeStatusHoldingStr     = "holding"
	StorageChargeStatusTestStr        = "test"
	StorageChargeStatusUnknownStr     = "unknown"
)

func StorageChargeStatusToString(storage uint16) string {
	switch storage {
	case StorageChargeStatusOff:
		return StorageChargeStatusOffStr
	case StorageChargeStatusEmpty:
		return StorageChargeStatusEmptyStr
	case StorageChargeStatusDischarging:
		return StorageChargeStatusDischargingStr
	case StorageChargeStatusCharging:
		return StorageChargeStatusChargingStr
	case StorageChargeStatusFull:
		return StorageChargeStatusFullStr
	case StorageChargeStatusHolding:
		return StorageChargeStatusHoldingStr
	case StorageChargeStatusTest:
		return StorageChargeStatusTestStr
	default:
		return fmt.Sprintf("%s(%d)", StorageChargeStatusUnknownStr, storage)
	}
}

const (
	InverterStatusOff          = 1
	InverterStatusSleeping     = 2
	InverterStatusStarting     = 3
	InverterStatusMPPT         = 4
	InverterStatusThrottled    = 5
	InverterStatusShuttingDown = 6
	InverterStatusFault        = 7
	InverterStatusStandby      = 8
)

const (
	InverterStatusOffStr          = "off"
	InverterStatusSleepingStr     = "sleeping"
	InverterStatusStartingStr     = "starting"
	InverterStatusMPPTStr         = "mppt_tracking"
	InverterStatusThrottledStr    = "throttled"
	InverterStatusShuttingDownStr = "shutting_down"
	InverterStatusFaultStr        = "fault"
	InverterStatusStandbyStr      = "standby"
	InverterStatusUnknown         = "unknown"
)

func InverterStatusToString(state uint16) string {
	switch state {
	case InverterStatusOff:
		return InverterStatusOffStr
	case InverterStatusSleeping:
		return InverterStatusSleepingStr
	case InverterStatusStarting:
		return InverterStatusStartingStr
	case InverterStatusMPPT:
		return InverterStatusMPPTStr
	case InverterStatusThrottled:
		return InverterStatusThrottledStr
	case InverterStatusShuttingDown:
		return InverterStatusShuttingDownStr
	case InverterStatusFault:
		return InverterStatusFaultStr
	case InverterStatusStandby:
		return InverterStatusStandbyStr
	default:
		return fmt.Sprintf("%s(%d)", InverterStatusUnknown, state)
	}
}

type InverterInfo struct {
	Manufacturer      string
	Model             string
	Version           string
	Serial            string
	MaxRatedPowerWatt uint32
	HasStorage        bool
}

type InverterState struct {
	CabinetTemperature float64
	OperatingState     uint16
	OperatingStateStr  string
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
	ChargeStatus        uint16
	ChargeStatusStr     string
}

type StorageControlParams struct {
	MinChargePowerWatt    int32
	MaxChargePowerWatt    int32
	MinDischargePowerWatt int32
	MaxDischargePowerWatt int32
	RevertTimeSeconds     uint32
}

type InverterModbusReader interface {
	Open() error
	Close() error
	Validate() error
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
