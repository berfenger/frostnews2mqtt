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
