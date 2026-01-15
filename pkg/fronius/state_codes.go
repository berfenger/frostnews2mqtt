package fronius

import "fmt"

type FroniusInverterStatus uint16

const (
	FroniusInverterStatusOff               FroniusInverterStatus = 1
	FroniusInverterStatusSleeping          FroniusInverterStatus = 2
	FroniusInverterStatusStarting          FroniusInverterStatus = 3
	FroniusInverterStatusMPPT              FroniusInverterStatus = 4
	FroniusInverterStatusThrottled         FroniusInverterStatus = 5
	FroniusInverterStatusShuttingDown      FroniusInverterStatus = 6
	FroniusInverterStatusFault             FroniusInverterStatus = 7
	FroniusInverterStatusStandby           FroniusInverterStatus = 8
	FroniusInverterStatusNoBusInit         FroniusInverterStatus = 9
	FroniusInverterStatusNoCommInv         FroniusInverterStatus = 10
	FroniusInverterStatusSNPlugOvercurrent FroniusInverterStatus = 11
	FroniusInverterStatusBootload          FroniusInverterStatus = 12
	FroniusInverterStatusAFCI              FroniusInverterStatus = 13
)

var FroniusInverterStatusStrings = map[FroniusInverterStatus]string{
	FroniusInverterStatusOff:               "off",
	FroniusInverterStatusSleeping:          "sleeping",
	FroniusInverterStatusStarting:          "starting",
	FroniusInverterStatusMPPT:              "mppt_tracking",
	FroniusInverterStatusThrottled:         "throttled",
	FroniusInverterStatusShuttingDown:      "shutting_down",
	FroniusInverterStatusFault:             "fault",
	FroniusInverterStatusStandby:           "standby",
	FroniusInverterStatusNoBusInit:         "no_solar_net_communication",
	FroniusInverterStatusNoCommInv:         "no_communication_with_inverter",
	FroniusInverterStatusSNPlugOvercurrent: "solar_net_plug_overcurrent",
	FroniusInverterStatusBootload:          "update_in_progress",
	FroniusInverterStatusAFCI:              "afci",
}

func (s FroniusInverterStatus) String() string {
	if str, ok := FroniusInverterStatusStrings[s]; ok {
		return str
	}
	return fmt.Sprintf("unknown(%d)", s)
}
