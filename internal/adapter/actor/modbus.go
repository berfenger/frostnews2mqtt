package actor

import (
	"fmt"
	"frostnews2mqtt/internal/core/domain"
	"frostnews2mqtt/internal/util/actorutil"
	"frostnews2mqtt/pkg/sunspec_modbus"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/reugn/go-quartz/logger"
	"go.uber.org/zap"
)

const (
	MODBUS_ACTOR_ID = "modbus"
)

type ModbusActor struct {
	behavior actor.Behavior
	stash    *actorutil.Stash
	inverter sunspec_modbus.InverterModbusReader
	acMeter  sunspec_modbus.ACMeterModbusReader
	logger   *zap.Logger
}

type backgroundTaskResult struct {
	message any
	replyTo *actor.PID
}

func NewModbusActor(inverter sunspec_modbus.InverterModbusReader, acMeter sunspec_modbus.ACMeterModbusReader, logger *zap.Logger) *ModbusActor {
	act := &ModbusActor{
		inverter: inverter,
		acMeter:  acMeter,
		behavior: actor.NewBehavior(),
		stash:    &actorutil.Stash{},
		logger:   actorutil.ActorLogger("modbus", logger),
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
		state.logger.Debug("modbus@starting: stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *ModbusActor) DefaultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domain.ActorHealthRequest:
		state.logger.Debug("modbus@default: ActorHealthRequest")
		ctx.Respond(domain.ActorHealthResponse{
			Id:      MODBUS_ACTOR_ID,
			Healthy: true,
			State:   "idle",
		})
	case domain.GetDevicesInfoRequest:
		state.logger.Debug("modbus@default: GetDevicesInfoRequest")
		sender := actorutil.ForRequest(msg).ReplyTo(ctx)

		actorutil.MapBackgroundTask(actorutil.NewBackgroundTask(ctx, state.getDevicesInfo),
			mapTaskResult[domain.GetDevicesInfoResponse](sender)).Recover(func(err error) backgroundTaskResult {
			return backgroundTaskResult{
				message: domain.GetDevicesInfoResponse{
					ActorResponseMixIn: domain.ActorResponseMixIn{
						ResponseError: err,
					},
				},
				replyTo: sender,
			}
		}).WithTimeout(2 * time.Second).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case domain.GetPowerFlowRequest:
		state.logger.Debug("modbus@default: GetPowerFlowRequest")
		sender := ctx.Sender()
		actorutil.MapBackgroundTask(actorutil.NewBackgroundTask(ctx, state.getPowerFlow),
			mapTaskResult[domain.GetPowerFlowResponse](sender)).Recover(func(err error) backgroundTaskResult {
			return backgroundTaskResult{
				message: domain.GetPowerFlowResponse{
					ActorResponseMixIn: domain.ActorResponseMixIn{
						ResponseError: err,
					},
				},
				replyTo: sender,
			}
		}).WithTimeout(2 * time.Second).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case domain.GetInverterStateRequest:
		state.logger.Debug("modbus@default: GetInverterStateRequest")
		sender := ctx.Sender()
		actorutil.MapBackgroundTask(actorutil.NewBackgroundTask(ctx, state.getInverterState),
			mapTaskResult[domain.GetInverterStateResponse](sender)).Recover(func(err error) backgroundTaskResult {
			return backgroundTaskResult{
				message: domain.GetInverterStateResponse{
					ActorResponseMixIn: domain.ActorResponseMixIn{
						ResponseError: err,
					},
				},
				replyTo: sender,
			}
		}).WithTimeout(2 * time.Second).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case domain.GetStorageStateRequest:
		state.logger.Debug("modbus@default: GetStorageStateRequest")
		sender := ctx.Sender()
		actorutil.MapBackgroundTask(actorutil.NewBackgroundTask(ctx, state.getInverterStorageState),
			mapTaskResult[domain.GetStorageStateResponse](sender)).Recover(func(err error) backgroundTaskResult {
			return backgroundTaskResult{
				message: domain.GetStorageStateResponse{
					ActorResponseMixIn: domain.ActorResponseMixIn{
						ResponseError: err,
					},
				},
				replyTo: sender,
			}
		}).WithTimeout(2 * time.Second).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case domain.SetStorageControlRequest:
		state.logger.Debug("modbus@default: SetStorageControlRequest")
		sender := ctx.Sender()
		actorutil.MapBackgroundTask(actorutil.NewBackgroundTaskNoError(ctx, func() *domain.SetStorageControlResponse {
			a := state.setStorageControl(msg.Params)
			return &a
		}),
			mapTaskResult[domain.SetStorageControlResponse](sender)).Recover(func(err error) backgroundTaskResult {
			return backgroundTaskResult{
				message: domain.SetStorageControlResponse{
					ActorResponseMixIn: domain.ActorResponseMixIn{
						ResponseError: err,
					},
				},
				replyTo: sender,
			}
		}).WithTimeout(2 * time.Second).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case domain.GetStorageControlPowerFlowRequest:
		state.logger.Debug("modbus@default: GetStorageControlPowerFlowRequest")
		sender := ctx.Sender()
		actorutil.MapBackgroundTask(actorutil.NewBackgroundTask(ctx, state.getStorageControlPowerFlow),
			mapTaskResult[domain.GetStorageControlPowerFlowResponse](sender)).Recover(func(err error) backgroundTaskResult {
			return backgroundTaskResult{
				message: domain.GetStorageControlPowerFlowResponse{
					ActorResponseMixIn: domain.ActorResponseMixIn{
						ResponseError: err,
					},
				},
				replyTo: sender,
			}
		}).WithTimeout(2 * time.Second).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(state.WaitingModbus)
	case *actor.Stopping:
		state.inverter.Close()
		state.acMeter.Close()
	default:
		state.logger.Debug("modbus@default default recv", zap.String("type", fmt.Sprintf("%T", msg)))
	}
}

func (state *ModbusActor) WaitingModbus(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case backgroundTaskResult:
		state.logger.Debug("modbus@WaitingModbus backgroundTaskResult", zap.String("type", fmt.Sprintf("%T", msg.message)))
		ctx.Send(msg.replyTo, msg.message)
		state.behavior.UnbecomeStacked()
		state.stash.UnstashAll(ctx)
	case *actor.Stopping:
		state.inverter.Close()
		state.acMeter.Close()
	default:
		state.logger.Debug("modbus@WaitingModbus stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (a *ModbusActor) getDevicesInfo() (*domain.GetDevicesInfoResponse, error) {
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
	return &domain.GetDevicesInfoResponse{
		Inverter: inverter,
		ACMeter:  acMeter,
	}, nil
}

func (a *ModbusActor) getPowerFlow() (*domain.GetPowerFlowResponse, error) {
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
	return &domain.GetPowerFlowResponse{
		Inverter: inverter,
		ACMeter:  acMeter,
	}, nil
}

func (a *ModbusActor) getInverterState() (*domain.GetInverterStateResponse, error) {
	var state *sunspec_modbus.InverterState
	var err error

	if a.inverter != nil {
		state, err = a.inverter.GetState()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	return &domain.GetInverterStateResponse{
		InverterState: state,
	}, nil
}

func (a *ModbusActor) getInverterStorageState() (*domain.GetStorageStateResponse, error) {
	var state *sunspec_modbus.StorageState
	var err error

	if a.inverter != nil {
		state, err = a.inverter.GetStorageState()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	return &domain.GetStorageStateResponse{
		StorageState: state,
	}, nil
}

func (a *ModbusActor) getStorageControlPowerFlow() (*domain.GetStorageControlPowerFlowResponse, error) {
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
	return &domain.GetStorageControlPowerFlowResponse{
		StorageState:      state,
		ACMeterPowerFlow:  meterFlow,
		InverterPowerFlow: invFlow,
	}, nil
}

func (a *ModbusActor) setStorageControl(params sunspec_modbus.StorageControlParams) domain.SetStorageControlResponse {
	if a.inverter != nil {
		err := a.inverter.SetStorageControl(params)
		if err != nil {
			logger.Error(err)
			return domain.SetStorageControlResponse{
				ActorResponseMixIn: domain.ActorResponseMixIn{
					ResponseError: err,
				},
			}
		}
	}
	return domain.SetStorageControlResponse{}
}

func mapTaskResult[T any](sender *actor.PID) func(t *T) *backgroundTaskResult {
	return func(t *T) *backgroundTaskResult {
		return &backgroundTaskResult{
			message: *t,
			replyTo: sender,
		}
	}
}
