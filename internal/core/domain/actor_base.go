package domain

import (
	"github.com/asynkron/protoactor-go/actor"
)

type ActorRef actor.PID

type ActorRequestMixIn struct {
	ReplyToRef *ActorRef
}

type ActorRequest interface {
	ReplyTo() *ActorRef
}

func (r ActorRequestMixIn) ReplyTo() *ActorRef {
	return r.ReplyToRef
}

type ActorResponseMixIn struct {
	ResponseError error
}

func (r ActorResponseMixIn) GetResponseError() error {
	return r.ResponseError
}

func (r ActorResponseMixIn) HasResponseError() bool {
	return r.ResponseError != nil
}

type ActorResponse interface {
	GetResponseError() error
	HasResponseError() bool
}
