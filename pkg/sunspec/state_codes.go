package sunspec

import "fmt"

type StorageChargeStatus uint16

// Sunspec StorageChargeStatus

const (
	StorageChargeStatusOff         StorageChargeStatus = 1
	StorageChargeStatusEmpty       StorageChargeStatus = 2
	StorageChargeStatusDischarging StorageChargeStatus = 3
	StorageChargeStatusCharging    StorageChargeStatus = 4
	StorageChargeStatusFull        StorageChargeStatus = 5
	StorageChargeStatusHolding     StorageChargeStatus = 6
	StorageChargeStatusTest        StorageChargeStatus = 7
)

var storageChargeStatusStrings = map[StorageChargeStatus]string{
	StorageChargeStatusOff:         "off",
	StorageChargeStatusEmpty:       "empty",
	StorageChargeStatusDischarging: "discharging",
	StorageChargeStatusCharging:    "charging",
	StorageChargeStatusFull:        "full",
	StorageChargeStatusHolding:     "holding",
	StorageChargeStatusTest:        "test",
}

func (s StorageChargeStatus) String() string {
	if str, ok := storageChargeStatusStrings[s]; ok {
		return str
	}
	return fmt.Sprintf("unknown(%d)", s)
}

// Sunspec InverterStatus

type StateCode interface {
	String() string
}

type InverterStatus uint16

const (
	InverterStatusOff          InverterStatus = 1
	InverterStatusSleeping     InverterStatus = 2
	InverterStatusStarting     InverterStatus = 3
	InverterStatusMPPT         InverterStatus = 4
	InverterStatusThrottled    InverterStatus = 5
	InverterStatusShuttingDown InverterStatus = 6
	InverterStatusFault        InverterStatus = 7
	InverterStatusStandby      InverterStatus = 8
)

var inverterStatusStrings = map[InverterStatus]string{
	InverterStatusOff:          "off",
	InverterStatusSleeping:     "sleeping",
	InverterStatusStarting:     "starting",
	InverterStatusMPPT:         "mppt_tracking",
	InverterStatusThrottled:    "throttled",
	InverterStatusShuttingDown: "shutting_down",
	InverterStatusFault:        "fault",
	InverterStatusStandby:      "standby",
}

func (s InverterStatus) String() string {
	if str, ok := inverterStatusStrings[s]; ok {
		return str
	}
	return fmt.Sprintf("unknown(%d)", s)
}

// Ensure InverterStatus implements StateCode
var _ StateCode = InverterStatus(0)
