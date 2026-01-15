package sunspec

import "maps"

// model
type DeviceEvents interface {
	Flags() map[string]bool
}

// Standard InverterSunspecEvent
type InverterSunspecEvent uint32

const (
	EvtGndFault         InverterSunspecEvent = 1 << 0  // Ground fault
	EvtDCOverVolt       InverterSunspecEvent = 1 << 1  // DC over voltage
	EvtACDisconnectOpen InverterSunspecEvent = 1 << 2  // AC disconnect open
	EvtDCDisconnectOpen InverterSunspecEvent = 1 << 3  // DC disconnect open
	EvtGridShutdown     InverterSunspecEvent = 1 << 4  // Grid shutdown
	EvtCabinetOpen      InverterSunspecEvent = 1 << 5  // Cabinet open
	EvtManualShutdown   InverterSunspecEvent = 1 << 6  // Manual shutdown
	EvtOverTemp         InverterSunspecEvent = 1 << 7  // Over temperature
	EvtOverFreq         InverterSunspecEvent = 1 << 8  // Frequency above limit
	EvtUnderFreq        InverterSunspecEvent = 1 << 9  // Frequency under limit
	EvtACOverVolt       InverterSunspecEvent = 1 << 10 // AC voltage above limit
	EvtACUnderVolt      InverterSunspecEvent = 1 << 11 // AC voltage under limit
	EvtBlownFuse        InverterSunspecEvent = 1 << 12 // Blown string fuse
	EvtUnderTemp        InverterSunspecEvent = 1 << 13 // Under temperature
	EvtMemoryCommError  InverterSunspecEvent = 1 << 14 // Memory or communication error
	EvtHWTestFailure    InverterSunspecEvent = 1 << 15 // Hardware test failure
)

var inverterStandardEventStrings = map[InverterSunspecEvent]string{
	EvtGndFault:         "ground_fault",
	EvtDCOverVolt:       "dc_over_voltage",
	EvtACDisconnectOpen: "ac_disconnect_open",
	EvtDCDisconnectOpen: "dc_disconnect_open",
	EvtGridShutdown:     "grid_shutdown",
	EvtCabinetOpen:      "cabinet_open",
	EvtManualShutdown:   "manual_shutdown",
	EvtOverTemp:         "over_temperature",
	EvtOverFreq:         "frequency_above_limit",
	EvtUnderFreq:        "frequency_below_limit",
	EvtACOverVolt:       "ac_over_voltage",
	EvtACUnderVolt:      "ac_under_voltage",
	EvtBlownFuse:        "blown_fuse",
	EvtUnderTemp:        "under_temperature",
	EvtMemoryCommError:  "memory_or_communication_error",
	EvtHWTestFailure:    "hardware_test_failure",
}

func InverterStandardEventsToMap(events uint32) map[string]bool {
	eventMap := make(map[string]bool)
	for event, name := range inverterStandardEventStrings {
		if (events & uint32(event)) != 0 {
			eventMap[name] = true
		} else {
			eventMap[name] = false
		}
	}
	return eventMap
}

func (e InverterSunspecEvent) Flags() map[string]bool {
	return InverterStandardEventsToMap(uint32(e))
}

// Ensure InverterSunspecEvent implements DeviceEvents
var _ DeviceEvents = InverterSunspecEvent(0)

type CompositeDeviceEvents struct {
	EventList []DeviceEvents
}

func (c CompositeDeviceEvents) Flags() map[string]bool {
	flags := make(map[string]bool)
	for _, devEvt := range c.EventList {
		maps.Copy(flags, devEvt.Flags())
	}
	return flags
}

// Ensure CompositeDeviceEvents implements DeviceEvents
var _ DeviceEvents = CompositeDeviceEvents{}
