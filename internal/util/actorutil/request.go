package actorutil

import (
	"frostnews2mqtt/internal/core/domain"

	"github.com/asynkron/protoactor-go/actor"
)

type forRequest struct {
	req domain.ActorRequest
}

type ExtendedRequest interface {
	Respond(ctx actor.Context, resp domain.ActorResponse)
	ReplyTo(ctx actor.Context) *actor.PID
}

func ForRequest(r domain.ActorRequest) ExtendedRequest {
	return forRequest{req: r}
}

func (r forRequest) Respond(ctx actor.Context, resp domain.ActorResponse) {
	if r.req.ReplyTo() != nil {
		ctx.Send((*actor.PID)(r.req.ReplyTo()), resp)
	} else {
		ctx.Respond(resp)
	}
}

func (r forRequest) ReplyTo(ctx actor.Context) *actor.PID {
	if r.req.ReplyTo() != nil {
		return (*actor.PID)(r.req.ReplyTo())
	}
	return ctx.Sender()
}

// func GetDevicesInfoTask(ctx actor.Context, srv port.InverterService, timeout time.Duration) {
// 	NewBackgroundTaskNoError(ctx, func() *GetDevicesInfoResponse {
// 		info, err := srv.GetDevicesInfo(context.Background())
// 		if err != nil {
// 			return &GetDevicesInfoResponse{
// 				ActorResponseMixIn: domain.ActorResponseMixIn{
// 					ResponseError: err,
// 				},
// 			}
// 		}
// 		return &GetDevicesInfoResponse{
// 			Inverter: info.Inverter,
// 			ACMeter:  info.ACMeter,
// 		}
// 	}).WithTimeout(2 * time.Second).OnError(func(err error) {
// 		ctx.Send(ctx.Self(), GetDevicesInfoResponse{
// 			ActorResponseMixIn: domain.ActorResponseMixIn{
// 				ResponseError: err,
// 			},
// 		})
// 	}).PipeTo(ctx.Self())
// }

// func GetPowerFlowTask(ctx actor.Context, srv port.InverterService, timeout time.Duration) {
// 	NewBackgroundTaskNoError(ctx, func() *GetPowerFlowResponse {
// 		info, err := srv.GetPowerFlow(context.Background())
// 		if err != nil {
// 			return &GetPowerFlowResponse{
// 				ActorResponseMixIn: domain.ActorResponseMixIn{
// 					ResponseError: err,
// 				},
// 			}
// 		}
// 		return &GetPowerFlowResponse{
// 			Inverter: info.Inverter,
// 			ACMeter:  info.ACMeter,
// 		}
// 	}).WithTimeout(2 * time.Second).OnError(func(err error) {
// 		ctx.Send(ctx.Self(), GetPowerFlowResponse{
// 			ActorResponseMixIn: domain.ActorResponseMixIn{
// 				ResponseError: err,
// 			},
// 		})
// 	}).PipeTo(ctx.Self())
// }

// func GetInverterStateTask(ctx actor.Context, srv port.InverterService, timeout time.Duration) {
// 	NewBackgroundTaskNoError(ctx, func() *GetInverterStateResponse {
// 		info, err := srv.GetInverterState(context.Background())
// 		if err != nil {
// 			return &GetInverterStateResponse{
// 				ActorResponseMixIn: domain.ActorResponseMixIn{
// 					ResponseError: err,
// 				},
// 			}
// 		}
// 		return &GetInverterStateResponse{
// 			InverterState: (*sunspec_modbus.InverterState)(info),
// 		}
// 	}).WithTimeout(2 * time.Second).OnError(func(err error) {
// 		ctx.Send(ctx.Self(), GetInverterStateResponse{
// 			ActorResponseMixIn: domain.ActorResponseMixIn{
// 				ResponseError: err,
// 			},
// 		})
// 	}).PipeTo(ctx.Self())
// }

// func GetStorageStateTask(ctx actor.Context, srv port.InverterService, timeout time.Duration) {
// 	NewBackgroundTaskNoError(ctx, func() *GetStorageStateResponse {
// 		info, err := srv.GetStorageState(context.Background())
// 		if err != nil {
// 			return &GetStorageStateResponse{
// 				ActorResponseMixIn: domain.ActorResponseMixIn{
// 					ResponseError: err,
// 				},
// 			}
// 		}
// 		return &GetStorageStateResponse{
// 			StorageState: (*sunspec_modbus.StorageState)(info),
// 		}
// 	}).WithTimeout(2 * time.Second).OnError(func(err error) {
// 		ctx.Send(ctx.Self(), GetStorageStateResponse{
// 			ActorResponseMixIn: domain.ActorResponseMixIn{
// 				ResponseError: err,
// 			},
// 		})
// 	}).PipeTo(ctx.Self())
// }

// func GetStorageControlPowerFlowTask(ctx actor.Context, srv port.InverterService, timeout time.Duration) {
// 	NewBackgroundTaskNoError(ctx, func() *GetStorageControlPowerFlowResponse {
// 		info, err := srv.GetStorageControlPowerFlow(context.Background())
// 		if err != nil {
// 			return &GetStorageControlPowerFlowResponse{
// 				ActorResponseMixIn: domain.ActorResponseMixIn{
// 					ResponseError: err,
// 				},
// 			}
// 		}
// 		return &GetStorageControlPowerFlowResponse{
// 			StorageState:      info.StorageState,
// 			InverterPowerFlow: info.InverterPowerFlow,
// 			ACMeterPowerFlow:  info.ACMeterPowerFlow,
// 		}
// 	}).WithTimeout(2 * time.Second).OnError(func(err error) {
// 		ctx.Send(ctx.Self(), GetStorageControlPowerFlowResponse{
// 			ActorResponseMixIn: domain.ActorResponseMixIn{
// 				ResponseError: err,
// 			},
// 		})
// 	}).PipeTo(ctx.Self())
// }

// func SetStorageControlTask(ctx actor.Context, srv port.InverterService, timeout time.Duration, params domain.StorageControlParams) {
// 	NewBackgroundTask(ctx, func() (*SetStorageControlResponse, error) {
// 		err := srv.SetStorageControl(context.Background(), params)
// 		return &SetStorageControlResponse{
// 			ActorResponseMixIn: domain.ActorResponseMixIn{
// 				ResponseError: err,
// 			},
// 		}, nil
// 	}).WithTimeout(2 * time.Second).OnError(func(err error) {
// 		ctx.Send(ctx.Self(), SetStorageControlResponse{
// 			ActorResponseMixIn: domain.ActorResponseMixIn{
// 				ResponseError: err,
// 			},
// 		})
// 	}).PipeTo(ctx.Self())
// }
