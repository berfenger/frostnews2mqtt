package actor

import (
	"errors"
	"fmt"
	"time"

	"github.com/berfenger/frostnews2mqtt/internal/config"
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/internal/util/actorutil"

	"github.com/asynkron/protoactor-go/actor"
	"go.uber.org/zap"
)

const (
	HADISCOVERY_ACTOR_ID = "hadiscovery"
)

type HADiscoveryActor struct {
	config             *config.Config
	behavior           actor.Behavior
	stash              *actorutil.Stash
	modbusActor        *actor.PID
	mqttActor          *actor.PID
	modbusActorHealthy bool
	mqttActorHealthy   bool
	healthyRecv        int

	logger *zap.Logger
}

func NewHADiscoveryActor(config *config.Config, modbusActor *actor.PID, mqttActor *actor.PID, logger *zap.Logger) *HADiscoveryActor {
	act := &HADiscoveryActor{
		config:      config,
		modbusActor: modbusActor,
		mqttActor:   mqttActor,
		behavior:    actor.NewBehavior(),
		stash:       &actorutil.Stash{},
		logger:      actorutil.ActorLogger(domain.ACTOR_ID_HA_DISCOVERY, logger),
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
		actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, domain.ActorHealthRequest{}, 2*time.Second), func(err error) any {
			return domain.ActorHealthResponse{
				Id:      domain.ACTOR_ID_MODBUS,
				Healthy: false,
			}
		})
		// MQTT Actor Request
		actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.mqttActor, domain.ActorHealthRequest{}, 2*time.Second), func(err error) any {
			return domain.ActorHealthResponse{
				Id:      domain.ACTOR_ID_MQTT,
				Healthy: false,
			}
		})
		state.behavior.Become(state.WaitingHealthyReceive)
	case *actor.Restarting:
	default:
		state.logger.Debug("hadiscovery@starting: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *HADiscoveryActor) WaitingHealthyReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.ActorHealthResponse:
		state.logger.Debug("hadiscovery@healthcheck ActorHealthResponse", zap.String("sender", msg.Id), zap.Bool("healthy", msg.Healthy))
		state.healthyRecv++
		if msg.Healthy {
			switch msg.Id {
			case domain.ACTOR_ID_MODBUS:
				state.modbusActorHealthy = true
			case domain.ACTOR_ID_MQTT:
				state.mqttActorHealthy = true
			}
		}
		if state.healthyRecv == 2 {

			if state.modbusActorHealthy && state.mqttActorHealthy {
				// Ask Modbus GetDevicesInfoRequest
				actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, domain.GetDevicesInfoRequest{}, 2*time.Second), func(err error) any {
					return domain.GetDevicesInfoResponse{
						ActorResponseMixIn: domain.ActorResponseMixIn{
							ResponseError: err,
						},
					}
				})
				state.behavior.Become(state.WaitingInfoReceive)
				state.stash.UnstashAll(ctx)
			} else {
				panic(errors.New("MQTT Actor or Modbus Actor are not healthy"))
			}
		}
	default:
		state.logger.Debug("hadiscovery@healthcheck: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *HADiscoveryActor) Done(ctx actor.Context) {

}

func (state *HADiscoveryActor) WaitingInfoReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.GetDevicesInfoResponse:
		if msg.HasResponseError() {
			panic(msg.GetResponseError())
		}
		state.logger.Debug("hadiscovery@info: GetDevicesInfoResponse", zap.Any("response", msg))

		var sensors []domain.GenericSensor
		var switches []domain.GenericSwitch
		var inputNumbers []domain.GenericInputNumber

		bridgeDevice := domain.BridgeDevice(state.config.MQTT.BaseTopic)
		bridgeSensors := domain.BridgeSensors(bridgeDevice)
		sensors = append(sensors, bridgeSensors...)

		inverterDevice := domain.InverterDevice(msg.Inverter)
		inverterDevice.ViaDevice = bridgeDevice.Id
		inverterSensors := domain.InverterBaseSensors(inverterDevice, msg.Inverter, state.config.MonitorConfig.TrackHousePower && msg.ACMeter != nil)
		for i := range inverterSensors {
			if i > 0 {
				inverterSensors[i].Device = domain.IdDevice(inverterDevice)
			}
			sensors = append(sensors, inverterSensors[i])
		}

		if msg.Inverter.HasStorage {
			storageSensors := domain.InverterStorageSensors(domain.IdDevice(inverterDevice))
			sensors = append(sensors, storageSensors...)
		}
		if msg.ACMeter != nil {
			acmeterDevice := domain.ACMeterDevice(msg.ACMeter)
			acmeterDevice.ViaDevice = bridgeDevice.Id
			acMeterSensors := domain.ACMeterBaseSensors(acmeterDevice, msg.ACMeter)
			for i := range acMeterSensors {
				if i > 0 {
					acMeterSensors[i].Device = domain.IdDevice(acmeterDevice)
				}
				sensors = append(sensors, acMeterSensors[i])
			}
		}

		if msg.Inverter.HasStorage && msg.ACMeter != nil {
			switches = append(switches, domain.BatteryControlSwitches(inverterDevice)...)
			inputNumbers = append(inputNumbers, domain.BatteryControlInputNumbers(inverterDevice)...)
		}

		ctx.Send(state.mqttActor, domain.PublishDiscoveryRequest{
			Sensors:      sensors,
			Switches:     switches,
			InputNumbers: inputNumbers,
		})
		state.behavior.Become(state.Done)

	default:
		state.logger.Debug("hadiscovery@info: default recv", zap.String("type", fmt.Sprintf("%T", msg)))
	}
}
