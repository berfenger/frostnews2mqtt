package actor

import (
	"errors"
	"fmt"
	"frostnews2mqtt/internal/config"
	"frostnews2mqtt/internal/events"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/sirupsen/logrus"
)

const (
	HADISCOVERY_ACTOR_ID = "hadiscovery"
)

type HADiscoveryActor struct {
	config             *config.Config
	behavior           actor.Behavior
	stash              *Stash
	modbusActor        *actor.PID
	mqttActor          *actor.PID
	modbusActorHealthy bool
	mqttActorHealthy   bool
	healthyRecv        int

	logger *logrus.Entry
}

func NewHADiscoveryActor(config *config.Config, modbusActor *actor.PID, mqttActor *actor.PID, logger *logrus.Logger) *HADiscoveryActor {
	act := &HADiscoveryActor{
		config:      config,
		modbusActor: modbusActor,
		mqttActor:   mqttActor,
		behavior:    actor.NewBehavior(),
		stash:       &Stash{},
		logger:      ActorLogger("hadiscovery", logger),
	}
	act.behavior.Become(act.StartingReceive)
	return act
}

func (state *HADiscoveryActor) Receive(context actor.Context) {
	state.behavior.Receive(context)
}

func (state *HADiscoveryActor) StartingReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.logger.Debug("hadiscovery@starting started")

		// Check Modbus and MQTT actor healthy
		state.healthyRecv = 0
		state.modbusActorHealthy = false
		state.mqttActorHealthy = false
		// Modbus Actor Request
		PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, ActorHealthRequest{}, 2*time.Second), func(err error) any {
			return ActorHealthResponse{
				Id:      MODBUS_ACTOR_ID,
				Healthy: false,
				Error:   fmt.Sprintf("%s", err),
			}
		})
		// MQTT Actor Request
		PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.mqttActor, ActorHealthRequest{}, 2*time.Second), func(err error) any {
			return ActorHealthResponse{
				Id:      MQTT_ACTOR_ID,
				Healthy: false,
				Error:   fmt.Sprintf("%s", err),
			}
		})
		state.behavior.Become(state.WaitingHealthyReceive)
	case *actor.Restarting:
	default:
		state.logger.Trace("hadiscovery@starting: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

func (state *HADiscoveryActor) WaitingHealthyReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ActorHealthResponse:
		state.logger.Debug("hadiscovery@healthcheck ActorHealthResponse", "sender", msg.Id, "healthy", msg.Healthy)
		state.healthyRecv++
		if msg.Healthy {
			if msg.Id == MODBUS_ACTOR_ID {
				state.modbusActorHealthy = true
			} else if msg.Id == MQTT_ACTOR_ID {
				state.mqttActorHealthy = true
			}
		}
		if state.healthyRecv == 2 {

			if state.modbusActorHealthy && state.mqttActorHealthy {
				// Ask Modbus GetDevicesInfoRequest
				PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, GetDevicesInfoRequest{}, 2*time.Second), func(err error) any {
					return CommandErrorResponse{
						Error: fmt.Sprintf("%s", err),
					}
				})
				state.behavior.Become(state.WaitingInfoReceive)
				state.stash.UnstashAll(ctx)
			} else {
				panic(errors.New("MQTT Actor or Modbus Actor are not healthy"))
			}
		}
	default:
		state.logger.Trace("hadiscovery@healthcheck: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

func (state *HADiscoveryActor) Done(ctx actor.Context) {

}

func (state *HADiscoveryActor) WaitingInfoReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case GetDevicesInfoResponse:
		state.logger.Debug("hadiscovery@info: GetDevicesInfoResponse", "response", msg)

		var sensors []events.GenericSensor
		var switches []events.GenericSwitch
		var inputNumbers []events.GenericInputNumber

		bridgeDevice := events.BridgeDevice(state.config.MQTT.BaseTopic)
		bridgeSensors := events.BridgeSensors(bridgeDevice)
		sensors = append(sensors, bridgeSensors...)

		inverterDevice := events.InverterDevice(msg.Inverter)
		inverterDevice.ViaDevice = bridgeDevice.Id
		inverterSensors := events.InverterBaseSensors(inverterDevice, msg.Inverter, state.config.TrackHousePower && msg.ACMeter != nil)
		for i := range inverterSensors {
			if i > 0 {
				inverterSensors[i].Device = events.IdDevice(inverterDevice)
			}
			sensors = append(sensors, inverterSensors[i])
		}

		if msg.Inverter.HasStorage {
			storageSensors := events.InverterStorageSensors(events.IdDevice(inverterDevice))
			sensors = append(sensors, storageSensors...)
		}
		if msg.ACMeter != nil {
			acmeterDevice := events.ACMeterDevice(msg.ACMeter)
			acmeterDevice.ViaDevice = bridgeDevice.Id
			acMeterSensors := events.ACMeterBaseSensors(acmeterDevice, msg.ACMeter)
			for i := range acMeterSensors {
				if i > 0 {
					acMeterSensors[i].Device = events.IdDevice(acmeterDevice)
				}
				sensors = append(sensors, acMeterSensors[i])
			}
		}

		if msg.Inverter.HasStorage && msg.ACMeter != nil {
			switches = append(switches, events.BatteryControlSwitches(inverterDevice)...)
			inputNumbers = append(inputNumbers, events.BatteryControlInputNumbers(inverterDevice)...)
		}

		ctx.Send(state.mqttActor, PublishHADiscovery{
			Sensors:      sensors,
			Switches:     switches,
			InputNumbers: inputNumbers,
		})
		state.behavior.Become(state.Done)

	case CommandErrorResponse:
		panic(msg.Error)
	default:
		state.logger.Trace("hadiscovery@info: default recv", "type", fmt.Sprintf("%T", msg))
	}
}
