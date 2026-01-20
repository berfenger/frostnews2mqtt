package actor

import (
	"fmt"
	"time"

	"github.com/berfenger/frostnews2mqtt/internal/config"
	"github.com/berfenger/frostnews2mqtt/internal/core/component"
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/internal/util/actorutil"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/scheduler"
	"go.uber.org/zap"
)

type PowerFlowActor struct {
	behavior  actor.Behavior
	stash     *actorutil.Stash
	scheduler *scheduler.TimerScheduler

	modbusActor       *actor.PID
	mqttActor         *actor.PID
	config            *config.Config
	hasStorage        bool
	currentStateCount uint
	stateCount        uint
	readTimeout       time.Duration

	logger *zap.Logger
}

type powerFlowTick struct {
}

func NewPowerFlowActor(config *config.Config, modbusActor *actor.PID, mqttActor *actor.PID, logger *zap.Logger) *PowerFlowActor {
	act := &PowerFlowActor{
		config:            config,
		modbusActor:       modbusActor,
		mqttActor:         mqttActor,
		behavior:          actor.NewBehavior(),
		stash:             &actorutil.Stash{},
		logger:            actorutil.ActorLogger(domain.ACTOR_ID_POWERFLOW, logger),
		hasStorage:        false,
		currentStateCount: 2,
		stateCount:        2,
		readTimeout:       time.Duration(config.InverterModbusTcp.ReadTimeoutMillis) * time.Millisecond,
	}
	act.behavior.Become(act.StartingReceive)
	return act
}

func (state *PowerFlowActor) Receive(context actor.Context) {
	state.behavior.Receive(context)
}

func (state *PowerFlowActor) StartingReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.logger.Debug("powerflow@starting started")

		if state.config.MonitorConfig.PollIntervalMillis > 0 {
			state.scheduler = scheduler.NewTimerScheduler(ctx)
			state.scheduler.RequestOnce(time.Duration(state.config.MonitorConfig.PollIntervalMillis)*time.Millisecond, ctx.Self(), powerFlowTick{})
		}

		actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, domain.GetDevicesInfoRequest{}, state.readTimeout), func(err error) any {
			return domain.GetDevicesInfoResponse{
				ActorResponseMixIn: domain.ActorResponseMixIn{
					ResponseError: err,
				},
			}
		})
		state.behavior.Become(state.WaitingInfoReceive)
	case *actor.Restarting:
	default:
		state.logger.Debug("powerflow@starting: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *PowerFlowActor) DefaultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.ActorHealthRequest:
		state.logger.Debug("powerflow@default: ActorHealthRequest")
		ctx.Respond(domain.ActorHealthResponse{
			Id:      domain.ACTOR_ID_POWERFLOW,
			Healthy: true,
			State:   "idle",
		})
	case powerFlowTick:
		state.logger.Debug("powerflow@default tick")
		// get power flow
		actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, domain.GetPowerFlowRequest{}, state.readTimeout), func(err error) any {
			return domain.GetPowerFlowResponse{
				ActorResponseMixIn: domain.ActorResponseMixIn{
					ResponseError: err,
				},
			}
		})
		// get Inverter/Storage states
		if state.currentStateCount == state.stateCount {
			state.currentStateCount = 0
			actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, domain.GetInverterStateRequest{}, state.readTimeout), func(err error) any {
				return domain.GetInverterStateResponse{
					ActorResponseMixIn: domain.ActorResponseMixIn{
						ResponseError: err,
					},
				}
			})
			if state.hasStorage {
				actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, domain.GetStorageStateRequest{}, state.readTimeout), func(err error) any {
					return domain.GetStorageStateResponse{
						ActorResponseMixIn: domain.ActorResponseMixIn{
							ResponseError: err,
						},
					}
				})
			}
		} else {
			state.currentStateCount++
		}

		// schedule next tick
		state.scheduler.RequestOnce(time.Duration(state.config.MonitorConfig.PollIntervalMillis)*time.Millisecond, ctx.Self(), powerFlowTick{})
		state.behavior.BecomeStacked(state.WaitingPFReceive)
	case domain.GetInverterStateResponse:
		state.logger.Debug("powerflow@default GetInverterStateResponse")
		events := component.NewSensorUpdateEvents()
		if !msg.HasResponseError() && msg.InverterState != nil {
			events.AddInverterStateToUpdateEvents(msg.InverterState)
			if msg.VendorInverterState != nil {
				events.AddVendorInverterStateToUpdateEvents(msg.VendorInverterState)
			}
		}
		state.sendEventsToMQTT(ctx, events)
	case domain.GetStorageStateResponse:
		state.logger.Debug("powerflow@default GetStorageStateResponse")
		if !msg.HasResponseError() && msg.StorageState != nil {
			state.sendEventsToMQTT(ctx, component.NewSensorUpdateEvents().AddInverterStorageStateToUpdateEvents(msg.StorageState))
		}
	default:
		state.logger.Debug("powerflow@default: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *PowerFlowActor) WaitingPFReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.GetPowerFlowResponse:
		if msg.HasResponseError() {
			state.logger.Error("powerflow@waiting GetPowerFlowResponse error", zap.Error(msg.GetResponseError()))
			state.behavior.UnbecomeStacked()
			state.stash.UnstashAll(ctx)
			return
		}
		state.logger.Debug("powerflow@waiting GetPowerFlowResponse")
		events := component.NewSensorUpdateEvents()
		// Inverter power flow
		if msg.Inverter != nil {
			events.AddInverterPowerFlowEvents(msg.Inverter, state.hasStorage)
		}
		// ACMeter power flow
		if msg.ACMeter != nil {
			events.AddACMeterPowerFlowToUpdateEvents(msg.ACMeter)
		}
		// House power
		if state.config.MonitorConfig.TrackHousePower && msg.Inverter != nil {
			events.AddHousePowerUpdateEvents(msg.Inverter, msg.ACMeter)
		}
		state.sendEventsToMQTT(ctx, events)

		state.behavior.UnbecomeStacked()
		state.stash.UnstashAll(ctx)
	default:
		state.logger.Debug("powerflow@waiting: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *PowerFlowActor) WaitingInfoReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.GetDevicesInfoResponse:
		if msg.HasResponseError() {
			state.logger.Error("powerflow@waitingInfo GetDevicesInfoResponse", zap.Error(msg.GetResponseError()))
			state.behavior.Become(state.DefaultReceive)
			state.stash.UnstashAll(ctx)
			return
		}
		state.logger.Debug("powerflow@waitingInfo GetDevicesInfoResponse")
		state.hasStorage = msg.Inverter.HasStorage
		state.behavior.Become(state.DefaultReceive)
		state.stash.UnstashAll(ctx)
	default:
		state.logger.Debug("powerflow@waitingInfo: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *PowerFlowActor) sendEventToMQTT(ctx actor.Context, ev domain.SensorUpdateEvent) {
	ctx.Send(state.mqttActor, domain.PublishSensorUpdateRequest{
		Event: ev,
	})
}

func (state *PowerFlowActor) sendEventsToMQTT(ctx actor.Context, ev *component.SensorUpdateEvents) {
	ev.ForEach(func(event domain.SensorUpdateEvent) {
		state.sendEventToMQTT(ctx, event)
	})
}
