package component

import (
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/internal/util"
)

type SensorUpdateEvents struct {
	util.SensorUpdateEventBuilder
}

func NewSensorUpdateEvents() *SensorUpdateEvents {
	return &SensorUpdateEvents{
		SensorUpdateEventBuilder: *util.NewSensorUpdateEventBuilder(),
	}
}

func (events *SensorUpdateEvents) AddInverterPowerFlowEvents(pf *domain.InverterPowerFlow, hasStorage bool) *SensorUpdateEvents {

	// Inverter AC Power
	events.AddFloatSensorUpdateEvent(SENSOR_ID_INVERTER_AC_POWER_FLOW, pf.ACPowerWatt, 2)

	var acdc_power float64 = 0
	var dcac_power float64 = 0
	if pf.ACPowerWatt > 0 {
		dcac_power = pf.ACPowerWatt
	} else if pf.ACPowerWatt < 0 {
		acdc_power = pf.ACPowerWatt
	}
	events.AddFloatSensorUpdateEvent(SENSOR_ID_INVERTER_ACDC_POWER, acdc_power, 2)
	events.AddFloatSensorUpdateEvent(SENSOR_ID_INVERTER_DCAC_POWER, dcac_power, 2)

	// Inverter PV Power
	events.AddFloatSensorUpdateEvent(SENSOR_ID_INVERTER_PV_POWER, pf.PVPowerWatt, 2)

	if hasStorage {
		// Battery Charge Power
		events.AddFloatSensorUpdateEvent(SENSOR_ID_BATTERY_CHARGE_POWER, pf.BatteryChargePowerWatt, 2)

		// Battery Discharge Power
		events.AddFloatSensorUpdateEvent(SENSOR_ID_BATTERY_DISCHARGE_POWER, pf.BatteryDischargePowerWatt, 2)

		// Battery Power Flow
		events.AddFloatSensorUpdateEvent(SENSOR_ID_BATTERY_POWER_FLOW, pf.BatteryDCPowerFlowWatt, 2)
	}
	return events
}

func (events *SensorUpdateEvents) AddInverterStateToUpdateEvents(is *domain.InverterState) *SensorUpdateEvents {

	// Inverter Cabinet Temperature
	events.AddFloatSensorUpdateEvent(SENSOR_ID_INVERTER_CABINET_TEMP, is.CabinetTemperature, 1)

	// Inverter Operating State
	events.AddTextSensorUpdateEventWithAttributes(SENSOR_ID_INVERTER_OPERATING_STATE, is.OperatingStateStr, is.SunspecDeviceEvents.Flags())

	return events
}

func (events *SensorUpdateEvents) AddVendorInverterStateToUpdateEvents(ivs *domain.VendorInverterState) *SensorUpdateEvents {

	// Inverter Vendor Operating State
	events.AddTextSensorUpdateEventWithAttributes(SENSOR_ID_INVERTER_VENDOR_OPERATING_STATE, ivs.VendorOperatingStateStr, ivs.VendorDeviceEvents.Flags())

	return events
}

func (events *SensorUpdateEvents) AddInverterStorageStateToUpdateEvents(is *domain.StorageState) *SensorUpdateEvents {

	// Battery SoC
	events.AddFloatSensorUpdateEvent(SENSOR_ID_BATTERY_SOC, is.StateOfCharge, 2)

	// Battery Max Capacity
	events.AddFloatSensorUpdateEvent(SENSOR_ID_BATTERY_MAX_CAPACITY, float64(is.MaxCapacityWatt)/1000, 3)

	// Battery Current Capacity
	events.AddFloatSensorUpdateEvent(SENSOR_ID_BATTERY_CURRENT_CAPACITY, float64(is.CurrentCapacityWatt)/1000, 3)

	// Battery Charge State
	events.AddTextSensorUpdateEvent(SENSOR_ID_BATTERY_OPERATING_STATE, is.ChargeStatusStr)

	return events
}

func (events *SensorUpdateEvents) AddACMeterPowerFlowToUpdateEvents(pf *domain.ACMeterPowerFlow) *SensorUpdateEvents {

	// ACMeter Power Flow
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_POWER_FLOW, pf.CurrentPowerFlowWatt, 2)

	// ACMeter Import Power
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_IMPORT_POWER, pf.CurrentImportPowerWatt, 2)

	// ACMeter Export Power
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_EXPORT_POWER, pf.CurrentExportPowerWatt, 2)

	// ACMeter Total Import Energy
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_TOTAL_ENERGY_IMPORTED, pf.TotalEnergyImportedKWh, 3)

	// ACMeter Total Export Energy
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_TOTAL_ENERGY_EXPORTED, pf.TotalEnergyExportedKWh, 3)

	// ACMeter Grid Frequency
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_GRID_FREQUENCY, pf.Frequency, 1)

	// ACMeter Grid Voltage
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_GRID_VOLTAGE, pf.PhaseAVoltage, 2)

	return events
}

func (events *SensorUpdateEvents) AddDisconnectedACMeterPowerFlowToUpdateEvents() *SensorUpdateEvents {

	// ACMeter Power Flow
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_POWER_FLOW, 0, 2)

	// ACMeter Import Power
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_IMPORT_POWER, 0, 2)

	// ACMeter Export Power
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_EXPORT_POWER, 0, 2)

	// ACMeter Grid Frequency
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_GRID_FREQUENCY, 0, 1)

	// ACMeter Grid Voltage
	events.AddFloatSensorUpdateEvent(SENSOR_ID_ACMETER_GRID_VOLTAGE, 0, 2)

	return events
}

func (events *SensorUpdateEvents) AddHousePowerUpdateEvents(invPf *domain.InverterPowerFlow, acMeterPf *domain.ACMeterPowerFlow) *SensorUpdateEvents {

	var acMeterPower float64 = 0
	if acMeterPf != nil {
		acMeterPower = acMeterPf.CurrentPowerFlowWatt
	}
	if invPf != nil {
		events.AddFloatSensorUpdateEvent(SENSOR_ID_HOUSE_POWER, invPf.ACPowerWatt+acMeterPower, 2)
	}

	return events
}

func (events *SensorUpdateEvents) AddBatteryControlHoldSwitchUpdateEvent(controlHold bool) *SensorUpdateEvents {
	events.AddSwitchSensorUpdateEvent(SWITCH_ID_BATTERY_HOLD, controlHold)

	return events
}

func (events *SensorUpdateEvents) AddBatteryControlChargeSwitchUpdateEvent(controlCharge bool) *SensorUpdateEvents {
	events.AddSwitchSensorUpdateEvent(SWITCH_ID_BATTERY_CHARGE, controlCharge)

	return events
}

func (events *SensorUpdateEvents) AddBatteryControlSetTargetSoCUpdateEvents(value uint8) *SensorUpdateEvents {
	events.AddInputNumberSensorUpdateEvent(INPUT_NUMBER_ID_BATTERY_CHARGE_TARGET_SOC, float64(value))

	return events
}

func (events *SensorUpdateEvents) Sensors() []domain.SensorUpdateEvent {
	return events.Build()
}

func (events *SensorUpdateEvents) ForEach(fn func(event domain.SensorUpdateEvent)) {
	for _, event := range events.Build() {
		fn(event)
	}
}
