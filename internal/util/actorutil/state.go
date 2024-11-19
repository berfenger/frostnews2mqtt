package actorutil

import (
	"github.com/asynkron/protoactor-go/actor"
)

type ActorWithStates struct {
	Behavior actor.Behavior
}

type ActorState interface {
	Name() string
	Receive(actor.Context)
}

func (s *ActorWithStates) Become(state ActorState) {
	s.Behavior.Become(state.Receive)
}

func (s *ActorWithStates) BecomeStacked(state ActorState) {
	s.Behavior.BecomeStacked(state.Receive)
}

func (s *ActorWithStates) UnbecomeStacked() {
	s.Behavior.UnbecomeStacked()
}
