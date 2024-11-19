package domain

import "fmt"

type SensorUpdateEventMixIn struct {
	Id string
}

type SensorUpdateEvent interface {
	SensorUpdateEvent() string
	SensorId() string
}

func (e SensorUpdateEventMixIn) SensorUpdateEvent() string {
	return fmt.Sprintf("%T", e)
}

func (e SensorUpdateEventMixIn) SensorId() string {
	return e.Id
}

type FloatSensorUpdateEvent struct {
	SensorUpdateEventMixIn
	Value    float64
	Decimals uint
}

type BinarySensorUpdateEvent struct {
	SensorUpdateEventMixIn
	Value bool
}

type SwitchSensorUpdateEvent struct {
	SensorUpdateEventMixIn
	Value bool
}

type TextSensorUpdateEvent struct {
	SensorUpdateEventMixIn
	Value string
}

type BridgeStateUpdateEvent struct {
	SensorUpdateEventMixIn
	Value bool
}

type InputNumberSensorUpdateEvent struct {
	SensorUpdateEventMixIn
	Value    float64
	Decimals uint
}
