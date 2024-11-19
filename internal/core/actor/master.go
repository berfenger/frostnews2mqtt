package actor

import (
	"errors"
	"fmt"
	adactor "frostnews2mqtt/internal/adapter/actor"
	"frostnews2mqtt/internal/config"
	"frostnews2mqtt/internal/core/domain"
	. "frostnews2mqtt/internal/util/actorutil"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"go.uber.org/zap"
)

type MQTTActorProvider func(*eventstream.EventStream) *adactor.MQTTActor

type ModbusActorProvider func() *adactor.ModbusActor

type MasterOfPuppetsActor struct {
	config   config.Config
	behavior actor.Behavior
	stash    *Stash

	currentHealthCheck  healthCheckResult
	eventStream         *eventstream.EventStream
	modbusActor         *actor.PID
	mqttActor           *actor.PID
	powerFlowActor      *actor.PID
	batteryControlActor *actor.PID
	modbusActorProvider ModbusActorProvider
	mqttActorProvider   MQTTActorProvider
	logger              *zap.Logger
	testMQTT            bool
}

type healthCheckResult struct {
	modbusActorHealthy    bool
	mqttActorHealthy      bool
	powerFlowActorHealthy bool
	checksReceived        int
	respondTo             *actor.PID
}

func NewMasterOfPuppetsActor(config config.Config, modbusActorProvider ModbusActorProvider, mqttActorProvider MQTTActorProvider, logger *zap.Logger) *MasterOfPuppetsActor {
	act := &MasterOfPuppetsActor{
		config:              config,
		behavior:            actor.NewBehavior(),
		stash:               &Stash{},
		logger:              ActorLogger("master", logger),
		eventStream:         &eventstream.EventStream{},
		modbusActorProvider: modbusActorProvider,
		mqttActorProvider:   mqttActorProvider,
	}
	act.behavior.Become(act.StartingReceive)
	return act
}

func (state *MasterOfPuppetsActor) Receive(context actor.Context) {
	state.behavior.Receive(context)
}

func (state *MasterOfPuppetsActor) StartingReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.logger.Debug("master@starting started")

		state.currentHealthCheck = healthCheckResult{}
		state.currentHealthCheck.reset()

		// start Modbus child
		modbusActorPID, err := state.startModbusActor(ctx)
		if err != nil {
			panic(err)
		}
		state.modbusActor = modbusActorPID

		// start MQTT child
		mqttActorPID, err := state.startMQTTActor(ctx)
		if err != nil {
			panic(err)
		}
		state.mqttActor = mqttActorPID

		// start PowerFlow child
		powerFlowActorPID, err := state.startPowerFlowActor(ctx)
		if err != nil {
			panic(err)
		}
		state.powerFlowActor = powerFlowActorPID

		// start BatteryControl child
		batteryControlActorPID, err := state.startBatteryControlActor(ctx)
		if err != nil {
			panic(err)
		}
		state.batteryControlActor = batteryControlActorPID

		// start HA Discovery
		if state.config.MQTT.HADiscoveryEnable {
			_, err := state.startHADiscoveryActor(ctx)
			if err != nil {
				panic(err)
			}
		}

		state.behavior.Become(state.DefaultReceive)
		state.stash.UnstashAll(ctx)
	default:
		state.logger.Debug("master@starting stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *MasterOfPuppetsActor) DefaultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.ActorHealthRequest:
		state.logger.Debug("master@default ActorHealthRequest")
		state.currentHealthCheck.reset()
		state.currentHealthCheck.respondTo = ctx.Sender()
		// Modbus Actor Request
		PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, domain.ActorHealthRequest{}, 500*time.Millisecond), func(err error) any {
			return domain.ActorHealthResponse{
				Id:      domain.ACTOR_ID_MODBUS,
				Healthy: false,
			}
		})
		// MQTT Actor Request
		PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.mqttActor, domain.ActorHealthRequest{}, 500*time.Millisecond), func(err error) any {
			return domain.ActorHealthResponse{
				Id:      domain.ACTOR_ID_MQTT,
				Healthy: false,
			}
		})
		// PowerFlow Actor Request
		PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.powerFlowActor, domain.ActorHealthRequest{}, 500*time.Millisecond), func(err error) any {
			return domain.ActorHealthResponse{
				Id:      domain.ACTOR_ID_POWERFLOW,
				Healthy: false,
			}
		})

		ctx.SetReceiveTimeout(1 * time.Second)

		state.behavior.BecomeStacked(state.HealthCheckReceive)
	case adactor.ParsedCommand:
		// redirect parsedCommand to actor
		state.logger.Debug("master@default parsedCommand", zap.Any("command", msg.Command))
		if msg.Command != nil {
			cmd, err := ParsedMQTTCommandToCommand(*msg.Command)
			if err == nil && cmd != nil {
				switch pcmd := cmd.(type) {
				case domain.BatteryControlRequest:
					ctx.Send(state.batteryControlActor, pcmd)
				}
			}
		}
	case *actor.Terminated:
		// if some actor fails on boot, terminate
		if msg.Who.Id == fmt.Sprintf("%s/%s", domain.ACTOR_ID_MASTER, domain.ACTOR_ID_MODBUS) {
			state.logger.Error("master@default modbus error")
			panic(errors.New("modbus terminated"))
		}
	default:
		state.logger.Debug("master@default stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *MasterOfPuppetsActor) HealthCheckReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.ReceiveTimeout:
		// if some actor does not respond to healthCheck, assume not healthy
		state.currentHealthCheck.respond(ctx)
		state.behavior.UnbecomeStacked()
		state.stash.UnstashAll(ctx)
	case domain.ActorHealthResponse:
		state.logger.Debug("master@healthcheck ActorHealthResponse", zap.String("sender", msg.Id), zap.Bool("healthy", msg.Healthy))
		state.currentHealthCheck.checksReceived++
		if msg.Healthy {
			if msg.Id == domain.ACTOR_ID_MODBUS {
				state.currentHealthCheck.modbusActorHealthy = true
			} else if msg.Id == domain.ACTOR_ID_MQTT {
				state.currentHealthCheck.mqttActorHealthy = true
			} else if msg.Id == domain.ACTOR_ID_POWERFLOW {
				state.currentHealthCheck.powerFlowActorHealthy = true
			}
		}
		if state.currentHealthCheck.allReceived() {

			state.currentHealthCheck.respond(ctx)

			state.behavior.UnbecomeStacked()
			state.stash.UnstashAll(ctx)
		} else {
			ctx.SetReceiveTimeout(1 * time.Second)
		}
	default:
		state.logger.Debug("master@healthcheck stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *MasterOfPuppetsActor) startModbusActor(ctx actor.Context) (*actor.PID, error) {

	supervisor := actor.NewExponentialBackoffStrategy(10*time.Second, 1*time.Second)

	modbusProps := actor.PropsFromProducer(func() actor.Actor {
		return state.modbusActorProvider()
	}, actor.WithSupervisor(supervisor))
	modbusActorPID, err := ctx.SpawnNamed(modbusProps, domain.ACTOR_ID_MODBUS)
	if err != nil {
		return nil, err
	}

	return modbusActorPID, nil
}

func (state *MasterOfPuppetsActor) startPowerFlowActor(ctx actor.Context) (*actor.PID, error) {

	decider := func(reason interface{}) actor.Directive {
		log.Printf("handling failure for child. reason: %v", reason)
		return actor.RestartDirective
	}
	supervisor := actor.NewAllForOneStrategy(1, 10*time.Second, decider)

	powerFlowProps := actor.PropsFromProducer(func() actor.Actor {
		return NewPowerFlowActor(&state.config, state.modbusActor, state.eventStream, state.logger)
	}, actor.WithSupervisor(supervisor))
	powerFlowActorPID, err := ctx.SpawnNamed(powerFlowProps, domain.ACTOR_ID_POWERFLOW)
	if err != nil {
		return nil, err
	}

	return powerFlowActorPID, nil
}

func (state *MasterOfPuppetsActor) startHADiscoveryActor(ctx actor.Context) (*actor.PID, error) {

	decider := func(reason interface{}) actor.Directive {
		log.Printf("handling failure for child. reason: %v", reason)
		return actor.RestartDirective
	}
	supervisor := actor.NewOneForOneStrategy(1, 10*time.Second, decider)

	haDiscProps := actor.PropsFromProducer(func() actor.Actor {
		return NewHADiscoveryActor(&state.config, state.modbusActor, state.mqttActor, state.logger)
	}, actor.WithSupervisor(supervisor))
	haDiscPID, err := ctx.SpawnNamed(haDiscProps, HADISCOVERY_ACTOR_ID)
	if err != nil {
		return nil, err
	}

	return haDiscPID, nil
}

func (state *MasterOfPuppetsActor) startMQTTActor(ctx actor.Context) (*actor.PID, error) {

	supervisor := actor.NewExponentialBackoffStrategy(10*time.Second, 1*time.Second)

	mqttProps := actor.PropsFromProducer(func() actor.Actor {
		return state.mqttActorProvider(state.eventStream)
	}, actor.WithSupervisor(supervisor))
	mqttActorPID, err := ctx.SpawnNamed(mqttProps, domain.ACTOR_ID_MQTT)
	if err != nil {
		return nil, err
	}

	return mqttActorPID, nil
}

func (state *MasterOfPuppetsActor) startBatteryControlActor(ctx actor.Context) (*actor.PID, error) {

	decider := func(reason interface{}) actor.Directive {
		log.Printf("handling failure for child. reason: %v", reason)
		return actor.RestartDirective
	}
	supervisor := actor.NewOneForOneStrategy(1, 10*time.Second, decider)

	battControlProps := actor.PropsFromProducer(func() actor.Actor {
		return NewBatteryControlActor(&state.config, state.modbusActor, state.eventStream, state.logger)
	}, actor.WithSupervisor(supervisor))
	battControlPID, err := ctx.SpawnNamed(battControlProps, domain.ACTOR_ID_BATTERY_CONTROL)
	if err != nil {
		return nil, err
	}

	return battControlPID, nil
}

func (state *healthCheckResult) reset() {
	state.modbusActorHealthy = false
	state.mqttActorHealthy = false
	state.powerFlowActorHealthy = false
	state.checksReceived = 0
}

func (state *healthCheckResult) allReceived() bool {
	return state.checksReceived == 3
}

func (state *healthCheckResult) allHealthy() bool {
	return state.modbusActorHealthy && state.mqttActorHealthy && state.powerFlowActorHealthy
}

func (state *healthCheckResult) respond(ctx actor.Context) {
	resp := domain.ActorHealthResponse{
		Id:      "master",
		Healthy: state.allHealthy(),
	}
	if state.respondTo != nil {
		ctx.Send(state.respondTo, resp)
	}
}
