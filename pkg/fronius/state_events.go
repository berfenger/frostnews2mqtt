package fronius

import "github.com/berfenger/frostnews2mqtt/pkg/sunspec"

type FroniusModel int

const (
	Primo FroniusModel = iota
	Symo
	Galvo
	IGPlus
	UnknownModel
)

// InverterFroniusEvent1 - EvtVnd1 events
type InverterFroniusEvent1 uint32

const (
	Evt1GridError                     InverterFroniusEvent1 = 0x2        // Grid error
	Evt1OvercurrentAC                 InverterFroniusEvent1 = 0x4        // Overcurrent AC
	Evt1OvercurrentDC                 InverterFroniusEvent1 = 0x8        // Overcurrent DC
	Evt1OverTemperature               InverterFroniusEvent1 = 0x10       // Over-temperature
	Evt1PowerLow                      InverterFroniusEvent1 = 0x20       // Power low
	Evt1DCLow                         InverterFroniusEvent1 = 0x40       // DC low
	Evt1IntermediateCircuitError      InverterFroniusEvent1 = 0x80       // Intermediate circuit error
	Evt1ACFrequencyTooHigh            InverterFroniusEvent1 = 0x100      // AC frequency too high
	Evt1ACFrequencyTooLow             InverterFroniusEvent1 = 0x200      // AC frequency too low
	Evt1ACVoltageTooHigh              InverterFroniusEvent1 = 0x400      // AC voltage too high
	Evt1ACVoltageTooLow               InverterFroniusEvent1 = 0x800      // AC voltage too low
	Evt1DirectCurrentFeedIn           InverterFroniusEvent1 = 0x1000     // Direct current feed in
	Evt1RelayProblem                  InverterFroniusEvent1 = 0x2000     // Relay problem
	Evt1InternalPowerStageError       InverterFroniusEvent1 = 0x4000     // Internal power stage error
	Evt1ControlProblems               InverterFroniusEvent1 = 0x8000     // Control problems
	Evt1GuardControllerACVoltageError InverterFroniusEvent1 = 0x10000    // Guard Controller - AC voltage error
	Evt1GuardControllerACFreqError    InverterFroniusEvent1 = 0x20000    // Guard Controller - AC Frequency Error
	Evt1EnergyTransferNotPossible     InverterFroniusEvent1 = 0x40000    // Energy transfer not possible
	Evt1RefPowerSourceACOutOfTol      InverterFroniusEvent1 = 0x80000    // Reference power source AC outside tolerances
	Evt1ErrorAntiIslandingTest        InverterFroniusEvent1 = 0x100000   // Error during anti islanding test
	Evt1FixedVoltageLowerThanMPP      InverterFroniusEvent1 = 0x200000   // Fixed voltage lower than current MPP voltage
	Evt1MemoryFault                   InverterFroniusEvent1 = 0x400000   // Memory fault
	Evt1Display                       InverterFroniusEvent1 = 0x800000   // Display
	Evt1InternalCommError             InverterFroniusEvent1 = 0x1000000  // Internal communication error
	Evt1TempSensorsDefective          InverterFroniusEvent1 = 0x2000000  // Temperature sensors defective
	Evt1DCOrACBoardFault              InverterFroniusEvent1 = 0x4000000  // DC or AC board fault
	Evt1ENSError                      InverterFroniusEvent1 = 0x8000000  // ENS error
	Evt1FanError                      InverterFroniusEvent1 = 0x10000000 // Fan error
	Evt1DefectiveFuse                 InverterFroniusEvent1 = 0x20000000 // Defective fuse
	Evt1OutputChokeWrongPoles         InverterFroniusEvent1 = 0x40000000 // Output choke connected to wrong poles
	Evt1BuckConverterRelayNoOpen      InverterFroniusEvent1 = 0x80000000 // The buck converter relay does not open at high DC voltage
)

var inverterEvent1Strings = map[InverterFroniusEvent1]string{
	Evt1GridError:                     "grid_error",
	Evt1OvercurrentAC:                 "overcurrent_ac",
	Evt1OvercurrentDC:                 "overcurrent_dc",
	Evt1OverTemperature:               "over_temperature",
	Evt1PowerLow:                      "power_low",
	Evt1DCLow:                         "dc_low",
	Evt1IntermediateCircuitError:      "intermediate_circuit_error",
	Evt1ACFrequencyTooHigh:            "ac_frequency_too_high",
	Evt1ACFrequencyTooLow:             "ac_frequency_too_low",
	Evt1ACVoltageTooHigh:              "ac_voltage_too_high",
	Evt1ACVoltageTooLow:               "ac_voltage_too_low",
	Evt1DirectCurrentFeedIn:           "direct_current_feed_in",
	Evt1RelayProblem:                  "relay_problem",
	Evt1InternalPowerStageError:       "internal_power_stage_error",
	Evt1ControlProblems:               "control_problems",
	Evt1GuardControllerACVoltageError: "guard_controller_ac_voltage_error",
	Evt1GuardControllerACFreqError:    "guard_controller_ac_frequency_error",
	Evt1EnergyTransferNotPossible:     "energy_transfer_not_possible",
	Evt1RefPowerSourceACOutOfTol:      "ref_power_source_ac_outside_tolerances",
	Evt1ErrorAntiIslandingTest:        "error_during_anti_islanding_test",
	Evt1FixedVoltageLowerThanMPP:      "fixed_voltage_lower_than_current_mpp_voltage",
	Evt1MemoryFault:                   "memory_fault",
	Evt1Display:                       "display",
	Evt1InternalCommError:             "internal_communication_error",
	Evt1TempSensorsDefective:          "temperature_sensors_defective",
	Evt1DCOrACBoardFault:              "dc_or_ac_board_fault",
	Evt1ENSError:                      "ens_error",
	Evt1FanError:                      "fan_error",
	Evt1DefectiveFuse:                 "defective_fuse",
	Evt1OutputChokeWrongPoles:         "output_choke_connected_to_wrong_poles",
	Evt1BuckConverterRelayNoOpen:      "buck_converter_relay_does_not_open_at_high_dc_voltage",
}

// Event masks per model - defines which events are supported by each Fronius model
var event1SupportedMasks = map[FroniusModel]uint32{
	Primo:        0x0F777FFE,
	Symo:         0x2F737FFE,
	Galvo:        0x2F633FFE,
	IGPlus:       0xeFFFFFFE,
	UnknownModel: 0xFFFFFFFF, // All events supported by default
}

func FroniusInverterEvent1ToMap(events uint32, supportedMask uint32) map[string]bool {
	eventMap := make(map[string]bool)
	for event, name := range inverterEvent1Strings {
		// Only include events that are in the supported mask
		if (supportedMask & uint32(event)) != 0 {
			if (events & uint32(event)) != 0 {
				eventMap[name] = true
			} else {
				eventMap[name] = false
			}
		}
	}
	return eventMap
}

type InverterFroniusEvent1ForMask struct {
	event InverterFroniusEvent1
	mask  uint32
}

func (e InverterFroniusEvent1ForMask) Flags() map[string]bool {
	return FroniusInverterEvent1ToMap(uint32(e.event), e.mask)
}

// Ensure InverterFroniusEvent1 implements DeviceEvents
var _ sunspec.DeviceEvents = InverterFroniusEvent1ForMask{}

// InverterFroniusEvent2 - EvtVnd2 events
type InverterFroniusEvent2 uint32

const (
	Evt2NoSolarNetComm                InverterFroniusEvent2 = 0x1        // No SolarNet communication
	Evt2InverterAddressIncorrect      InverterFroniusEvent2 = 0x2        // Inverter address incorrect
	Evt224hNoFeedIn                   InverterFroniusEvent2 = 0x4        // 24h no feed in
	Evt2FaultyPlugConnections         InverterFroniusEvent2 = 0x8        // Faulty plug connections
	Evt2IncorrectPhaseAllocation      InverterFroniusEvent2 = 0x10       // Incorrect phase allocation
	Evt2GridConductorOpenPhaseFailure InverterFroniusEvent2 = 0x20       // Grid conductor open or supply phase has failed
	Evt2IncompatibleOldSoftware       InverterFroniusEvent2 = 0x40       // Incompatible or old software
	Evt2PowerDeratingOvertemp         InverterFroniusEvent2 = 0x80       // Power Derating Due To Overtemperature
	Evt2JumperSetIncorrectly          InverterFroniusEvent2 = 0x100      // Jumper set incorrectly
	Evt2IncompatibleFeature           InverterFroniusEvent2 = 0x200      // Incompatible feature
	Evt2DefectiveVentilatorAirVents   InverterFroniusEvent2 = 0x400      // Defective ventilator/air vents blocked
	Evt2PowerReductionOnError         InverterFroniusEvent2 = 0x800      // Power reduction on error
	Evt2ArcDetected                   InverterFroniusEvent2 = 0x1000     // Arc Detected
	Evt2AFCISelfTestFailed            InverterFroniusEvent2 = 0x2000     // AFCI Self Test Failed
	Evt2CurrentSensorError            InverterFroniusEvent2 = 0x4000     // Current Sensor Error
	Evt2DCSwitchFault                 InverterFroniusEvent2 = 0x8000     // DC switch fault
	Evt2AFCIDefective                 InverterFroniusEvent2 = 0x10000    // AFCI Defective
	Evt2AFCIManualTestSuccessful      InverterFroniusEvent2 = 0x20000    // AFCI Manual Test Successful
	Evt2PowerStackSupplyMissing       InverterFroniusEvent2 = 0x40000    // Power Stack Supply Missing
	Evt2AFCICommStopped               InverterFroniusEvent2 = 0x80000    // AFCI Communication Stopped
	Evt2AFCIManualTestFailed          InverterFroniusEvent2 = 0x100000   // AFCI Manual Test Failed
	Evt2ACPolarityReversed            InverterFroniusEvent2 = 0x200000   // AC polarity reversed
	Evt2ACMeasurementDeviceFault      InverterFroniusEvent2 = 0x400000   // AC measurement device fault
	Evt2FlashFault                    InverterFroniusEvent2 = 0x800000   // Flash fault
	Evt2GeneralError                  InverterFroniusEvent2 = 0x1000000  // General error
	Evt2GroundingFault                InverterFroniusEvent2 = 0x2000000  // Grounding fault
	Evt2PowerLimitationFault          InverterFroniusEvent2 = 0x4000000  // Power limitation fault
	Evt2ExternalNOContactOpen         InverterFroniusEvent2 = 0x8000000  // External NO contact open
	Evt2ExternalOvervoltageProtection InverterFroniusEvent2 = 0x10000000 // External overvoltage protection has tripped
	Evt2InternalProcessorProgStatus   InverterFroniusEvent2 = 0x20000000 // Internal processor program status
	Evt2SolarNetIssue                 InverterFroniusEvent2 = 0x40000000 // SolarNet issue
	Evt2SupplyVoltageFault            InverterFroniusEvent2 = 0x80000000 // Supply voltage fault
)

var inverterEvent2Strings = map[InverterFroniusEvent2]string{
	Evt2NoSolarNetComm:                "no_solarnet_communication",
	Evt2InverterAddressIncorrect:      "inverter_address_incorrect",
	Evt224hNoFeedIn:                   "24h_no_feed_in",
	Evt2FaultyPlugConnections:         "faulty_plug_connections",
	Evt2IncorrectPhaseAllocation:      "incorrect_phase_allocation",
	Evt2GridConductorOpenPhaseFailure: "grid_conductor_open_or_supply_phase_has_failed",
	Evt2IncompatibleOldSoftware:       "incompatible_or_old_software",
	Evt2PowerDeratingOvertemp:         "power_derating_due_to_overtemperature",
	Evt2JumperSetIncorrectly:          "jumper_set_incorrectly",
	Evt2IncompatibleFeature:           "incompatible_feature",
	Evt2DefectiveVentilatorAirVents:   "defective_ventilator_air_vents_blocked",
	Evt2PowerReductionOnError:         "power_reduction_on_error",
	Evt2ArcDetected:                   "arc_detected",
	Evt2AFCISelfTestFailed:            "afci_self_test_failed",
	Evt2CurrentSensorError:            "current_sensor_error",
	Evt2DCSwitchFault:                 "dc_switch_fault",
	Evt2AFCIDefective:                 "afci_defective",
	Evt2AFCIManualTestSuccessful:      "afci_manual_test_successful",
	Evt2PowerStackSupplyMissing:       "power_stack_supply_missing",
	Evt2AFCICommStopped:               "afci_communication_stopped",
	Evt2AFCIManualTestFailed:          "afci_manual_test_failed",
	Evt2ACPolarityReversed:            "ac_polarity_reversed",
	Evt2ACMeasurementDeviceFault:      "ac_measurement_device_fault",
	Evt2FlashFault:                    "flash_fault",
	Evt2GeneralError:                  "general_error",
	Evt2GroundingFault:                "grounding_fault",
	Evt2PowerLimitationFault:          "power_limitation_fault",
	Evt2ExternalNOContactOpen:         "external_no_contact_open",
	Evt2ExternalOvervoltageProtection: "external_overvoltage_protection_has_tripped",
	Evt2InternalProcessorProgStatus:   "internal_processor_program_status",
	Evt2SolarNetIssue:                 "solarnet_issue",
	Evt2SupplyVoltageFault:            "supply_voltage_fault",
}

var event2SupportedMasks = map[FroniusModel]uint32{
	Primo:        0x442578CC,
	Symo:         0x65A57AC4,
	Galvo:        0x440178C4,
	IGPlus:       0x001F7BCF,
	UnknownModel: 0xFFFFFFFF, // All events supported by default
}

func FroniusInverterEvent2ToMap(events uint32, supportedMask uint32) map[string]bool {
	eventMap := make(map[string]bool)
	for event, name := range inverterEvent2Strings {
		// Only include events that are in the supported mask
		if (supportedMask & uint32(event)) != 0 {
			if (events & uint32(event)) != 0 {
				eventMap[name] = true
			} else {
				eventMap[name] = false
			}
		}
	}
	return eventMap
}

type InverterFroniusEvent2ForMask struct {
	event InverterFroniusEvent2
	mask  uint32
}

func (e InverterFroniusEvent2ForMask) Flags() map[string]bool {
	return FroniusInverterEvent2ToMap(uint32(e.event), e.mask)
}

// Ensure InverterFroniusEvent2 implements DeviceEvents
var _ sunspec.DeviceEvents = InverterFroniusEvent2ForMask{}

// InverterFroniusEvent3 - EvtVnd3 events
type InverterFroniusEvent3 uint32

const (
	Evt3TimeError InverterFroniusEvent3 = 0x1 // Time error
	Evt3USBError  InverterFroniusEvent3 = 0x2 // USB error
	Evt3DCHigh    InverterFroniusEvent3 = 0x4 // DC high
)

var inverterEvent3Strings = map[InverterFroniusEvent3]string{
	Evt3TimeError: "time_error",
	Evt3USBError:  "usb_error",
	Evt3DCHigh:    "dc_high",
}

var event3SupportedMasks = map[FroniusModel]uint32{
	Primo:        0x00000007,
	Symo:         0x00000007,
	Galvo:        0x00000007,
	IGPlus:       0x00000000,
	UnknownModel: 0xFFFFFFFF, // All events supported
}

func FroniusInverterEvent3ToMap(events uint32, supportedMask uint32) map[string]bool {
	eventMap := make(map[string]bool)
	for event, name := range inverterEvent3Strings {
		// Only include events that are in the supported mask
		if (supportedMask & uint32(event)) != 0 {
			if (events & uint32(event)) != 0 {
				eventMap[name] = true
			} else {
				eventMap[name] = false
			}
		}
	}
	return eventMap
}

type InverterFroniusEvent3ForMask struct {
	event InverterFroniusEvent3
	mask  uint32
}

func (e InverterFroniusEvent3ForMask) Flags() map[string]bool {
	return FroniusInverterEvent3ToMap(uint32(e.event), e.mask)
}

// Ensure InverterFroniusEvent3 implements DeviceEvents
var _ sunspec.DeviceEvents = InverterFroniusEvent3ForMask{}

// ParseEvents creates a composite device events structure from raw event values
// If model is Unknown, all events are included without filtering
func ParseEvents(evt1, evt2, evt3 uint32, model FroniusModel) sunspec.DeviceEvents {
	fronEvt1 := InverterFroniusEvent1ForMask{event: InverterFroniusEvent1(evt1), mask: event1SupportedMasks[model]}
	fronEvt2 := InverterFroniusEvent2ForMask{event: InverterFroniusEvent2(evt2), mask: event2SupportedMasks[model]}
	fronEvt3 := InverterFroniusEvent3ForMask{event: InverterFroniusEvent3(evt3), mask: event3SupportedMasks[model]}

	return sunspec.CompositeDeviceEvents{
		EventList: []sunspec.DeviceEvents{fronEvt1, fronEvt2, fronEvt3},
	}
}
