package actor

import (
	"errors"
	"fmt"
	"frostnews2mqtt/internal/config"
	"frostnews2mqtt/internal/core/domain"
	"frostnews2mqtt/internal/core/events"
	"frostnews2mqtt/internal/core/port"
	"frostnews2mqtt/internal/util/actorutil"
	"frostnews2mqtt/pkg/sunspec_modbus"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/scheduler"
	"go.uber.org/zap"
)

type BatteryControlActor struct {
	actorutil.ActorWithStates
	scheduler          *scheduler.TimerScheduler
	stash              *actorutil.Stash
	modbusActor        *actor.PID
	mqttActor          *actor.PID
	config             *config.Config
	targetSOC          uint8
	control            port.BatteryChargeControlLogic
	tickIntervalMillis uint32
	logger             *zap.Logger
}

type batteryControlTick struct {
}

func NewBatteryControlActor(config *config.Config, modbusActor *actor.PID, mqttActor *actor.PID, control port.BatteryChargeControlLogic, logger *zap.Logger) *BatteryControlActor {
	act := &BatteryControlActor{
		config:             config,
		modbusActor:        modbusActor,
		mqttActor:          mqttActor,
		stash:              &actorutil.Stash{},
		logger:             actorutil.ActorLogger(domain.ACTOR_ID_BATTERY_CONTROL, logger),
		control:            control,
		tickIntervalMillis: config.BatteryControlConfig.ControlIntervalMillis,
		targetSOC:          100,
		ActorWithStates: actorutil.ActorWithStates{
			Behavior: actor.NewBehavior(),
		},
	}
	act.Become(BCStartingState{
		actor: act,
	})
	return act
}

func (state *BatteryControlActor) Receive(context actor.Context) {
	state.Behavior.Receive(context)
}

// Starting state

type BCStartingState struct {
	actorutil.ActorState
	actor *BatteryControlActor
}

func (state BCStartingState) Name() string {
	return "starting"
}

func (state BCStartingState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.actor.logger.Debug("battery_control@starting started")

		state.actor.scheduler = scheduler.NewTimerScheduler(ctx)

		actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.actor.modbusActor, domain.GetDevicesInfoRequest{}, 1*time.Second), func(err error) any {
			return domain.GetDevicesInfoResponse{
				ActorResponseMixIn: domain.ActorResponseMixIn{
					ResponseError: err,
				},
			}
		})
		state.actor.Become(BCWaitingInfoState{
			actor: state.actor,
		})
	case *actor.Restarting:
	default:
		state.actor.logger.Debug("battery_control@starting: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.actor.stash.Stash(ctx, msg)
	}
}

// Waiting info state

type BCWaitingInfoState struct {
	actorutil.ActorState
	actor *BatteryControlActor
}

func (state BCWaitingInfoState) Name() string {
	return "waitingInfo"
}

func (state BCWaitingInfoState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.GetDevicesInfoResponse:
		if msg.HasResponseError() {
			state.actor.logger.Error("battery_control@waitingInfo GetDevicesInfoResponse error", zap.Error(msg.GetResponseError()))
			panic(msg.GetResponseError())
		}
		state.actor.logger.Debug("battery_control@waitingInfo GetDevicesInfoResponse")
		if msg.ACMeter != nil && msg.Inverter != nil {
			// auto discover MaxGridImportPower if necessary
			if state.actor.control.MaxGridImportPower() <= 0 {
				state.actor.logger.Sugar().Infof("max_import_power not defined. assuming max rated power of inverter = %d", msg.Inverter.MaxRatedPowerWatt)
				state.actor.control.SetMaxGridImportPower(msg.Inverter.MaxRatedPowerWatt)
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
	default:
		state.actor.logger.Debug("battery_control@waitingInfo: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.actor.stash.Stash(ctx, msg)
	}
}

// Idle state

type BCIdleState struct {
	actorutil.ActorState
	actor *BatteryControlActor
}

func (state BCIdleState) Name() string {
	return "idle"
}

func (state BCIdleState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.ActorHealthRequest:
		state.actor.logger.Debug("battery_control@idle: ActorHealthRequest")
		ctx.Respond(domain.ActorHealthResponse{
			Id:      domain.ACTOR_ID_BATTERY_CONTROL,
			Healthy: true,
			State:   state.Name(),
		})
	case batteryControlTick:
	case domain.BatteryControlRequest:
		switch cmd := msg.(type) {
		case domain.BatteryControlHoldRequest:
			state.actor.logger.Sugar().Debugf("battery_control@idle: cmd hold %t", cmd.Enable)
			if cmd.Enable {
				state.actor.Become(NewBCHoldingState(state.actor).OnEnterAction(ctx))
			}
		case domain.BatteryControlChargeRequest:
			state.actor.logger.Sugar().Debugf("battery_control@idle: cmd charge %t", cmd.Enable)
			if cmd.Enable {
				state.actor.Become(NewBCChargingState(state.actor, false).OnEnterAction(ctx))
			}
		case domain.BatteryControlSetTargetSoCRequest:
			state.actor.logger.Sugar().Debugf("battery_control@idle: cmd setTargetSoC %d", cmd.TargetSoC)
			state.actor.setChargeTargetSoC(ctx, uint8(cmd.TargetSoC))
		}
	case domain.SetStorageControlResponse:
		// can be received after exiting holding or charging
		ctx.SetReceiveTimeout(0)
		if msg.HasResponseError() {
			state.actor.logger.Error("battery_control@idle: SetStorageControlResponse error", zap.Error(msg.GetResponseError()))
			panic(msg.GetResponseError())
		}
	default:
		state.actor.logger.Debug("battery_control@idle: recv", zap.String("type", fmt.Sprintf("%T", msg)))
	}
}

func (state BCIdleState) OnEnter(ctx actor.Context) BCIdleState {
	state.actor.updateSwitchState(ctx, false, false)
	state.actor.updateChargeTargetSoC(ctx, uint8(state.actor.targetSOC))
	return state
}

// Charging state

func NewBCChargingState(fromActor *BatteryControlActor, hold bool) BCChargingState {
	return BCChargingState{
		actor: fromActor,
		hold:  hold,
		params: sunspec_modbus.StorageControlParams{
			MinChargePowerWatt:    -1,
			MaxChargePowerWatt:    -1,
			MinDischargePowerWatt: -1,
			MaxDischargePowerWatt: -1,
			RevertTimeSeconds:     (fromActor.tickIntervalMillis / 1000) * 2,
		},
	}
}

type BCChargingState struct {
	actorutil.ActorState
	actor      *BatteryControlActor
	params     sunspec_modbus.StorageControlParams
	hold       bool
	cancelTick scheduler.CancelFunc
}

func (state BCChargingState) Name() string {
	return "charging"
}

func (state BCChargingState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.ActorHealthRequest:
		state.actor.logger.Debug("battery_control@charging: ActorHealthRequest")
		ctx.Respond(domain.ActorHealthResponse{
			Id:      domain.ACTOR_ID_BATTERY_CONTROL,
			Healthy: true,
			State:   state.Name(),
		})
	case batteryControlTick:
		// on tick, get power flow to decide what to do
		state.actor.logger.Debug("battery_control@charging batteryControlTick")
		state.actor.BecomeStacked(BCAwaitPowerFlowResponseState{
			actor: state.actor,
		}.OnEnterAction(ctx))
	case domain.SetStorageControlResponse:
		ctx.SetReceiveTimeout(0)
		if msg.HasResponseError() {
			state.actor.logger.Error("battery_control@charging SetStorageControlResponse error", zap.Error(msg.GetResponseError()))
			panic(msg.GetResponseError())
		}
		// on successful control response, schedule next control tick
		state.cancelTick = state.actor.scheduler.RequestOnce(time.Duration(state.actor.tickIntervalMillis)*time.Millisecond, ctx.Self(), batteryControlTick{})
		state.actor.Become(state)
	case domain.GetStorageControlPowerFlowResponse:
		if msg.HasResponseError() {
			state.actor.logger.Error("battery_control@charging GetStorageControlPowerFlowResponse error", zap.Error(msg.GetResponseError()))
			panic(msg.GetResponseError())
		}
		// Battery charge control loop
		controlResult := state.actor.control.Loop(state.params.MinChargePowerWatt,
			msg.StorageState, msg.ACMeterPowerFlow, msg.InverterPowerFlow, state.actor.targetSOC)

		if controlResult.Exit {
			state.actor.logger.Info("battery_control@charging: charge targetSoC is met. Turning off charging control.")
			state.Exit(ctx)
		} else {
			state.actor.logger.Sugar().Infof("battery_control@charging: set new charge power %f", controlResult.NewPowerValue)
			npv := controlResult.NewPowerValue
			// add bounds adjustment for safety
			if npv < 0 {
				npv = -1
			}
			state.params.MinChargePowerWatt = npv
			state.sendStorageControl(ctx)
			state.actor.Become(state)
		}
	case domain.BatteryControlRequest:
		switch cmd := msg.(type) {
		case domain.BatteryControlHoldRequest:
			state.actor.logger.Sugar().Debugf("battery_control@charging: cmd hold %t", cmd.Enable)
			if cmd.Enable && !state.hold {
				state.hold = true
				state.actor.Become(state)
				state.actor.updateHoldSwitchState(ctx, true)
			} else if !cmd.Enable && state.hold {
				state.hold = false
				state.actor.Become(state)
				state.actor.updateHoldSwitchState(ctx, false)
			}
		case domain.BatteryControlChargeRequest:
			state.actor.logger.Sugar().Debugf("battery_control@charging: cmd charge %t", cmd.Enable)
			if !cmd.Enable {
				state.Exit(ctx)
			}
		case domain.BatteryControlSetTargetSoCRequest:
			state.actor.logger.Sugar().Debugf("battery_control@charging: cmd setTargetSoC %d", cmd.TargetSoC)
			state.actor.setChargeTargetSoC(ctx, uint8(cmd.TargetSoC))
		}
	default:
		state.actor.logger.Debug("battery_control@charging: recv", zap.String("type", fmt.Sprintf("%T", msg)))
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
	state.actor.updateChargeSwitchState(ctx, false)
}

func (state BCChargingState) OnEnter(ctx actor.Context) BCChargingState {
	state.actor.updateChargeSwitchState(ctx, true)
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

func NewBCHoldingState(fromActor *BatteryControlActor) BCHoldingState {
	return BCHoldingState{
		actor: fromActor,
		params: sunspec_modbus.StorageControlParams{
			MinChargePowerWatt:    -1,
			MaxChargePowerWatt:    -1,
			MinDischargePowerWatt: -1,
			MaxDischargePowerWatt: 0,
			RevertTimeSeconds:     (fromActor.tickIntervalMillis / 1000) * 2,
		},
	}
}

type BCHoldingState struct {
	actorutil.ActorState
	actor      *BatteryControlActor
	params     sunspec_modbus.StorageControlParams
	cancelTick scheduler.CancelFunc
}

func (state BCHoldingState) Name() string {
	return "holding"
}

func (state BCHoldingState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.ActorHealthRequest:
		state.actor.logger.Debug("battery_control@holding: ActorHealthRequest")
		ctx.Respond(domain.ActorHealthResponse{
			Id:      domain.ACTOR_ID_BATTERY_CONTROL,
			Healthy: true,
			State:   state.Name(),
		})
	case batteryControlTick:
		state.actor.logger.Debug("battery_control@holding batteryControlTick")
		state.sendStorageControl(ctx)
	case domain.SetStorageControlResponse:
		ctx.SetReceiveTimeout(0)
		if msg.HasResponseError() {
			state.actor.logger.Error("battery_control@holding SetStorageControlResponse error", zap.Error(msg.GetResponseError()))
			panic(msg.GetResponseError())
		}
		// on successful control response, schedule next control tick
		state.cancelTick = state.actor.scheduler.RequestOnce(time.Duration(state.actor.tickIntervalMillis)*time.Millisecond, ctx.Self(), batteryControlTick{})
		state.actor.Become(state)
	case domain.BatteryControlRequest:
		switch cmd := msg.(type) {
		case domain.BatteryControlHoldRequest:
			state.actor.logger.Sugar().Debugf("battery_control@holding: cmd hold %t", cmd.Enable)
			if !cmd.Enable {
				state.Exit(ctx)
			}
		case domain.BatteryControlChargeRequest:
			state.actor.logger.Sugar().Debugf("battery_control@holding: cmd charge %t", cmd.Enable)
			if cmd.Enable {
				if state.cancelTick != nil {
					state.cancelTick()
				}
				state.actor.Become(NewBCChargingState(state.actor, true).OnEnterAction(ctx))
			}
		case domain.BatteryControlSetTargetSoCRequest:
			state.actor.logger.Sugar().Debugf("battery_control@holding: cmd setTargetSoC %d", cmd.TargetSoC)
			state.actor.setChargeTargetSoC(ctx, uint8(cmd.TargetSoC))
		}
	default:
		state.actor.logger.Debug("battery_control@holding: recv", zap.String("type", fmt.Sprintf("%T", msg)))
	}
}

func (state BCHoldingState) OnEnter(ctx actor.Context) BCHoldingState {
	state.actor.updateHoldSwitchState(ctx, true)
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
	actorutil.ActorState
	actor *BatteryControlActor
}

func (state BCDoneState) Name() string {
	return "done"
}

func (state BCDoneState) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case domain.ActorHealthRequest:
		state.actor.logger.Debug("battery_control@done: ActorHealthRequest")
		ctx.Respond(domain.ActorHealthResponse{
			Id:      domain.ACTOR_ID_BATTERY_CONTROL,
			Healthy: true,
			State:   state.Name(),
		})
	default:
	}
}

// Await modbus response state

type BCAwaitStorageControlResponseState struct {
	actorutil.ActorState
	actor *BatteryControlActor
}

func (state BCAwaitStorageControlResponseState) Name() string {
	return "awaitStorageControlReceive"
}

func (state BCAwaitStorageControlResponseState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.SetStorageControlResponse:
		ctx.SetReceiveTimeout(0)
		if msg.HasResponseError() {
			state.actor.logger.Error("battery_control@awaitStorageControlReceive: SetStorageControlResponse error", zap.Error(msg.GetResponseError()))
		} else {
			state.actor.logger.Debug("battery_control@awaitStorageControlReceive: SetStorageControlResponse", zap.Any("response", msg))
		}
		ctx.RequestWithCustomSender(ctx.Self(), msg, ctx.Sender())
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	case *actor.ReceiveTimeout:
		ctx.SetReceiveTimeout(0)
		state.actor.logger.Debug("battery_control@awaitStorageControlReceive: ReceiveTimeout")
		ctx.RequestWithCustomSender(ctx.Self(), domain.SetStorageControlResponse{
			ActorResponseMixIn: domain.ActorResponseMixIn{
				ResponseError: errors.New("receive timeout"),
			},
		}, ctx.Sender())
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	default:
		state.actor.logger.Debug("battery_control@awaitStorageControlReceive: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.actor.stash.Stash(ctx, msg)
	}
}

func (state BCAwaitStorageControlResponseState) OnEnterAction(ctx actor.Context, params sunspec_modbus.StorageControlParams) BCAwaitStorageControlResponseState {
	actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.actor.modbusActor,
		domain.SetStorageControlRequest{Params: params}, 2*time.Second),
		func(err error) any {
			return domain.SetStorageControlResponse{
				ActorResponseMixIn: domain.ActorResponseMixIn{
					ResponseError: err,
				},
			}
		})
	ctx.SetReceiveTimeout(2 * time.Second)
	return state
}

// Await powerflow response state

type BCAwaitPowerFlowResponseState struct {
	actorutil.ActorState
	actor *BatteryControlActor
}

func (state BCAwaitPowerFlowResponseState) Name() string {
	return "awaitPowerFlowReceive"
}

func (state BCAwaitPowerFlowResponseState) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.ActorHealthRequest:
		state.actor.logger.Debug("battery_control@awaitPowerFlowReceive: ActorHealthRequest")
		ctx.Respond(domain.ActorHealthResponse{
			Id:      domain.ACTOR_ID_BATTERY_CONTROL,
			Healthy: true,
			State:   state.Name(),
		})
	case domain.GetStorageControlPowerFlowResponse:
		ctx.SetReceiveTimeout(0)
		if msg.HasResponseError() {
			state.actor.logger.Error("battery_control@awaitPowerFlowReceive: GetStorageControlPowerFlowResponse error", zap.Error(msg.GetResponseError()))
		} else {
			state.actor.logger.Debug("battery_control@awaitPowerFlowReceive: GetStorageControlPowerFlowResponse", zap.Any("response", msg))
		}
		ctx.RequestWithCustomSender(ctx.Self(), msg, ctx.Sender())
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	case *actor.ReceiveTimeout:
		state.actor.logger.Debug("battery_control@awaitPowerFlowReceive: ReceiveTimeout")
		ctx.RequestWithCustomSender(ctx.Self(), domain.GetStorageControlPowerFlowResponse{
			ActorResponseMixIn: domain.ActorResponseMixIn{
				ResponseError: errors.New("receive timeout"),
			},
		}, ctx.Sender())
		state.actor.UnbecomeStacked()
		state.actor.stash.UnstashAll(ctx)
	default:
		state.actor.logger.Debug("battery_control@awaitPowerFlowReceive: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.actor.stash.Stash(ctx, msg)
	}
}

func (state BCAwaitPowerFlowResponseState) OnEnterAction(ctx actor.Context) BCAwaitPowerFlowResponseState {
	actorutil.PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.actor.modbusActor,
		domain.GetStorageControlPowerFlowRequest{}, 2*time.Second),
		func(err error) any {
			return domain.GetStorageControlPowerFlowResponse{
				ActorResponseMixIn: domain.ActorResponseMixIn{
					ResponseError: err,
				},
			}
		})
	ctx.SetReceiveTimeout(2 * time.Second)
	return state
}

// Other actor function helpers

func (state *BatteryControlActor) setChargeTargetSoC(ctx actor.Context, targetSOC uint8) {
	state.targetSOC = targetSOC
	state.updateChargeTargetSoC(ctx, targetSOC)
}

func (state *BatteryControlActor) updateSwitchState(ctx actor.Context, controlHold, controlCharge bool) {
	state.updateHoldSwitchState(ctx, controlHold)
	state.updateChargeSwitchState(ctx, controlCharge)
}

func (state *BatteryControlActor) updateHoldSwitchState(ctx actor.Context, switchState bool) {
	event := events.BatteryControlHoldSwitchUpdateEvents(switchState)
	state.sendEventToMQTT(ctx, event)
}

func (state *BatteryControlActor) updateChargeSwitchState(ctx actor.Context, switchState bool) {
	event := events.BatteryControlChargeSwitchUpdateEvents(switchState)
	state.sendEventToMQTT(ctx, event)
}

func (state *BatteryControlActor) updateChargeTargetSoC(ctx actor.Context, targetSoC uint8) {
	events := events.BatteryControlSetTargetSoCUpdateEvents(targetSoC)
	for _, ev := range events {
		state.sendEventToMQTT(ctx, ev)
	}
}

func (state *BatteryControlActor) sendEventToMQTT(ctx actor.Context, ev domain.SensorUpdateEvent) {
	ctx.Send(state.mqttActor, domain.PublishSensorUpdateRequest{
		Event: ev,
	})
}
