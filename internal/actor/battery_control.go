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
	BATTERY_CONTROL_ACTOR_ID = "battery_control"
)

type BatteryControlActor struct {
	behavior            actor.Behavior
	stash               *Stash
	scheduler           *scheduler.TimerScheduler
	modbusActor         *actor.PID
	config              *config.Config
	eventStream         *eventstream.EventStream
	maxImportPower      uint32
	currentControlState storageControlState

	logger *logrus.Entry
}

type batteryControlTick struct {
}

func NewBatteryControlActor(config *config.Config, modbusActor *actor.PID, eventStream *eventstream.EventStream, logger *logrus.Logger) *BatteryControlActor {
	act := &BatteryControlActor{
		config:         config,
		modbusActor:    modbusActor,
		behavior:       actor.NewBehavior(),
		stash:          &Stash{},
		logger:         ActorLogger("battery_control", logger),
		eventStream:    eventStream,
		maxImportPower: uint32(config.MaxImportPower),
		currentControlState: storageControlState{
			hold:      false,
			targetSoC: 100,
			params: sunspec_modbus.StorageControlParams{
				MinChargePowerWatt:    -1,
				MaxChargePowerWatt:    -1,
				MinDischargePowerWatt: -1,
				MaxDischargePowerWatt: -1,
				RevertTimeSeconds:     uint32(config.BatteryControlRevertTimeoutSeconds),
			},
		},
	}
	act.behavior.Become(act.StartingReceive)
	return act
}

func (state *BatteryControlActor) Receive(context actor.Context) {
	state.behavior.Receive(context)
}

func (state *BatteryControlActor) StartingReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.logger.Debug("battery_control@starting started")

		state.scheduler = scheduler.NewTimerScheduler(ctx)

		PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor, GetDevicesInfoRequest{}, 1*time.Second), func(err error) any {
			return CommandErrorResponse{
				Error: fmt.Sprintf("%s", err),
			}
		})
		state.behavior.Become(state.WaitingInfoReceive)
	case *actor.Restarting:
	default:
		state.logger.Trace("battery_control@starting: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

// wait for start commands
func (state *BatteryControlActor) DefaultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ActorHealthRequest:
		state.logger.Debug("battery_control@default: ActorHealthRequest")
		ctx.Respond(ActorHealthResponse{
			Id:      BATTERY_CONTROL_ACTOR_ID,
			Healthy: true,
		})
	case batteryControlTick:
	case events.BatteryControlCommand:
		switch cmd := msg.(type) {
		case events.BatteryControlHold:
			state.logger.Debug("battery_control@default: cmd hold", "on", cmd.On)
			if cmd.On {
				changed := state.currentControlState.setHold()
				if changed {
					state.currentControlState.hold = true
					state.sendControlRequestAndTransition(ctx)
				}
			}
		case events.BatteryControlCharge:
			state.logger.Debug("battery_control@default: cmd charge", "on", cmd.On)
			if cmd.On {
				changed := state.currentControlState.setForceCharge(uint32(state.config.MaxImportPower) / 2)
				if changed {
					state.sendControlRequestAndTransition(ctx)
				}
			}
		case events.BatteryControlSetTargetSoC:
			state.logger.Debug("battery_control@default: cmd setTargetSoC", "soc", cmd.TargetSoC)
			state.setChargeTargetSoC(cmd.TargetSoC)
		}
	default:
		state.logger.Trace("battery_control@default: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

// wait for SetStorageControlResponse, update states, and transition to control or default state
func (state *BatteryControlActor) ModbusRecvControlReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case SetStorageControlResponse:
		state.logger.Debug("battery_control@modbus_recv: SetStorageControlResponse", "response", msg)
		if msg.Error == "" {
			state.updateSwitchState(state.currentControlState.isHoldOn(), state.currentControlState.isChargeOn())
			if state.currentControlState.isHoldOn() || state.currentControlState.isChargeOn() {
				// control on
				state.scheduler.RequestOnce(time.Duration(state.currentControlState.revertSeconds())*time.Second/2, ctx.Self(), batteryControlTick{})
				state.behavior.Become(state.ControlReceive)
				state.logger.Debug("battery_control@modbus_recv: control ON")
			} else {
				// control off
				state.behavior.Become(state.DefaultReceive)
				state.logger.Debug("battery_control@modbus_recv: control OFF")
			}
		} else {
			panic(errors.New(msg.Error))
		}
	default:
		state.logger.Trace("battery_control@default: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

// wait for control tick or commands
func (state *BatteryControlActor) ControlReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ActorHealthRequest:
		state.logger.Debug("battery_control@control: ActorHealthRequest")
		ctx.Respond(ActorHealthResponse{
			Id:      BATTERY_CONTROL_ACTOR_ID,
			Healthy: true,
		})
	case batteryControlTick:
		state.logger.Debug("battery_control@control batteryControlTick")
		if state.currentControlState.isChargeOn() {
			// if charge, get power flow and transition to ControlReceivePowerFlowReceive
			PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor,
				GetStorageControlPowerFlowRequest{}, 2*time.Second),
				func(err error) any {
					return CommandErrorResponse{
						Error: fmt.Sprintf("%s", err),
					}
				})
			state.behavior.Become(state.ControlReceivePowerFlowReceive)
		} else if state.currentControlState.isHoldOn() {
			// if hold, send control params again
			state.sendControlRequestAndTransition(ctx)
		}
	case events.BatteryControlCommand:
		switch cmd := msg.(type) {
		case events.BatteryControlHold:
			state.logger.Debug("battery_control@control: cmd hold", "on", cmd.On)
			var changed = false
			if cmd.On {
				changed = state.currentControlState.setHold()
			} else {
				changed = state.currentControlState.disableHold()
			}
			if changed {
				state.sendControlRequestAndTransition(ctx)
			}
		case events.BatteryControlCharge:
			state.logger.Debug("battery_control@control: cmd charge", "on", cmd.On)
			var changed = false
			if cmd.On {
				changed = state.currentControlState.setForceCharge(uint32(state.config.MaxImportPower) / 2)
			} else {
				changed = state.currentControlState.disableForceCharge()
			}
			if changed {
				state.sendControlRequestAndTransition(ctx)
			}
		case events.BatteryControlSetTargetSoC:
			state.logger.Debug("battery_control@control: cmd setTargetSoC", "soc", cmd.TargetSoC)
			state.setChargeTargetSoC(cmd.TargetSoC)
		}
	default:
		state.logger.Trace("battery_control@control: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

// wait for GetStorageControlPowerFlowResponse and do control logic
func (state *BatteryControlActor) ControlReceivePowerFlowReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case GetStorageControlPowerFlowResponse:
		if state.currentControlState.isChargeOn() {
			if msg.StorageState.StateOfCharge > float64(state.currentControlState.targetSoC) {
				// when battery SoC target is reached, disable force charge
				ctx.Send(ctx.Self(), events.BatteryControlCharge{On: false})
				state.behavior.Become(state.ControlReceive)
			} else {
				// adjust charge power
				var newPowerValue = float64(state.currentControlState.getChargePower())
				newPowerValue += math.Min(float64(state.maxImportPower)*0.2, float64(state.maxImportPower)-200-msg.ACMeterPowerFlow.CurrentImportPowerWatt)
				newPowerValue = math.Min(float64(msg.StorageState.MaxCapacityWatt), newPowerValue)
				state.logger.Debug("battery_control@controlCharge: set new charge power", "power", newPowerValue)
				state.currentControlState.setForceCharge(uint32(newPowerValue))
				state.sendControlRequestAndTransition(ctx)
			}
		} else if state.currentControlState.isHoldOn() {
			state.sendControlRequestAndTransition(ctx)
		}
	default:
		state.logger.Trace("battery_control@controlCharge: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

// wait for initial GetDevicesInfoResponse to setup states
func (state *BatteryControlActor) WaitingInfoReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case GetDevicesInfoResponse:
		state.logger.Debug("battery_control@waitingInfo GetDevicesInfoResponse")
		if state.maxImportPower <= 0 {
			state.logger.Info(fmt.Sprintf("max_import_power not defined. assuming max rated power of inverter = %d", msg.Inverter.MaxRatedPowerWatt))
			state.maxImportPower = uint32(msg.Inverter.MaxRatedPowerWatt)
		}
		if msg.ACMeter != nil && msg.Inverter != nil {
			state.updateSwitchState(false, false)
			state.updateChargeTargetSoC(state.currentControlState.targetSoC)
			state.behavior.Become(state.DefaultReceive)
		} else {
			state.behavior.Become(state.DoneReceive)
		}
		state.stash.UnstashAll(ctx)
	case CommandErrorResponse:
		state.logger.Error("battery_control@waitingInfo CommandErrorResponse", "error", msg.Error)
		panic(msg.Error)
	default:
		state.logger.Trace("battery_control@waitingInfo: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

// if no storage or acMeter, stay idle
func (state *BatteryControlActor) DoneReceive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case ActorHealthRequest:
		state.logger.Debug("battery_control@done: ActorHealthRequest")
		ctx.Respond(ActorHealthResponse{
			Id:      BATTERY_CONTROL_ACTOR_ID,
			Healthy: true,
		})
	default:
	}
}

func (state *BatteryControlActor) sendControlRequestAndTransition(ctx actor.Context) {
	PipeToSelfWithRecover(ctx, ctx.RequestFuture(state.modbusActor,
		SetStorageControlRequest{params: state.currentControlState.params}, 2*time.Second),
		func(err error) any {
			return CommandErrorResponse{
				Error: fmt.Sprintf("%s", err),
			}
		})
	state.behavior.Become(state.ModbusRecvControlReceive)
}

func (state *BatteryControlActor) updateSwitchState(controlHold, controlCharge bool) {
	events := events.BatteryControlSwitchesUpdateEvents(controlHold, controlCharge)
	for _, ev := range events {
		state.eventStream.Publish(ev)
	}
}

func (state *BatteryControlActor) setChargeTargetSoC(targetSoC uint8) {
	state.currentControlState.targetSoC = targetSoC
	state.updateChargeTargetSoC(targetSoC)
}

func (state *BatteryControlActor) updateChargeTargetSoC(targetSoC uint8) {
	events := events.BatteryControlSetTargetSoCUpdateEvents(targetSoC)
	for _, ev := range events {
		state.eventStream.Publish(ev)
	}
}

type storageControlState struct {
	params    sunspec_modbus.StorageControlParams
	hold      bool
	targetSoC uint8
}

func (s *storageControlState) isHoldOn() bool {
	return s.params.MaxDischargePowerWatt == 0 || s.hold
}

func (s *storageControlState) isChargeOn() bool {
	return s.params.MinChargePowerWatt > 0
}

func (s *storageControlState) setForceCharge(power uint32) bool {
	prev := s.isChargeOn()
	s.params.MaxDischargePowerWatt = -1
	s.params.MinChargePowerWatt = int32(power)
	return !prev
}

func (s *storageControlState) setHold() bool {
	prev := s.isHoldOn()
	s.hold = true
	if s.isChargeOn() {
		return false
	} else {
		s.params.MaxDischargePowerWatt = 0
	}
	return !prev
}

func (s *storageControlState) disableForceCharge() bool {
	prev := s.isChargeOn()
	s.params.MinChargePowerWatt = -1
	if s.hold {
		s.params.MaxDischargePowerWatt = 0
		return true
	}
	return prev
}

func (s *storageControlState) disableHold() bool {
	prev := s.isHoldOn()
	s.hold = false
	if s.isChargeOn() {
		return false
	} else {
		s.params.MaxDischargePowerWatt = -1
		return prev
	}
}

func (s *storageControlState) revertSeconds() int {
	return int(s.params.RevertTimeSeconds)
}

func (s *storageControlState) getChargePower() uint32 {
	return uint32(s.params.MinChargePowerWatt)
}
