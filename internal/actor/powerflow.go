package actor

import (
	"fmt"
	"frostnews2mqtt/internal/config"
	"frostnews2mqtt/internal/events"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/asynkron/protoactor-go/scheduler"
	"github.com/sirupsen/logrus"
)

const (
	POWERFLOW_ACTOR_ID = "powerflow"
)

type PowerFlowActor struct {
	behavior  actor.Behavior
	stash     *Stash
	scheduler *scheduler.TimerScheduler

	modbusActor       *actor.PID
	config            *config.Config
	eventStream       *eventstream.EventStream
	hasStorage        bool
	currentStateCount uint
	stateCount        uint

	logger *logrus.Entry
}

type powerFlowTick struct {
}

func NewPowerFlowActor(config *config.Config, modbusActor *actor.PID, eventStream *eventstream.EventStream, logger *logrus.Logger) *PowerFlowActor {
	act := &PowerFlowActor{
		config:            config,
		modbusActor:       modbusActor,
		behavior:          actor.NewBehavior(),
		stash:             &Stash{},
		logger:            ActorLogger("powerflow", logger),
		eventStream:       eventStream,
		hasStorage:        false,
		currentStateCount: 2,
		stateCount:        2,
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

		if state.config.PowerFlowPollIntervalMillis > 0 {
			state.scheduler = scheduler.NewTimerScheduler(ctx)
			state.scheduler.RequestOnce(time.Duration(state.config.PowerFlowPollIntervalMillis)*time.Millisecond, ctx.Self(), powerFlowTick{})
		}

		PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, GetDevicesInfoRequest{}, 1*time.Second), func(err error) any {
			return CommandErrorResponse{
				Error: fmt.Sprintf("%s", err),
			}
		})
		state.behavior.Become(state.WaitingInfoReceive)
	case *actor.Restarting:
	default:
		state.logger.Trace("powerflow@starting: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

func (state *PowerFlowActor) DefaultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ActorHealthRequest:
		state.logger.Debug("powerflow@default: ActorHealthRequest")
		ctx.Respond(ActorHealthResponse{
			Id:      POWERFLOW_ACTOR_ID,
			Healthy: true,
			State:   "idle",
		})
	case powerFlowTick:
		state.logger.Debug("powerflow@default tick")
		// get power flow
		PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, GetPowerFlowRequest{}, 1*time.Second), func(err error) any {
			return CommandErrorResponse{
				Error: fmt.Sprintf("%s", err),
			}
		})
		// get Inverter/Storage states
		if state.currentStateCount == state.stateCount {
			state.currentStateCount = 0
			PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, GetInverterStateRequest{}, 1*time.Second), func(err error) any {
				return CommandErrorResponse{
					Error: fmt.Sprintf("%s", err),
				}
			})
			if state.hasStorage {
				PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, GetStorageStateRequest{}, 1*time.Second), func(err error) any {
					return CommandErrorResponse{
						Error: fmt.Sprintf("%s", err),
					}
				})
			}
		} else {
			state.currentStateCount++
		}

		// schedule next tick
		state.scheduler.RequestOnce(time.Duration(state.config.PowerFlowPollIntervalMillis)*time.Millisecond, ctx.Self(), powerFlowTick{})
		state.behavior.BecomeStacked(state.WaitingPFReceive)
	case GetInverterStateResponse:
		state.logger.Debug("powerflow@default GetInverterStateResponse")
		if msg.InverterState != nil {
			evs := events.InverterStateToUpdateEvents(msg.InverterState)
			for _, ev := range evs {
				state.eventStream.Publish(ev)
			}
		}
	case GetStorageStateResponse:
		state.logger.Debug("powerflow@default GetStorageStateResponse")
		if msg.StorageState != nil {
			evs := events.InverterStorageStateToUpdateEvents(msg.StorageState)
			for _, ev := range evs {
				state.eventStream.Publish(ev)
			}
		}
	case CommandErrorResponse:
		state.logger.Error("powerflow@default Error", "error", msg.Error)
	default:
		state.logger.Trace("powerflow@default: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

func (state *PowerFlowActor) WaitingPFReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case GetPowerFlowResponse:
		state.logger.Debug("powerflow@waiting GetPowerFlowResponse")
		// Inverter power flow
		if msg.Inverter != nil {
			evs := events.InverterPowerFlowToUpdateEvents(msg.Inverter)
			for _, ev := range evs {
				state.eventStream.Publish(ev)
			}
		}
		// ACMeter power flow
		if msg.ACMeter != nil {
			evs := events.ACMeterPowerFlowToUpdateEvents(msg.ACMeter)
			for _, ev := range evs {
				state.eventStream.Publish(ev)
			}
		}
		// House power
		if state.config.TrackHousePower && msg.Inverter != nil && msg.ACMeter != nil {
			evs := events.HousePowerUpdateEvents(msg.Inverter, msg.ACMeter)
			for _, ev := range evs {
				state.eventStream.Publish(ev)
			}
		}

		state.behavior.UnbecomeStacked()
		state.stash.UnstashAll(ctx)
	case CommandErrorResponse:
		state.logger.Error("powerflow@waiting GetPowerFlowResponse", "error", msg.Error)
		state.behavior.UnbecomeStacked()
		state.stash.UnstashAll(ctx)
	default:
		state.logger.Trace("powerflow@waiting: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

func (state *PowerFlowActor) WaitingInfoReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case GetDevicesInfoResponse:
		state.logger.Debug("powerflow@waitingInfo GetDevicesInfoResponse")
		state.hasStorage = msg.Inverter.HasStorage
		state.behavior.Become(state.DefaultReceive)
		state.stash.UnstashAll(ctx)
	case CommandErrorResponse:
		state.logger.Error("powerflow@waitingInfo GetDevicesInfoResponse", "error", msg.Error)
		state.behavior.Become(state.DefaultReceive)
		state.stash.UnstashAll(ctx)
	default:
		state.logger.Trace("powerflow@waitingInfo: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}
