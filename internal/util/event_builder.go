package util

import "github.com/berfenger/frostnews2mqtt/internal/core/domain"

type SensorUpdateEventBuilder struct {
	events []domain.SensorUpdateEvent
}

func NewSensorUpdateEventBuilder() *SensorUpdateEventBuilder {
	return &SensorUpdateEventBuilder{
		events: make([]domain.SensorUpdateEvent, 0),
	}
}

func (b *SensorUpdateEventBuilder) AddFloatSensorUpdateEvent(id string, value float64, decimals uint) *SensorUpdateEventBuilder {
	event := domain.FloatSensorUpdateEvent{
		SensorUpdateEventMixIn: domain.SensorUpdateEventMixIn{
			Id: id,
		},
		Value:    value,
		Decimals: decimals,
	}
	b.events = append(b.events, event)
	return b
}

func (b *SensorUpdateEventBuilder) AddBinarySensorUpdateEvent(id string, value bool) *SensorUpdateEventBuilder {
	event := domain.BinarySensorUpdateEvent{
		SensorUpdateEventMixIn: domain.SensorUpdateEventMixIn{
			Id: id,
		},
		Value: value,
	}
	b.events = append(b.events, event)
	return b
}

func (b *SensorUpdateEventBuilder) AddSwitchSensorUpdateEvent(id string, value bool) *SensorUpdateEventBuilder {
	event := domain.SwitchSensorUpdateEvent{
		SensorUpdateEventMixIn: domain.SensorUpdateEventMixIn{
			Id: id,
		},
		Value: value,
	}
	b.events = append(b.events, event)
	return b
}

func (b *SensorUpdateEventBuilder) AddInputNumberSensorUpdateEvent(id string, value float64) *SensorUpdateEventBuilder {
	event := domain.InputNumberSensorUpdateEvent{
		SensorUpdateEventMixIn: domain.SensorUpdateEventMixIn{
			Id: id,
		},
		Value: value,
	}
	b.events = append(b.events, event)
	return b
}

func (b *SensorUpdateEventBuilder) AddTextSensorUpdateEvent(id string, value string) *SensorUpdateEventBuilder {
	event := domain.TextSensorUpdateEvent{
		SensorUpdateEventMixIn: domain.SensorUpdateEventMixIn{
			Id: id,
		},
		Value: value,
	}
	b.events = append(b.events, event)
	return b
}

func (b *SensorUpdateEventBuilder) AddTextSensorUpdateEventWithAttributes(id string, value string, attributes map[string]bool) *SensorUpdateEventBuilder {
	event := domain.TextSensorUpdateEvent{
		SensorUpdateEventMixIn: domain.SensorUpdateEventMixIn{
			Id:         id,
			Attributes: boolMapToAnyMap(attributes),
		},
		Value: value,
	}
	b.events = append(b.events, event)
	return b
}

func (b *SensorUpdateEventBuilder) Build() []domain.SensorUpdateEvent {
	return b.events
}

func boolMapToAnyMap(in map[string]bool) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
