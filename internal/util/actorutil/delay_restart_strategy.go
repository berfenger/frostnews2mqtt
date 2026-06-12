package actorutil

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

func NewRestartAfterDelayStrategy(delay time.Duration, maxAttempts int) actor.SupervisorStrategy {
	return &restartAfterDelayStrategy{
		delay:       delay,
		maxAttempts: maxAttempts,
	}
}

type restartAfterDelayStrategy struct {
	delay       time.Duration
	maxAttempts int
}

var _ actor.SupervisorStrategy = &restartAfterDelayStrategy{}

func (strategy *restartAfterDelayStrategy) HandleFailure(actorSystem *actor.ActorSystem, supervisor actor.Supervisor, child *actor.PID, rs *actor.RestartStatistics, reason any, _ any) {
	rs.Fail()
	if strategy.maxAttempts > 0 && rs.FailureCount() > strategy.maxAttempts {
		supervisor.StopChildren(child)
		return
	}

	time.AfterFunc(strategy.delay, func() {
		supervisor.RestartChildren(child)
	})
}
