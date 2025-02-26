package events

import (
	. "frostnews2mqtt/internal/core/domain"
	"frostnews2mqtt/pkg/sunspec_modbus"
)

func InverterPowerFlowToUpdateEvents(pf *sunspec_modbus.InverterPowerFlow) []any {
	var events []any

	// Inverter AC Power
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_INVERTER_AC_POWER_FLOW,
		},
		Value:    pf.ACPowerWatt,
		Decimals: 2,
	})
	var acdc_power float64 = 0
	var dcac_power float64 = 0
	if pf.ACPowerWatt > 0 {
		dcac_power = pf.ACPowerWatt
	} else if pf.ACPowerWatt < 0 {
		acdc_power = pf.ACPowerWatt
	}
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_INVERTER_ACDC_POWER,
		},
		Value:    acdc_power,
		Decimals: 2,
	})
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_INVERTER_DCAC_POWER,
		},
		Value:    dcac_power,
		Decimals: 2,
	})
	// Inverter PV Power
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_INVERTER_PV_POWER,
		},
		Value:    pf.PVPowerWatt,
		Decimals: 2,
	})
	// Battery Charge Power
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_BATTERY_CHARGE_POWER,
		},
		Value:    pf.BatteryChargePowerWatt,
		Decimals: 2,
	})
	// Battery Discharge Power
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_BATTERY_DISCHARGE_POWER,
		},
		Value:    pf.BatteryDischargePowerWatt,
		Decimals: 2,
	})
	// Battery Power Flow
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_BATTERY_POWER_FLOW,
		},
		Value:    pf.BatteryDCPowerFlowWatt,
		Decimals: 2,
	})

	return events
}

func InverterStateToUpdateEvents(is *sunspec_modbus.InverterState) []any {
	var events []any

	// Inverter Cabinet Temp
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_INVERTER_CABINET_TEMP,
		},
		Value:    is.CabinetTemperature,
		Decimals: 1,
	})
	// Inverter Operating State
	events = append(events, TextSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_INVERTER_OPERATING_STATE,
		},
		Value: is.OperatingStateStr,
	})

	return events
}

func InverterStorageStateToUpdateEvents(is *sunspec_modbus.StorageState) []any {
	var events []any

	// Battery SoC
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_BATTERY_SOC,
		},
		Value:    is.StateOfCharge,
		Decimals: 2,
	})
	// Battery Max Capacity
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_BATTERY_MAX_CAPACITY,
		},
		Value:    float64(is.MaxCapacityWatt) / 1000,
		Decimals: 3,
	})
	// Battery Current Capacity
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_BATTERY_CURRENT_CAPACITY,
		},
		Value:    float64(is.CurrentCapacityWatt) / 1000,
		Decimals: 3,
	})
	// Battery Charge State
	events = append(events, TextSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_BATTERY_OPERATING_STATE,
		},
		Value: is.ChargeStatusStr,
	})

	return events
}

func ACMeterPowerFlowToUpdateEvents(pf *sunspec_modbus.ACMeterPowerFlow) []any {
	var events []any

	// ACMeter Power Flow
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_ACMETER_POWER_FLOW,
		},
		Value:    pf.CurrentPowerFlowWatt,
		Decimals: 2,
	})
	// ACMeter Import Power
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_ACMETER_IMPORT_POWER,
		},
		Value:    pf.CurrentImportPowerWatt,
		Decimals: 2,
	})
	// ACMeter Export Power
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_ACMETER_EXPORT_POWER,
		},
		Value:    pf.CurrentExportPowerWatt,
		Decimals: 2,
	})
	// ACMeter Total Import Energy
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_ACMETER_TOTAL_ENERGY_IMPORTED,
		},
		Value:    pf.TotalEnergyImportedKWh,
		Decimals: 3,
	})
	// ACMeter Total Export Energy
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_ACMETER_TOTAL_ENERGY_EXPORTED,
		},
		Value:    pf.TotalEnergyExportedKWh,
		Decimals: 3,
	})
	// ACMeter Grid Frequency
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_ACMETER_GRID_FREQUENCY,
		},
		Value:    pf.Frequency,
		Decimals: 1,
	})
	// ACMeter Grid Voltage
	events = append(events, FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SENSOR_ID_ACMETER_GRID_VOLTAGE,
		},
		Value:    pf.PhaseAVoltage,
		Decimals: 2,
	})

	return events
}

func HousePowerUpdateEvents(invPf *sunspec_modbus.InverterPowerFlow, acMeterPf *sunspec_modbus.ACMeterPowerFlow) []any {
	var events []any
	if invPf != nil && acMeterPf != nil {
		events = append(events, FloatSensorUpdateEvent{
			SensorUpdateEventMixIn: SensorUpdateEventMixIn{
				Id: SENSOR_ID_HOUSE_POWER,
			},
			Value:    invPf.ACPowerWatt + acMeterPf.CurrentPowerFlowWatt,
			Decimals: 2,
		})
	}
	return events
}

func BatteryControlSwitchesUpdateEvents(controlHold, controlCharge bool) []any {
	var events []any
	events = append(events, BatteryControlHoldSwitchUpdateEvents(controlHold))
	events = append(events, BatteryControlChargeSwitchUpdateEvents(controlCharge))
	return events
}

func BatteryControlHoldSwitchUpdateEvents(controlHold bool) any {
	return SwitchSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SWITCH_ID_BATTERY_HOLD,
		},
		Value: controlHold,
	}
}

func BatteryControlChargeSwitchUpdateEvents(controlCharge bool) any {
	return SwitchSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: SWITCH_ID_BATTERY_CHARGE,
		},
		Value: controlCharge,
	}
}

func BatteryControlSetTargetSoCUpdateEvents(value uint8) []any {
	var events []any
	events = append(events, InputNumberSensorUpdateEvent{
		SensorUpdateEventMixIn: SensorUpdateEventMixIn{
			Id: INPUT_NUMBER_ID_BATTERY_CHARGE_TARGET_SOC,
		},
		Value: float64(value),
	})
	return events
}
