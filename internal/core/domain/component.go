package domain

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
