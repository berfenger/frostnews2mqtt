package actor

import (
	"errors"
	"fmt"
	"frostnews2mqtt/internal/config"
	"frostnews2mqtt/internal/events"
	"frostnews2mqtt/pkg/sunspec_modbus"
	"math"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/asynkron/protoactor-go/scheduler"
	"github.com/sirupsen/logrus"
)

const (
	BATTERY_CONTROL_ACTOR_ID     = "battery_control"
	POWER_IMPORT_SAFETY_MARGIN_W = 200
)

type BatteryControlActorNew struct {
	ActorWithStates
	scheduler      *scheduler.TimerScheduler
	stash          *Stash
	modbusActor    *actor.PID
	config         *config.Config
	eventStream    *eventstream.EventStream
	maxImportPower uint32
	targetSOC      uint8

	logger *logrus.Entry
}

type batteryControlTick struct {
}

func NewBatteryControlActor(config *config.Config, modbusActor *actor.PID, eventStream *eventstream.EventStream, logger *logrus.Logger) *BatteryControlActorNew {
	act := &BatteryControlActorNew{
		config:         config,
		modbusActor:    modbusActor,
		stash:          &Stash{},
		logger:         ActorLogger("battery_control", logger),
		eventStream:    eventStream,
		maxImportPower: uint32(config.MaxImportPower),
		targetSOC:      100,
		ActorWithStates: ActorWithStates{
			behavior: actor.NewBehavior(),
		},
	}
	act.Become(BCStartingState{
		actor: act,
	})
	return act
}

func (state *BatteryControlActorNew) Receive(context actor.Context) {
	state.behavior.Receive(context)
}

// Starting state

type BCStartingState struct {
	ActorState
	actor *BatteryControlActorNew
}

func (state BCStartingState) Name() string {
	return "starting"
}

func (state BCStartingState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.actor.logger.Debug("battery_control@starting started")

		state.actor.scheduler = scheduler.NewTimerScheduler(ctx)

		PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.actor.modbusActor, GetDevicesInfoRequest{}, 1*time.Second), func(err error) any {
			return CommandErrorResponse{
				Error: fmt.Sprintf("%s", err),
			}
		})
		state.actor.Become(BCWaitingInfoState{
			actor: state.actor,
		})
	case *actor.Restarting:
	default:
		state.actor.logger.Tracef("battery_control@starting: stash %s", fmt.Sprintf("%T", msg))
		state.actor.stash.Stash(ctx, msg)
	}
}

// Waiting info state

type BCWaitingInfoState struct {
	ActorState
	actor *BatteryControlActorNew
}

func (state BCWaitingInfoState) Name() string {
	return "waitingInfo"
}

func (state BCWaitingInfoState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case GetDevicesInfoResponse:
		state.actor.logger.Debug("battery_control@waitingInfo GetDevicesInfoResponse")
		if msg.ACMeter != nil && msg.Inverter != nil {
			if state.actor.maxImportPower <= 0 {
				state.actor.logger.Infof("max_import_power not defined. assuming max rated power of inverter = %d", msg.Inverter.MaxRatedPowerWatt)
				state.actor.maxImportPower = uint32(msg.Inverter.MaxRatedPowerWatt)
			}
			state.actor.Become(BCIdleState{
				actor: state.actor,
			}.OnEnter(ctx))
		} else {
			state.actor.Become(BCDoneState{
				actor: state.actor,
			})
		}
		state.actor.stash.UnstashAll(ctx)
	case CommandErrorResponse:
		state.actor.logger.Errorf("battery_control@waitingInfo CommandErrorResponse: %s", msg.Error)
		panic(msg.Error)
	default:
		state.actor.logger.Tracef("battery_control@waitingInfo: stash %s", fmt.Sprintf("%T", msg))
		state.actor.stash.Stash(ctx, msg)
	}
}

// Idle state

type BCIdleState struct {
	ActorState
	actor *BatteryControlActorNew
}

func (state BCIdleState) Name() string {
	return "idle"
}

func (state BCIdleState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ActorHealthRequest:
		state.actor.logger.Debug("battery_control@idle: ActorHealthRequest")
		ctx.Respond(ActorHealthResponse{
			Id:      BATTERY_CONTROL_ACTOR_ID,
			Healthy: true,
			State:   state.Name(),
		})
	case batteryControlTick:
	case events.BatteryControlCommand:
		switch cmd := msg.(type) {
		case events.BatteryControlHold:
			state.actor.logger.Debugf("battery_control@idle: cmd hold %t", cmd.On)
			if cmd.On {
				state.actor.Become(NewBCHoldingState(state.actor).OnEnterAction(ctx))
			}
		case events.BatteryControlCharge:
			state.actor.logger.Debugf("battery_control@idle: cmd charge %t", cmd.On)
			if cmd.On {
				state.actor.Become(NewBCChargingState(state.actor, false).OnEnterAction(ctx))
			}
		case events.BatteryControlSetTargetSoC:
			state.actor.logger.Debugf("battery_control@idle: cmd setTargetSoC %d", cmd.TargetSoC)
			state.actor.setChargeTargetSoC(cmd.TargetSoC)
		}
	case CommandErrorResponse:
		// can be received after exiting holding or charging
		panic(errors.New(msg.Error))
	case SetStorageControlResponse:
		// can be received after exiting holding or charging
	default:
		state.actor.logger.Tracef("battery_control@idle: recv %s", fmt.Sprintf("%T", msg))
	}
}

func (state BCIdleState) OnEnter(ctx actor.Context) BCIdleState {
	state.actor.updateSwitchState(false, false)
	state.actor.updateChargeTargetSoC(uint8(state.actor.targetSOC))
	return state
}

// Charging state

func NewBCChargingState(fromActor *BatteryControlActorNew, hold bool) BCChargingState {
	return BCChargingState{
		actor: fromActor,
		hold:  hold,
		params: sunspec_modbus.StorageControlParams{
			MinChargePowerWatt:    -1,
			MaxChargePowerWatt:    -1,
			MinDischargePowerWatt: -1,
			MaxDischargePowerWatt: -1,
			RevertTimeSeconds:     uint32(fromActor.config.BatteryControlRevertTimeoutSeconds),
		},
	}
}

type BCChargingState struct {
	ActorState
	actor      *BatteryControlActorNew
	params     sunspec_modbus.StorageControlParams
	hold       bool
	cancelTick scheduler.CancelFunc
}

func (state BCChargingState) Name() string {
	return "charging"
}

func (state BCChargingState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ActorHealthRequest:
		state.actor.logger.Debug("battery_control@charging: ActorHealthRequest")
		ctx.Respond(ActorHealthResponse{
			Id:      BATTERY_CONTROL_ACTOR_ID,
			Healthy: true,
			State:   state.Name(),
		})
	case batteryControlTick:
		// on tick, get power flow to decide what to do
		state.actor.logger.Debug("battery_control@charging batteryControlTick")
		state.actor.BecomeStacked(BCAwaitPowerFlowResponseState{
			actor: state.actor,
		}.OnEnterAction(ctx))
	case SetStorageControlResponse:
		// on successful control response, schedule next control tick
		state.cancelTick = state.actor.scheduler.RequestOnce(time.Duration(state.params.RevertTimeSeconds)*time.Second/2, ctx.Self(), batteryControlTick{})
		state.actor.Become(state)
	case GetStorageControlPowerFlowResponse:
		// Battery charge control loop
		if msg.StorageState.StateOfCharge >= float64(state.actor.targetSOC) {
			// when battery SoC target is reached, disable force charge
			state.actor.logger.Infof("battery_control@charging: charge targetSoC is met. Turning off charging control.")
			state.Exit(ctx)
		} else {
			var prevPowerValue = state.params.MinChargePowerWatt
			var newPowerValue float64 = float64(prevPowerValue)
			if prevPowerValue == -1 {
				// temp disable, check if should be restarted
				houseConsumption := msg.InverterPowerFlow.ACPowerWatt + msg.ACMeterPowerFlow.CurrentPowerFlowWatt
				availablePower := float64(state.actor.maxImportPower) - houseConsumption - POWER_IMPORT_SAFETY_MARGIN_W
				if availablePower > float64(state.actor.maxImportPower)*0.33 {
					newPowerValue = float64(state.actor.maxImportPower) * 0.2
				} else {
					newPowerValue = -1
				}
			} else {
				// adjust charge power
				availablePower := float64(state.actor.maxImportPower) - msg.ACMeterPowerFlow.CurrentImportPowerWatt - POWER_IMPORT_SAFETY_MARGIN_W
				newPowerValue += math.Min(float64(state.actor.maxImportPower)*0.2, availablePower)
			}

			// check bounds
			// max value
			newPowerValue = math.Min(float64(msg.StorageState.MaxCapacityWatt), newPowerValue)
			// min value
			// if negative, temp disable
			if newPowerValue < 0 {
				newPowerValue = -1
			}
			state.actor.logger.Infof("battery_control@charging: set new charge power %f", newPowerValue)
			state.params.MinChargePowerWatt = int32(newPowerValue)
			state.sendStorageControl(ctx)
		}
	case events.BatteryControlCommand:
		switch cmd := msg.(type) {
		case events.BatteryControlHold:
			state.actor.logger.Debugf("battery_control@charging: cmd hold %t", cmd.On)
			if cmd.On && !state.hold {
				state.hold = true
				state.actor.Become(state)
				state.actor.updateHoldSwitchState(true)
			} else if !cmd.On && state.hold {
				state.hold = false
				state.actor.Become(state)
				state.actor.updateHoldSwitchState(false)
			}
		case events.BatteryControlCharge:
			state.actor.logger.Debugf("battery_control@charging: cmd charge %t", cmd.On)
			if !cmd.On {
				state.Exit(ctx)
			}
		case events.BatteryControlSetTargetSoC:
			state.actor.logger.Debugf("battery_control@charging: cmd setTargetSoC %d", cmd.TargetSoC)
			state.actor.setChargeTargetSoC(cmd.TargetSoC)
		}
	case CommandErrorResponse:
		state.actor.logger.Debugf("battery_control@charging CommandErrorResponse: %s", msg.Error)
		panic(errors.New(msg.Error))
	default:
		state.actor.logger.Tracef("battery_control@charging: recv %s", fmt.Sprintf("%T", msg))
	}
}

func (state BCChargingState) Exit(ctx actor.Context) {
	state.params.MinChargePowerWatt = -1
	if state.cancelTick != nil {
		state.cancelTick()
	}
	if state.hold {
		// next state is holding
		state.params.MaxDischargePowerWatt = 0
		state.actor.Become(BCHoldingState{
			actor:  state.actor,
			params: state.params,
		}.OnEnterAction(ctx))
	} else {
		// next state is idle
		state.actor.Become(BCIdleState{
			actor: state.actor,
		}.OnEnter(ctx))
		state.actor.BecomeStacked(BCAwaitStorageControlResponseState{
			actor: state.actor,
		}.OnEnterAction(ctx, state.params))
	}
	state.actor.updateChargeSwitchState(false)
}

func (state BCChargingState) OnEnter(ctx actor.Context) BCChargingState {
	state.actor.updateChargeSwitchState(true)
	return state
}

func (state BCChargingState) OnEnterAction(ctx actor.Context) BCChargingState {
	state.OnEnter(ctx)
	state.sendStorageControl(ctx)
	return state
}

func (state BCChargingState) sendStorageControl(ctx actor.Context) BCChargingState {
	state.actor.BecomeStacked(BCAwaitStorageControlResponseState{
		actor: state.actor,
	}.OnEnterAction(ctx, state.params))
	return state
}

// Holding state

func NewBCHoldingState(fromActor *BatteryControlActorNew) BCHoldingState {
	return BCHoldingState{
		actor: fromActor,
		params: sunspec_modbus.StorageControlParams{
			MinChargePowerWatt:    -1,
			MaxChargePowerWatt:    -1,
			MinDischargePowerWatt: -1,
			MaxDischargePowerWatt: 0,
			RevertTimeSeconds:     uint32(fromActor.config.BatteryControlRevertTimeoutSeconds),
		},
	}
}

type BCHoldingState struct {
	ActorState
	actor      *BatteryControlActorNew
	params     sunspec_modbus.StorageControlParams
	cancelTick scheduler.CancelFunc
}

func (state BCHoldingState) Name() string {
	return "holding"
}

func (state BCHoldingState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ActorHealthRequest:
		state.actor.logger.Debug("battery_control@holding: ActorHealthRequest")
		ctx.Respond(ActorHealthResponse{
			Id:      BATTERY_CONTROL_ACTOR_ID,
			Healthy: true,
			State:   state.Name(),
		})
	case batteryControlTick:
		state.actor.logger.Debug("battery_control@holding batteryControlTick")
		state.sendStorageControl(ctx)
	case SetStorageControlResponse:
		// on successful control response, schedule next control tick
		state.cancelTick = state.actor.scheduler.RequestOnce(time.Duration(state.params.RevertTimeSeconds)*time.Second/2, ctx.Self(), batteryControlTick{})
		state.actor.Become(state)
	case CommandErrorResponse:
		state.actor.logger.Debugf("battery_control@holding CommandErrorResponse: %s", msg.Error)
		panic(errors.New(msg.Error))
	case events.BatteryControlCommand:
		switch cmd := msg.(type) {
		case events.BatteryControlHold:
			state.actor.logger.Debugf("battery_control@holding: cmd hold %t", cmd.On)
			if !cmd.On {
				state.Exit(ctx)
			}
		case events.BatteryControlCharge:
			state.actor.logger.Debugf("battery_control@holding: cmd charge %t", cmd.On)
			if cmd.On {
				if state.cancelTick != nil {
					state.cancelTick()
				}
				state.actor.Become(NewBCChargingState(state.actor, true).OnEnterAction(ctx))
			}
		case events.BatteryControlSetTargetSoC:
			state.actor.logger.Debugf("battery_control@holding: cmd setTargetSoC %d", cmd.TargetSoC)
			state.actor.setChargeTargetSoC(cmd.TargetSoC)
		}
	default:
		state.actor.logger.Tracef("battery_control@holding: recv %s", fmt.Sprintf("%T", msg))
	}
}

func (state BCHoldingState) OnEnter(ctx actor.Context) BCHoldingState {
	state.actor.updateHoldSwitchState(true)
	return state
}

func (state BCHoldingState) OnEnterAction(ctx actor.Context) BCHoldingState {
	state.OnEnter(ctx)
	state.sendStorageControl(ctx)
	return state
}

func (state BCHoldingState) Exit(ctx actor.Context) {
	state.params.MaxDischargePowerWatt = -1
	if state.cancelTick != nil {
		state.cancelTick()
	}
	// next state is idle
	state.actor.Become(BCIdleState{
		actor: state.actor,
	}.OnEnter(ctx))
	state.sendStorageControl(ctx)
}

func (state BCHoldingState) sendStorageControl(ctx actor.Context) BCHoldingState {
	state.actor.BecomeStacked(BCAwaitStorageControlResponseState{
		actor: state.actor,
	}.OnEnterAction(ctx, state.params))
	return state
}

// Done state

type BCDoneState struct {
	ActorState
	actor *BatteryControlActorNew
}

func (state BCDoneState) Name() string {
	return "done"
}

func (state BCDoneState) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case ActorHealthRequest:
		state.actor.logger.Debug("battery_control@done: ActorHealthRequest")
		ctx.Respond(ActorHealthResponse{
			Id:      BATTERY_CONTROL_ACTOR_ID,
			Healthy: true,
			State:   state.Name(),
		})
	default:
	}
}

// Await modbus response state

type BCAwaitStorageControlResponseState struct {
	ActorState
	actor *BatteryControlActorNew
}

func (state BCAwaitStorageControlResponseState) Name() string {
	return "awaitStorageControlReceive"
}

func (state BCAwaitStorageControlResponseState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case SetStorageControlResponse:
		state.actor.logger.Debugf("battery_control@awaitStorageControlReceive: SetStorageControlResponse %+v", msg)
		if msg.Error == "" {
			ctx.RequestWithCustomSender(ctx.Self(), msg, ctx.Sender())
		} else {
			ctx.RequestWithCustomSender(ctx.Self(), CommandErrorResponse{Error: msg.Error}, ctx.Sender())
		}
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	case CommandErrorResponse:
		state.actor.logger.Debugf("battery_control@awaitStorageControlReceive: CommandErrorResponse %+v", msg)
		ctx.RequestWithCustomSender(ctx.Self(), msg, ctx.Sender())
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	case *actor.ReceiveTimeout:
		state.actor.logger.Debugf("battery_control@awaitStorageControlReceive: ReceiveTimeout %+v", msg)
		ctx.RequestWithCustomSender(ctx.Self(), CommandErrorResponse{Error: "receive timeout"}, ctx.Sender())
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	default:
		state.actor.logger.Tracef("battery_control@awaitStorageControlReceive: stash %s", fmt.Sprintf("%T", msg))
		state.actor.stash.Stash(ctx, msg)
	}
}

func (state BCAwaitStorageControlResponseState) OnEnterAction(ctx actor.Context, params sunspec_modbus.StorageControlParams) BCAwaitStorageControlResponseState {
	PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.actor.modbusActor,
		SetStorageControlRequest{params: params}, 2*time.Second),
		func(err error) any {
			return CommandErrorResponse{
				Error: fmt.Sprintf("%s", err),
			}
		})
	ctx.SetReceiveTimeout(2 * time.Second)
	return state
}

// Await powerflow response state

type BCAwaitPowerFlowResponseState struct {
	ActorState
	actor *BatteryControlActorNew
}

func (state BCAwaitPowerFlowResponseState) Name() string {
	return "awaitPowerFlowReceive"
}

func (state BCAwaitPowerFlowResponseState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case GetStorageControlPowerFlowResponse:
		state.actor.logger.Debugf("battery_control@awaitPowerFlowReceive: GetStorageControlPowerFlowResponse %+v", msg)
		ctx.RequestWithCustomSender(ctx.Self(), msg, ctx.Sender())
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	case CommandErrorResponse:
		state.actor.logger.Debugf("battery_control@awaitPowerFlowReceive: CommandErrorResponse %+v", msg)
		ctx.RequestWithCustomSender(ctx.Self(), msg, ctx.Sender())
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	case *actor.ReceiveTimeout:
		state.actor.logger.Debugf("battery_control@awaitStorageControlReceive: ReceiveTimeout %+v", msg)
		ctx.RequestWithCustomSender(ctx.Self(), CommandErrorResponse{Error: "receive timeout"}, ctx.Sender())
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	default:
		state.actor.logger.Tracef("battery_control@awaitPowerFlowReceive: stash %s", fmt.Sprintf("%T", msg))
		state.actor.stash.Stash(ctx, msg)
	}
}

func (state BCAwaitPowerFlowResponseState) OnEnterAction(ctx actor.Context) BCAwaitPowerFlowResponseState {
	PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.actor.modbusActor,
		GetStorageControlPowerFlowRequest{}, 2*time.Second),
		func(err error) any {
			return CommandErrorResponse{
				Error: fmt.Sprintf("%s", err),
			}
		})
	ctx.SetReceiveTimeout(2 * time.Second)
	return state
}

// Other actor function helpers

func (state *BatteryControlActorNew) setChargeTargetSoC(targetSOC uint8) {
	state.targetSOC = targetSOC
	state.updateChargeTargetSoC(targetSOC)
}

func (state *BatteryControlActorNew) updateSwitchState(controlHold, controlCharge bool) {
	state.updateHoldSwitchState(controlHold)
	state.updateChargeSwitchState(controlCharge)
}

func (state *BatteryControlActorNew) updateHoldSwitchState(switchState bool) {
	event := events.BatteryControlHoldSwitchUpdateEvents(switchState)
	state.eventStream.Publish(event)
}

func (state *BatteryControlActorNew) updateChargeSwitchState(switchState bool) {
	event := events.BatteryControlChargeSwitchUpdateEvents(switchState)
	state.eventStream.Publish(event)
}

func (state *BatteryControlActorNew) updateChargeTargetSoC(targetSoC uint8) {
	events := events.BatteryControlSetTargetSoCUpdateEvents(targetSoC)
	for _, ev := range events {
		state.eventStream.Publish(ev)
	}
}
