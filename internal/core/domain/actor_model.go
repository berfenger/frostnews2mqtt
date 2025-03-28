package domain

import "github.com/berfenger/frostnews2mqtt/pkg/sunspec_modbus"

const (
	ACTOR_ID_MASTER          = "master"
	ACTOR_ID_MODBUS          = "modbus"
	ACTOR_ID_POWERFLOW       = "powerflow"
	ACTOR_ID_MQTT            = "mqtt"
	ACTOR_ID_BATTERY_CONTROL = "battery_control"
	ACTOR_ID_HA_DISCOVERY    = "hadiscovery"
)

type GetDevicesInfoRequest struct {
	ActorRequestMixIn
}

type GetDevicesInfoResponse struct {
	ActorResponseMixIn
	Inverter *sunspec_modbus.InverterInfo
	ACMeter  *sunspec_modbus.ACMeterInfo
}

type GetPowerFlowRequest struct {
	ActorRequestMixIn
}

type GetPowerFlowResponse struct {
	ActorResponseMixIn
	Inverter *sunspec_modbus.InverterPowerFlow
	ACMeter  *sunspec_modbus.ACMeterPowerFlow
}

type GetStorageStateRequest struct {
	ActorRequestMixIn
}

type GetStorageStateResponse struct {
	ActorResponseMixIn
	StorageState *sunspec_modbus.StorageState
}

type GetInverterStateRequest struct {
	ActorRequestMixIn
}

type GetInverterStateResponse struct {
	ActorResponseMixIn
	InverterState *sunspec_modbus.InverterState
}

type SetStorageControlRequest struct {
	ActorRequestMixIn
	Params sunspec_modbus.StorageControlParams
}

type SetStorageControlResponse struct {
	ActorResponseMixIn
}

type GetStorageControlPowerFlowRequest struct {
	ActorRequestMixIn
}

type GetStorageControlPowerFlowResponse struct {
	ActorResponseMixIn
	StorageState      *sunspec_modbus.StorageState
	ACMeterPowerFlow  *sunspec_modbus.ACMeterPowerFlow
	InverterPowerFlow *sunspec_modbus.InverterPowerFlow
}

type PublishMessageRequest struct {
	ActorRequestMixIn
	Topic   string
	Payload string
	Retain  bool
}

type PublishMessageResponse struct {
	ActorResponseMixIn
}

type PublishSensorUpdateRequest struct {
	ActorRequestMixIn
	Retain bool
	Event  SensorUpdateEvent
}

type PublishSensorUpdateResponse struct {
	ActorResponseMixIn
}

type PublishDiscoveryRequest struct {
	ActorRequestMixIn
	Sensors      []GenericSensor
	Switches     []GenericSwitch
	InputNumbers []GenericInputNumber
}

type PublishDiscoveryResponse struct {
	ActorResponseMixIn
}

type ActorHealthRequest struct {
	ActorRequestMixIn
}

type ActorHealthResponse struct {
	ActorResponseMixIn
	Id      string
	Healthy bool
	State   string
}
