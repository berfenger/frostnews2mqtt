package actor

import (
	"fmt"
	"frostnews2mqtt/pkg/sunspec_modbus"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/reugn/go-quartz/logger"
	"github.com/sirupsen/logrus"
)

const (
	MODBUS_ACTOR_ID = "modbus"
)

type ModbusActor struct {
	behavior actor.Behavior
	stash    *Stash
	inverter sunspec_modbus.InverterModbusReader
	acMeter  sunspec_modbus.ACMeterModbusReader
	logger   *logrus.Entry
}

type backgroundTaskResult struct {
	message any
	replyTo *actor.PID
}

func NewModbusActor(inverter sunspec_modbus.InverterModbusReader, acMeter sunspec_modbus.ACMeterModbusReader, logger *logrus.Logger) *ModbusActor {
	act := &ModbusActor{
		inverter: inverter,
		acMeter:  acMeter,
		behavior: actor.NewBehavior(),
		stash:    &Stash{},
		logger:   ActorLogger("modbus", logger),
	}
	act.behavior.Become(act.StartingReceive)
	return act
}

func (state *ModbusActor) Receive(context actor.Context) {
	state.behavior.Receive(context)
}

func (state *ModbusActor) StartingReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.logger.Debug("modbus@starting started")
		if state.inverter != nil {
			err := state.inverter.Open()
			if err != nil {
				panic(err)
			}

		}
		if state.acMeter != nil {
			err := state.acMeter.Open()
			if err != nil {
				panic(err)
			}
		}
		state.behavior.Become(state.DefaultReceive)
		state.stash.UnstashAll(ctx)
	case *actor.Restarting:
		state.acMeter.Close()
		state.inverter.Close()
	default:
		state.logger.Trace("modbus@starting: stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

func (state *ModbusActor) DefaultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ActorHealthRequest:
		state.logger.Debug("modbus@default: ActorHealthRequest")
		ctx.Respond(ActorHealthResponse{
			Id:      MODBUS_ACTOR_ID,
			Healthy: true,
			State:   "idle",
		})
	case GetDevicesInfoRequest:
		state.logger.Debug("modbus@default: GetDevicesInfoRequest")
		sender := ctx.Sender()
		NewBackgroundTask(ctx, func() any {
			return OrCommandErrorResponse(state.getDevicesInfo)
		}).Map(func(a any) any {
			return backgroundTaskResult{
				message: a,
				replyTo: sender,
			}
		}).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case GetPowerFlowRequest:
		state.logger.Debug("modbus@default: GetPowerFlowRequest")
		sender := ctx.Sender()
		NewBackgroundTask(ctx, func() any {
			return OrCommandErrorResponse(state.getPowerFlow)
		}).Map(func(a any) any {
			return backgroundTaskResult{
				message: a,
				replyTo: sender,
			}
		}).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case GetInverterStateRequest:
		state.logger.Debug("modbus@default: GetInverterStateRequest")
		sender := ctx.Sender()
		NewBackgroundTask(ctx, func() any {
			return OrCommandErrorResponse(state.getInverterState)
		}).Map(func(a any) any {
			return backgroundTaskResult{
				message: a,
				replyTo: sender,
			}
		}).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case GetStorageStateRequest:
		state.logger.Debug("modbus@default: GetStorageStateRequest")
		sender := ctx.Sender()
		NewBackgroundTask(ctx, func() any {
			return OrCommandErrorResponse(state.getInverterStorageState)
		}).Map(func(a any) any {
			return backgroundTaskResult{
				message: a,
				replyTo: sender,
			}
		}).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case SetStorageControlRequest:
		state.logger.Debug("modbus@default: SetStorageControlRequest")
		sender := ctx.Sender()
		NewBackgroundTask(ctx, func() any {
			return state.setStorageControl(msg.params)
		}).Map(func(a any) any {
			return backgroundTaskResult{
				message: a,
				replyTo: sender,
			}
		}).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case GetStorageControlPowerFlowRequest:
		state.logger.Debug("modbus@default: GetStorageControlPowerFlowRequest")
		sender := ctx.Sender()
		NewBackgroundTask(ctx, func() any {
			return OrCommandErrorResponse(state.getStorageControlPowerFlow)
		}).Map(func(a any) any {
			return backgroundTaskResult{
				message: a,
				replyTo: sender,
			}
		}).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case *actor.Stopping:
		state.inverter.Close()
		state.acMeter.Close()
	default:
		state.logger.Trace("modbus@default default recv", "type", fmt.Sprintf("%T", msg))
	}
}

func (state *ModbusActor) WaitingModbus(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case backgroundTaskResult:
		state.logger.Debug("modbus@WaitingModbus backgroundTaskResult", "type", fmt.Sprintf("%T", msg.message))
		ctx.Send(msg.replyTo, msg.message)
		state.behavior.UnbecomeStacked()
		state.stash.UnstashAll(ctx)
	case *actor.Stopping:
		state.inverter.Close()
		state.acMeter.Close()
	default:
		state.logger.Trace("modbus@WaitingModbus stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

func (a *ModbusActor) getDevicesInfo() (*GetDevicesInfoResponse, error) {
	var inverter *sunspec_modbus.InverterInfo
	var acMeter *sunspec_modbus.ACMeterInfo
	var err error

	if a.inverter != nil {
		inverter, err = a.inverter.GetInfo()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	if a.acMeter != nil {
		acMeter, err = a.acMeter.GetInfo()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	return &GetDevicesInfoResponse{
		Inverter: inverter,
		ACMeter:  acMeter,
	}, nil
}

func (a *ModbusActor) getPowerFlow() (*GetPowerFlowResponse, error) {
	var inverter *sunspec_modbus.InverterPowerFlow
	var acMeter *sunspec_modbus.ACMeterPowerFlow
	var err error

	if a.inverter != nil {
		inverter, err = a.inverter.GetPowerFlow()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	if a.acMeter != nil {
		acMeter, err = a.acMeter.GetPowerFlow()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	return &GetPowerFlowResponse{
		Inverter: inverter,
		ACMeter:  acMeter,
	}, nil
}

func (a *ModbusActor) getInverterState() (*GetInverterStateResponse, error) {
	var state *sunspec_modbus.InverterState
	var err error

	if a.inverter != nil {
		state, err = a.inverter.GetState()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	return &GetInverterStateResponse{
		InverterState: state,
	}, nil
}

func (a *ModbusActor) getInverterStorageState() (*GetStorageStateResponse, error) {
	var state *sunspec_modbus.StorageState
	var err error

	if a.inverter != nil {
		state, err = a.inverter.GetStorageState()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	return &GetStorageStateResponse{
		StorageState: state,
	}, nil
}

func (a *ModbusActor) getStorageControlPowerFlow() (*GetStorageControlPowerFlowResponse, error) {
	var state *sunspec_modbus.StorageState
	var meterFlow *sunspec_modbus.ACMeterPowerFlow
	var invFlow *sunspec_modbus.InverterPowerFlow
	var err error

	if a.inverter != nil {
		state, err = a.inverter.GetStorageState()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
		invFlow, err = a.inverter.GetPowerFlow()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	if a.acMeter != nil {
		meterFlow, err = a.acMeter.GetPowerFlow()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	return &GetStorageControlPowerFlowResponse{
		StorageState:      state,
		ACMeterPowerFlow:  meterFlow,
		InverterPowerFlow: invFlow,
	}, nil
}

func (a *ModbusActor) setStorageControl(params sunspec_modbus.StorageControlParams) SetStorageControlResponse {
	if a.inverter != nil {
		err := a.inverter.SetStorageControl(params)
		if err != nil {
			logger.Error(err)
			return SetStorageControlResponse{
				Error: fmt.Sprintf("%s", err),
			}
		}
	}
	return SetStorageControlResponse{
		Error: "",
	}
}
