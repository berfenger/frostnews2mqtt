package mqtt

import (
	"fmt"
	"frostnews2mqtt/internal/events"
)

type HADiscoveryConfig struct {
	Device            HADiscoveryDevice `json:"device"`
	StateTopic        string            `json:"state_topic"`
	CommandTopic      string            `json:"command_topic,omitempty"`
	StateClass        string            `json:"state_class,omitempty"`
	DeviceClass       string            `json:"device_class,omitempty"`
	UnitOfMeasurement string            `json:"unit_of_measurement,omitempty"`
	AvTopic           string            `json:"availability_topic,omitempty"`
	EntityCategory    string            `json:"entity_category,omitempty"`
	Name              string            `json:"name"`
	UniqueId          string            `json:"unique_id"`
	Platform          string            `json:"platform"`
	EnabledByDefault  *bool             `json:"enabled_by_default,omitempty"`
	PayloadOn         string            `json:"payload_on,omitempty"`
	PayloadOff        string            `json:"payload_off,omitempty"`
	Icon              string            `json:"icon,omitempty"`
	Min               float64           `json:"min,omitempty"`
	Max               float64           `json:"max,omitempty"`
	Step              float64           `json:"step,omitempty"`
	Mode              string            `json:"mode,omitempty"`
	InitialValue      float64           `json:"initial,omitempty"`
}

type HADiscoveryDevice struct {
	Id           []string `json:"identifiers"`
	Manufacturer string   `json:"manufacturer,omitempty"`
	Version      string   `json:"sw_version,omitempty"`
	Model        string   `json:"model,omitempty"`
	Name         string   `json:"name,omitempty"`
	ViaDevice    string   `json:"via_device,omitempty"`
}

func HADiscoverySensorTopic(sensor events.GenericSensor) string {
	return fmt.Sprintf("homeassistant/%s/%s/%s/config", sensor.SensorType, sensor.Device.Id, sensor.Id)
}

func HADiscoverySwitchTopic(sensor events.GenericSwitch) string {
	return fmt.Sprintf("homeassistant/switch/%s/%s/config", sensor.Device.Id, sensor.Id)
}

func HADiscoveryInputNumberTopic(sensor events.GenericInputNumber) string {
	return fmt.Sprintf("homeassistant/number/%s/%s/config", sensor.Device.Id, sensor.Id)
}

func GenericSensorToHADiscoveryMessage(client *MQTTClient, sensor events.GenericSensor) HADiscoveryConfig {
	dev := device(sensor.Device)
	var topic string
	switch {
	case sensor.Id == events.SENSOR_ID_BRIDGE_STATE:
		topic = client.BridgeStateTopic()
	case sensor.SensorType == events.SENSOR_TYPE_SENSOR:
		topic = client.SensorStateTopic(sensor.Id)
	case sensor.SensorType == events.SENSOR_TYPE_BINARY:
		topic = client.BinarySensorStateTopic(sensor.Id)
	}
	disConfig := HADiscoveryConfig{
		Device:            dev,
		StateTopic:        topic,
		StateClass:        sensor.StateClass,
		DeviceClass:       sensor.DeviceClass,
		UnitOfMeasurement: sensor.UnitOfMeasurement,
		AvTopic:           client.BridgeStateTopic(),
		EntityCategory:    sensor.EntityCategory,
		Name:              sensor.Name,
		UniqueId:          sensor.UniqueId,
		Icon:              sensor.Icon,
		EnabledByDefault:  sensor.EnabledByDefault,
		Platform:          "mqtt",
	}
	if sensor.Id == events.SENSOR_ID_BRIDGE_STATE {
		disConfig.PayloadOn = MQTT_PAYLOAD_ONLINE
		disConfig.PayloadOff = MQTT_PAYLOAD_OFFLINE
	} else if sensor.Id == events.SENSOR_TYPE_BINARY {
		disConfig.PayloadOn = MQTT_PAYLOAD_ON
		disConfig.PayloadOff = MQTT_PAYLOAD_OFF
	}
	return disConfig
}

func GenericSwitchToHADiscoveryMessage(client *MQTTClient, _switch events.GenericSwitch) HADiscoveryConfig {
	dev := device(_switch.Device)
	topic := client.SwitchStateTopic(_switch.Id)
	cmdTopic := client.SwitchCommandTopic(_switch.Id)
	disConfig := HADiscoveryConfig{
		Device:       dev,
		StateTopic:   topic,
		CommandTopic: cmdTopic,
		AvTopic:      client.BridgeStateTopic(),
		Name:         _switch.Name,
		UniqueId:     _switch.UniqueId,
		Icon:         _switch.Icon,
		Platform:     "mqtt",
		PayloadOn:    MQTT_PAYLOAD_ON,
		PayloadOff:   MQTT_PAYLOAD_OFF,
	}
	return disConfig
}

func GenericInputNumberToHADiscoveryMessage(client *MQTTClient, inputNumber events.GenericInputNumber) HADiscoveryConfig {
	dev := device(inputNumber.Device)
	topic := client.InputNumberStateTopic(inputNumber.Id)
	cmdTopic := client.InputNumberCommandTopic(inputNumber.Id)
	disConfig := HADiscoveryConfig{
		Device:       dev,
		StateTopic:   topic,
		CommandTopic: cmdTopic,
		AvTopic:      client.BridgeStateTopic(),
		Name:         inputNumber.Name,
		UniqueId:     inputNumber.UniqueId,
		Icon:         inputNumber.Icon,
		Platform:     "mqtt",
		Min:          inputNumber.Min,
		Max:          inputNumber.Max,
		Step:         inputNumber.Step,
		Mode:         inputNumber.Mode,
		InitialValue: inputNumber.InitialValue,
	}
	return disConfig
}

func device(d events.Device) HADiscoveryDevice {
	return HADiscoveryDevice{
		Id:           []string{d.Id},
		Manufacturer: d.Manufacturer,
		Version:      d.Version,
		Model:        d.Model,
		Name:         d.Name,
		ViaDevice:    d.ViaDevice,
	}
}
