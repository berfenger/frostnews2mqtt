package actor

import (
	"github.com/asynkron/protoactor-go/actor"
)

type ActorWithStates struct {
	behavior actor.Behavior
}

type ActorState interface {
	Name() string
	Receive(actor.Context)
}

func (s *ActorWithStates) Become(state ActorState) {
	s.behavior.Become(state.Receive)
}

func (s *ActorWithStates) BecomeStacked(state ActorState) {
	s.behavior.BecomeStacked(state.Receive)
}

func (s *ActorWithStates) UnbecomeStacked() {
	s.behavior.UnbecomeStacked()
}
