package events

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"frostnews2mqtt/pkg/sunspec_modbus"

	"github.com/carlmjohnson/versioninfo"
)

const (
	SENSOR_ID_BRIDGE_STATE                    = "bridge"
	SENSOR_ID_INVERTER_CABINET_TEMP           = "inverter_cabinet_temperature"
	SENSOR_ID_INVERTER_OPERATING_STATE        = "inverter_operating_state"
	SENSOR_ID_INVERTER_AC_POWER_FLOW          = "inverter_ac_power_flow"
	SENSOR_ID_INVERTER_ACDC_POWER             = "inverter_acdc_power"
	SENSOR_ID_INVERTER_DCAC_POWER             = "inverter_dcac_power"
	SENSOR_ID_INVERTER_PV_POWER               = "inverter_pv_power"
	SENSOR_ID_HOUSE_POWER                     = "house_power"
	SENSOR_ID_BATTERY_SOC                     = "battery_soc"
	SENSOR_ID_BATTERY_MAX_CAPACITY            = "battery_max_capacity"
	SENSOR_ID_BATTERY_CURRENT_CAPACITY        = "battery_current_capacity"
	SENSOR_ID_BATTERY_OPERATING_STATE         = "battery_operating_state"
	SENSOR_ID_BATTERY_CHARGE_POWER            = "battery_charge_power"
	SENSOR_ID_BATTERY_DISCHARGE_POWER         = "battery_discharge_power"
	SENSOR_ID_BATTERY_POWER_FLOW              = "battery_power_flow"
	SENSOR_ID_ACMETER_POWER_FLOW              = "acmeter_power_flow"
	SENSOR_ID_ACMETER_IMPORT_POWER            = "acmeter_import_power"
	SENSOR_ID_ACMETER_EXPORT_POWER            = "acmeter_export_power"
	SENSOR_ID_ACMETER_TOTAL_ENERGY_IMPORTED   = "acmeter_total_energy_imported"
	SENSOR_ID_ACMETER_TOTAL_ENERGY_EXPORTED   = "acmeter_total_energy_exported"
	SENSOR_ID_ACMETER_GRID_FREQUENCY          = "acmeter_grid_frequency"
	SENSOR_ID_ACMETER_GRID_VOLTAGE            = "acmeter_grid_voltage"
	SWITCH_ID_BATTERY_HOLD                    = "battery_hold"
	SWITCH_ID_BATTERY_CHARGE                  = "battery_charge"
	INPUT_NUMBER_ID_BATTERY_CHARGE_TARGET_SOC = "battery_charge_target_soc"
	STATE_CLASS_DURATION                      = "duration"
	STATE_CLASS_MEASUREMENT                   = "measurement"
	STATE_CLASS_TOTAL_INCREASING              = "total_increasing"
	DEVICE_CLASS_BATTERY                      = "battery"
	DEVICE_CLASS_CURRENT                      = "current"
	DEVICE_CLASS_ENERGY                       = "energy"
	DEVICE_CLASS_ENERGY_STORAGE               = "energy_storage"
	DEVICE_CLASS_FREQUENCY                    = "frequency"
	DEVICE_CLASS_POWER                        = "power"
	DEVICE_CLASS_TEMPERATURE                  = "temperature"
	DEVICE_CLASS_VOLTAGE                      = "voltage"
	DEVICE_CLASS_CONNECTIVITY                 = "connectivity"
	ENTITY_CLASS_DIAGNOSTIC                   = "diagnostic"
	ENTITY_CLASS_CONFIG                       = "config"
	SENSOR_TYPE_SENSOR                        = "sensor"
	SENSOR_TYPE_BINARY                        = "binary_sensor"
	INPUT_NUMBER_MODE_BOX                     = "box"
	INPUT_NUMBER_MODE_SLIDER                  = "slider"
)

func BridgeDevice(baseTopic string) Device {
	return Device{
		Id:           fmt.Sprintf("frostnews_bridge_%s", md5HashShort(baseTopic)),
		Manufacturer: "ACasal",
		Model:        "Frostnews",
		Version:      versioninfo.Short(),
		Name:         fmt.Sprintf("Frostnews %s", md5HashShort(baseTopic)),
	}
}

func InverterDevice(info *sunspec_modbus.InverterInfo) Device {
	return Device{
		Id:           fmt.Sprintf("fro_inverter_%s", md5HashShort(info.Serial)),
		Version:      info.Version,
		Manufacturer: info.Manufacturer,
		Model:        info.Model,
		Name:         fmt.Sprintf("%s %s %s", info.Manufacturer, info.Model, md5HashShort(info.Serial)),
	}
}

func IdDevice(device Device) Device {
	return Device{
		Id:   device.Id,
		Name: device.Name,
	}
}

func InverterBaseSensors(inverterDevice Device, info *sunspec_modbus.InverterInfo, trackHousePower bool) []GenericSensor {

	var sensors []GenericSensor

	// Inverter Cabinet Temperature
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_INVERTER_CABINET_TEMP,
		SensorType:        "sensor",
		Name:              "Inverter cabinet temperature",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_TEMPERATURE,
		UnitOfMeasurement: "°C",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_INVERTER_CABINET_TEMP),
	})

	// Inverter Operating State
	sensors = append(sensors, GenericSensor{
		Device:     inverterDevice,
		Id:         SENSOR_ID_INVERTER_OPERATING_STATE,
		SensorType: "sensor",
		Name:       "Inverter operating state",
		UniqueId:   uniqueId(inverterDevice.Id, SENSOR_ID_INVERTER_OPERATING_STATE),
	})

	// Inverter AC Power
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_INVERTER_AC_POWER_FLOW,
		SensorType:        "sensor",
		Name:              "Inverter AC power flow",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_INVERTER_AC_POWER_FLOW),
	})

	// Inverter AC-DC Power
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_INVERTER_ACDC_POWER,
		SensorType:        "sensor",
		Name:              "Inverter AC-DC power",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_INVERTER_ACDC_POWER),
	})

	// Inverter DC-AC Power
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_INVERTER_DCAC_POWER,
		SensorType:        "sensor",
		Name:              "Inverter DC-AC power",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_INVERTER_DCAC_POWER),
	})

	// Inverter PV Power
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_INVERTER_PV_POWER,
		SensorType:        "sensor",
		Name:              "Inverter PV power",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		Icon:              "mdi:solar-power",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_INVERTER_PV_POWER),
	})

	if trackHousePower {
		// House power
		sensors = append(sensors, GenericSensor{
			Device:            inverterDevice,
			Id:                SENSOR_ID_HOUSE_POWER,
			SensorType:        "sensor",
			Name:              "House power",
			StateClass:        STATE_CLASS_MEASUREMENT,
			DeviceClass:       DEVICE_CLASS_POWER,
			UnitOfMeasurement: "W",
			Icon:              "mdi:home-lightning-bolt",
			UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_HOUSE_POWER),
		})
	}

	return sensors
}

func InverterStorageSensors(inverterDevice Device) []GenericSensor {

	var sensors []GenericSensor

	// Battery SoC
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_BATTERY_SOC,
		SensorType:        "sensor",
		Name:              "Battery SoC",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_BATTERY,
		UnitOfMeasurement: "%",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_BATTERY_SOC),
	})

	// Battery Max Capacity
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_BATTERY_MAX_CAPACITY,
		SensorType:        "sensor",
		Name:              "Battery max capacity",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_ENERGY,
		UnitOfMeasurement: "kWh",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_BATTERY_MAX_CAPACITY),
	})

	// Battery Current Capacity
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_BATTERY_CURRENT_CAPACITY,
		SensorType:        "sensor",
		Name:              "Battery current capacity",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_ENERGY,
		UnitOfMeasurement: "kWh",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_BATTERY_CURRENT_CAPACITY),
	})

	// Battery Charge State
	sensors = append(sensors, GenericSensor{
		Device:     inverterDevice,
		Id:         SENSOR_ID_BATTERY_OPERATING_STATE,
		SensorType: "sensor",
		Name:       "Battery operating state",
		UniqueId:   uniqueId(inverterDevice.Id, SENSOR_ID_BATTERY_OPERATING_STATE),
	})

	// Battery Charge Power
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_BATTERY_CHARGE_POWER,
		SensorType:        "sensor",
		Name:              "Battery charge power",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_BATTERY_CHARGE_POWER),
	})

	// Battery Discharge Power
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_BATTERY_DISCHARGE_POWER,
		SensorType:        "sensor",
		Name:              "Battery discharge power",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_BATTERY_DISCHARGE_POWER),
	})

	// Battery Power Flow
	sensors = append(sensors, GenericSensor{
		Device:            inverterDevice,
		Id:                SENSOR_ID_BATTERY_POWER_FLOW,
		SensorType:        "sensor",
		Name:              "Battery power flow",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		UniqueId:          uniqueId(inverterDevice.Id, SENSOR_ID_BATTERY_POWER_FLOW),
	})

	return sensors
}

func ACMeterDevice(info *sunspec_modbus.ACMeterInfo) Device {
	return Device{
		Id:           fmt.Sprintf("fro_acmeter_%s", md5HashShort(info.Serial)),
		Version:      info.Version,
		Manufacturer: info.Manufacturer,
		Model:        info.Model,
		Name:         fmt.Sprintf("%s %s %s", info.Manufacturer, info.Model, md5HashShort(info.Serial)),
	}
}

func ACMeterBaseSensors(acmeterDevice Device, info *sunspec_modbus.ACMeterInfo) []GenericSensor {

	var sensors []GenericSensor

	// ACMeter Power Flow
	sensors = append(sensors, GenericSensor{
		Device:            acmeterDevice,
		Id:                SENSOR_ID_ACMETER_POWER_FLOW,
		SensorType:        "sensor",
		Name:              "Power flow",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		UniqueId:          uniqueId(acmeterDevice.Id, SENSOR_ID_ACMETER_POWER_FLOW),
	})

	// ACMeter Import Power
	sensors = append(sensors, GenericSensor{
		Device:            acmeterDevice,
		Id:                SENSOR_ID_ACMETER_IMPORT_POWER,
		SensorType:        "sensor",
		Name:              "Import power",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		UniqueId:          uniqueId(acmeterDevice.Id, SENSOR_ID_ACMETER_IMPORT_POWER),
	})

	// ACMeter Export Power
	sensors = append(sensors, GenericSensor{
		Device:            acmeterDevice,
		Id:                SENSOR_ID_ACMETER_EXPORT_POWER,
		SensorType:        "sensor",
		Name:              "Export power",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_POWER,
		UnitOfMeasurement: "W",
		UniqueId:          uniqueId(acmeterDevice.Id, SENSOR_ID_ACMETER_EXPORT_POWER),
	})

	// ACMeter Total Energy Imported
	sensors = append(sensors, GenericSensor{
		Device:            acmeterDevice,
		Id:                SENSOR_ID_ACMETER_TOTAL_ENERGY_IMPORTED,
		SensorType:        "sensor",
		Name:              "Total energy imported",
		StateClass:        STATE_CLASS_TOTAL_INCREASING,
		DeviceClass:       DEVICE_CLASS_ENERGY,
		UnitOfMeasurement: "kWh",
		UniqueId:          uniqueId(acmeterDevice.Id, SENSOR_ID_ACMETER_TOTAL_ENERGY_IMPORTED),
	})

	// ACMeter Total Energy Exported
	sensors = append(sensors, GenericSensor{
		Device:            acmeterDevice,
		Id:                SENSOR_ID_ACMETER_TOTAL_ENERGY_EXPORTED,
		SensorType:        "sensor",
		Name:              "Total energy exported",
		StateClass:        STATE_CLASS_TOTAL_INCREASING,
		DeviceClass:       DEVICE_CLASS_ENERGY,
		UnitOfMeasurement: "kWh",
		UniqueId:          uniqueId(acmeterDevice.Id, SENSOR_ID_ACMETER_TOTAL_ENERGY_EXPORTED),
	})

	// ACMeter Grid Frequency
	sensors = append(sensors, GenericSensor{
		Device:            acmeterDevice,
		Id:                SENSOR_ID_ACMETER_GRID_FREQUENCY,
		SensorType:        "sensor",
		Name:              "Grid frequency",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_FREQUENCY,
		UnitOfMeasurement: "Hz",
		Icon:              "mdi:sine-wave",
		EnabledByDefault:  optionalBool(false),
		UniqueId:          uniqueId(acmeterDevice.Id, SENSOR_ID_ACMETER_GRID_FREQUENCY),
	})

	// ACMeter Grid Voltage
	sensors = append(sensors, GenericSensor{
		Device:            acmeterDevice,
		Id:                SENSOR_ID_ACMETER_GRID_VOLTAGE,
		SensorType:        "sensor",
		Name:              "Grid voltage",
		StateClass:        STATE_CLASS_MEASUREMENT,
		DeviceClass:       DEVICE_CLASS_VOLTAGE,
		UnitOfMeasurement: "V",
		EnabledByDefault:  optionalBool(false),
		UniqueId:          uniqueId(acmeterDevice.Id, SENSOR_ID_ACMETER_GRID_VOLTAGE),
	})

	return sensors
}

func BridgeSensors(bridgeDevice Device) []GenericSensor {

	var sensors []GenericSensor

	// ACMeter Power Flow
	sensors = append(sensors, GenericSensor{
		Device:         bridgeDevice,
		Id:             SENSOR_ID_BRIDGE_STATE,
		SensorType:     "binary_sensor",
		Name:           "Connection state",
		DeviceClass:    DEVICE_CLASS_CONNECTIVITY,
		EntityCategory: ENTITY_CLASS_DIAGNOSTIC,
		UniqueId:       uniqueId(bridgeDevice.Id, SENSOR_ID_BRIDGE_STATE),
	})

	return sensors
}

func BatteryControlSwitches(inverterDevice Device) []GenericSwitch {

	var switches []GenericSwitch

	// Battery hold
	switches = append(switches, GenericSwitch{
		Device:   inverterDevice,
		Id:       SWITCH_ID_BATTERY_HOLD,
		Name:     "Battery hold",
		UniqueId: uniqueId(inverterDevice.Id, SWITCH_ID_BATTERY_HOLD),
		Icon:     "mdi:battery-lock",
	})
	// Battery charge
	switches = append(switches, GenericSwitch{
		Device:   inverterDevice,
		Id:       SWITCH_ID_BATTERY_CHARGE,
		Name:     "Battery charge",
		UniqueId: uniqueId(inverterDevice.Id, SWITCH_ID_BATTERY_CHARGE),
		Icon:     "mdi:battery-plus",
	})

	return switches
}

func BatteryControlInputNumbers(inverterDevice Device) []GenericInputNumber {

	var inputNumbers []GenericInputNumber

	// Battery charge target SoC
	inputNumbers = append(inputNumbers, GenericInputNumber{
		Device:       inverterDevice,
		Id:           INPUT_NUMBER_ID_BATTERY_CHARGE_TARGET_SOC,
		Name:         "Battery charge target SoC",
		UniqueId:     uniqueId(inverterDevice.Id, INPUT_NUMBER_ID_BATTERY_CHARGE_TARGET_SOC),
		Icon:         "mdi:ticket-percent",
		Max:          100,
		Min:          0,
		Step:         5,
		Mode:         INPUT_NUMBER_MODE_BOX,
		InitialValue: 100,
	})

	return inputNumbers
}

func uniqueId(baseId, id string) string {
	return fmt.Sprintf("uid_%s_%s", baseId, id)
}

func md5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func md5HashShort(text string) string {
	hash := md5Hash(text)
	return hash[0:8]
}

func optionalBool(value bool) *bool {
	return &value
}
