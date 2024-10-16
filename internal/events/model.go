package events

// Sensor Model
type Device struct {
	Id           string
	Name         string
	Version      string
	Model        string
	Manufacturer string
	ViaDevice    string
}

type GenericSensor struct {
	Device            Device
	Id                string
	SensorType        string
	Name              string
	UniqueId          string
	UnitOfMeasurement string
	StateClass        string // measurement, duration, total_increasing (for acc energy)
	DeviceClass       string // voltage, current, power, energy
	EntityCategory    string // diagnostic, config, nil
	EnabledByDefault  *bool
	Icon              string
}

type GenericSwitch struct {
	Device   Device
	Id       string
	Name     string
	UniqueId string
	Icon     string
}

type GenericInputNumber struct {
	Device       Device
	Id           string
	Name         string
	UniqueId     string
	Icon         string
	Max          float64
	Min          float64
	Step         float64
	Mode         string
	InitialValue float64
}

// EventStream model
type GenericSensorUpdateEvent struct {
	Id string
}

type SensorUpdateEvent struct {
	GenericSensorUpdateEvent
	Value    float64
	Decimals uint
}

type BinarySensorUpdateEvent struct {
	GenericSensorUpdateEvent
	Value bool
}

type SwitchSensorUpdateEvent struct {
	GenericSensorUpdateEvent
	Value bool
}

type TextSensorUpdateEvent struct {
	GenericSensorUpdateEvent
	Value string
}

type BridgeStateUpdateEvent struct {
	GenericSensorUpdateEvent
	Value bool
}

type InputNumberSensorUpdateEvent struct {
	GenericSensorUpdateEvent
	Value    float64
	Decimals uint
}

type ParsedCommand interface{}

type BatteryControlCommand interface {
	ParsedCommand
}

type BatteryControlHold struct {
	BatteryControlCommand
	On bool
}

type BatteryControlCharge struct {
	BatteryControlCommand
	On bool
}

type BatteryControlSetTargetSoC struct {
	BatteryControlCommand
	TargetSoC uint8
}
