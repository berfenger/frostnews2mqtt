package actor

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/scheduler"
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/internal/util/actorutil"
	"github.com/reugn/go-quartz/logger"
	"go.uber.org/zap"
)

const (
	MODBUS_ACTOR_ID = "modbus"
)

type ModbusActor struct {
	behavior            actor.Behavior
	stash               *actorutil.Stash
	scheduler           *scheduler.TimerScheduler
	inverter            domain.InverterDevice
	acMeter             domain.ACMeterModbusReader
	inverterDeviceInfo  *domain.InverterInfo
	acMeterDeviceInfo   *domain.ACMeterInfo
	logger              *zap.Logger
	readTimeout         time.Duration
	readTimeoutAfterSet time.Duration
}

type backgroundTaskResult struct {
	message any
	replyTo *actor.PID
}

func NewModbusActor(readTimeout time.Duration, readTimeoutAfterSet time.Duration, inverter domain.InverterDevice, acMeter domain.ACMeterModbusReader, logger *zap.Logger) *ModbusActor {
	act := &ModbusActor{
		inverter:            inverter,
		acMeter:             acMeter,
		inverterDeviceInfo:  nil,
		acMeterDeviceInfo:   nil,
		behavior:            actor.NewBehavior(),
		stash:               &actorutil.Stash{},
		logger:              actorutil.ActorLogger(domain.ACTOR_ID_MODBUS, logger),
		readTimeout:         readTimeout,
		readTimeoutAfterSet: readTimeoutAfterSet,
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
				logger.Warn(err)
			}
		}
		state.scheduler = scheduler.NewTimerScheduler(ctx)
		state.behavior.Become(state.DefaultReceive)
		state.stash.UnstashAll(ctx)
	case *actor.Restarting:
		//nolint errcheck
		state.acMeter.Close()
		//nolint errcheck
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
		if state.inverterDeviceInfo != nil && state.acMeterDeviceInfo != nil {
			ctx.Send(sender, domain.GetDevicesInfoResponse{
				Inverter: state.inverterDeviceInfo,
				ACMeter:  state.acMeterDeviceInfo,
			})
		} else {
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
			}).WithTimeout(state.readTimeout).PipeTo(ctx.Self())
			state.behavior.BecomeStacked(NewWaitingModbusActor(state, reflect.TypeOf(msg)).WaitingModbus)
		}
	case domain.GetPowerFlowRequest:
		state.logger.Debug("modbus@default: GetPowerFlowRequest")
		sender := actorutil.ForRequest(msg).ReplyTo(ctx)
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
		}).WithTimeout(state.readTimeout).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(NewWaitingModbusActor(state, reflect.TypeOf(msg)).WaitingModbus)
	case domain.GetInverterStateRequest:
		state.logger.Debug("modbus@default: GetInverterStateRequest")
		sender := actorutil.ForRequest(msg).ReplyTo(ctx)
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
		}).WithTimeout(state.readTimeout).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(NewWaitingModbusActor(state, reflect.TypeOf(msg)).WaitingModbus)
	case domain.GetStorageStateRequest:
		state.logger.Debug("modbus@default: GetStorageStateRequest")
		sender := actorutil.ForRequest(msg).ReplyTo(ctx)
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
		}).WithTimeout(state.readTimeout).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(NewWaitingModbusActor(state, reflect.TypeOf(msg)).WaitingModbus)
	case domain.GetStorageControlPowerFlowRequest:
		state.logger.Debug("modbus@default: GetStorageControlPowerFlowRequest")
		sender := actorutil.ForRequest(msg).ReplyTo(ctx)
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
		}).WithTimeout(state.readTimeout).PipeTo(ctx.Self())
		state.behavior.BecomeStacked(NewWaitingModbusActor(state, reflect.TypeOf(msg)).WaitingModbus)
	case domain.SetStorageControlRequest:
		state.logger.Debug("modbus@default: SetStorageControlRequest")
		sender := actorutil.ForRequest(msg).ReplyTo(ctx)
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
	case *actor.Stopping:
		//nolint errcheck
		state.inverter.Close()
		//nolint errcheck
		state.acMeter.Close()
	default:
		state.logger.Debug("modbus@default default recv", zap.String("type", fmt.Sprintf("%T", msg)))
	}
}

type waitingModbusActor struct {
	state           *ModbusActor
	requestType     reflect.Type
	otherRequesters []*actor.PID
}

func NewWaitingModbusActor(state *ModbusActor, requestType reflect.Type) *waitingModbusActor {
	return &waitingModbusActor{
		state:           state,
		requestType:     requestType,
		otherRequesters: []*actor.PID{},
	}
}

func (waitingState *waitingModbusActor) WaitingModbus(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case backgroundTaskResult:
		waitingState.state.logger.Debug("modbus@WaitingModbus backgroundTaskResult", zap.String("type", fmt.Sprintf("%T", msg.message)))
		ctx.Send(msg.replyTo, msg.message)
		for _, pid := range waitingState.otherRequesters {
			ctx.Send(pid, msg.message)
		}
		if devicesInfo, isGetDevicesInfo := msg.message.(domain.GetDevicesInfoResponse); isGetDevicesInfo {
			if devicesInfo.ResponseError == nil && devicesInfo.Inverter != nil && devicesInfo.ACMeter != nil {
				waitingState.state.inverterDeviceInfo = devicesInfo.Inverter
				waitingState.state.acMeterDeviceInfo = devicesInfo.ACMeter
			}
		}
		waitingState.state.behavior.UnbecomeStacked()
		waitingState.state.stash.UnstashAll(ctx)
	case *actor.Stopping:
		//nolint errcheck
		waitingState.state.inverter.Close()
		//nolint errcheck
		waitingState.state.acMeter.Close()
	default:
		if reflect.TypeOf(msg) == waitingState.requestType {
			sender := ctx.Sender()
			waitingState.otherRequesters = append(waitingState.otherRequesters, sender)
			return
		} else {
			// different message, process later
			waitingState.state.logger.Debug("modbus@WaitingModbus stash", zap.String("type", fmt.Sprintf("%T", msg)))
			waitingState.state.stash.Stash(ctx, msg)
		}
	}
}

func (state *ModbusActor) WaitingModbus(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case backgroundTaskResult:
		state.logger.Debug("modbus@WaitingModbus backgroundTaskResult", zap.String("type", fmt.Sprintf("%T", msg.message)))
		switch msg.message.(type) {
		case domain.SetStorageControlResponse:
			// transition to WaitingReadTimeout
			ctx.Send(msg.replyTo, msg.message)
			state.behavior.UnbecomeStacked()
			state.behavior.BecomeStacked(state.WaitingReadTimeout(ctx))
		default:
			ctx.Send(msg.replyTo, msg.message)
			state.behavior.UnbecomeStacked()
			state.stash.UnstashAll(ctx)
		}
	case *actor.Stopping:
		//nolint errcheck
		state.inverter.Close()
		//nolint errcheck
		state.acMeter.Close()
	default:
		state.logger.Debug("modbus@WaitingModbus stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *ModbusActor) WaitingReadTimeout(ctx actor.Context) func(actor.Context) {
	state.logger.Debug("modbus@WaitingReadTimeout: set read timeout after change")
	timeout := state.scheduler.RequestOnce(state.readTimeoutAfterSet, ctx.Self(), timeoutEnd{})
	return func(ctx actor.Context) {
		switch msg := ctx.Message().(type) {
		case domain.SetStorageControlRequest:
			state.logger.Debug("modbus@WaitingReadTimeout: SetStorageControlRequest")
			sender := ctx.Sender()
			timeout()
			ctx.RequestWithCustomSender(ctx.Self(), msg, sender)
			state.behavior.UnbecomeStacked()
		case domain.ActorHealthRequest:
			state.logger.Debug("modbus@WaitingReadTimeout: ActorHealthRequest")
			ctx.Respond(domain.ActorHealthResponse{
				Id:      MODBUS_ACTOR_ID,
				Healthy: true,
				State:   "readTimeout",
			})
		case timeoutEnd:
			state.logger.Debug("modbus@WaitingReadTimeout: timeout end")
			state.behavior.UnbecomeStacked()
			state.stash.UnstashAll(ctx)
		case domain.ActorRequest:
			state.logger.Debug("modbus@WaitingReadTimeout request stash", zap.String("type", fmt.Sprintf("%T", msg)))
			state.stash.Stash(ctx, msg)
		default:
			// ib stop or other event, end read timeout
			state.logger.Debug("modbus@WaitingReadTimeout exit", zap.String("type", fmt.Sprintf("%T", msg)))
			state.stash.Stash(ctx, msg)
			state.behavior.UnbecomeStacked()
			state.stash.UnstashAll(ctx)
		}
	}
}

func (a *ModbusActor) getDevicesInfo() (*domain.GetDevicesInfoResponse, error) {
	var inverter *domain.InverterInfo
	var acMeter *domain.ACMeterInfo
	var err error

	if a.inverter != nil {
		inverter, err = a.inverter.GetInfo()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	if a.acMeter != nil {
		if !a.acMeter.IsOpen() {
			err := a.acMeter.Open()
			if err != nil {
				logger.Warn(err)
			}
		}
		if a.acMeter.IsOpen() {
			acMeter, err = a.acMeter.GetInfo()
			if err != nil {
				logger.Error(err)
			}
		}
	}
	return &domain.GetDevicesInfoResponse{
		Inverter: inverter,
		ACMeter:  acMeter,
	}, nil
}

func (a *ModbusActor) getPowerFlow() (*domain.GetPowerFlowResponse, error) {
	var inverter *domain.InverterPowerFlow
	var acMeter *domain.ACMeterPowerFlow
	var err error

	if a.inverter != nil {
		inverter, err = a.inverter.GetPowerFlow()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	if a.acMeter != nil {
		if !a.acMeter.IsOpen() {
			err := a.acMeter.Open()
			if err != nil {
				logger.Warn(err)
			}
		}
		if a.acMeter.IsOpen() {
			acMeter, err = a.acMeter.GetPowerFlow()
			if err != nil {
				logger.Error(err)
			}
		}
	}
	return &domain.GetPowerFlowResponse{
		Inverter: inverter,
		ACMeter:  acMeter,
	}, nil
}

func (a *ModbusActor) getInverterState() (*domain.GetInverterStateResponse, error) {
	var state *domain.InverterState
	var err error

	if a.inverter != nil {
		state, err = a.inverter.GetState()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	var vendorInverterState *domain.VendorInverterState
	if vendorInverter, ok := a.inverter.(domain.VendorInverterDevice); ok {
		// additional vendor state fetch if needed
		vendorInverterState, err = vendorInverter.GetVendorState()
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}
	return &domain.GetInverterStateResponse{
		InverterState:       state,
		VendorInverterState: vendorInverterState,
	}, nil
}

func (a *ModbusActor) getInverterStorageState() (*domain.GetStorageStateResponse, error) {
	var state *domain.StorageState
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
	var state *domain.StorageState
	var meterFlow *domain.ACMeterPowerFlow
	var invFlow *domain.InverterPowerFlow
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

func (a *ModbusActor) setStorageControl(params domain.StorageControlParams) domain.SetStorageControlResponse {
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

type timeoutEnd struct{}

type MasterModbusActor struct {
	ActorProv func() *ModbusActor
	pid       *actor.PID
}

func (state *MasterModbusActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		modbusProps := actor.PropsFromProducer(func() actor.Actor {
			return state.ActorProv()
		})
		pid := ctx.Spawn(modbusProps)
		state.pid = pid
	case *actor.Terminated:
		panic(errors.New("could not initialize Modbus connection"))
	default:
		ctx.RequestWithCustomSender(state.pid, msg, ctx.Sender())
	}
}
