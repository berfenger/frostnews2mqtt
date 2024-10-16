package actor

import (
	"fmt"
	"frostnews2mqtt/internal/events"
	"frostnews2mqtt/internal/mqtt"
	"log/slog"
	"strconv"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/lmittmann/tint"
	"github.com/sirupsen/logrus"
)

func PipeToSelfWithRecover(ctx actor.Context, future *actor.Future, mapFn func(error) any) {
	ctx.ReenterAfter(future, func(msg any, err error) {
		if err != nil {
			ctx.Send(ctx.Self(), mapFn(err))
			return
		}
		ctx.Send(ctx.Self(), msg)
	})
}

func OrCommandErrorResponse[T any](fn func() (*T, error)) any {
	value, err := fn()
	var resp any
	if err != nil {
		resp = CommandErrorResponse{
			Error: fmt.Sprintf("%s", err),
		}
	} else {
		resp = *value
	}
	return resp
}

func DoBackgroundTask[T any](ctx actor.Context, fn func() T) {
	self := ctx.Self()
	sender := ctx.Sender()
	go func() {
		result := fn()
		ctx.Send(self, backgroundTaskResult{
			message: result,
			replyTo: sender,
		})
	}()
}

type BackgroundTask[T any] struct {
	ctx actor.Context
	fn  func() T
}

func NewBackgroundTask[T any](ctx actor.Context, fn func() T) BackgroundTask[T] {
	return BackgroundTask[T]{
		ctx: ctx,
		fn:  fn,
	}
}

func MapBackgroundTask[T, T2 any](t BackgroundTask[T], fn func(T) T2) BackgroundTask[T2] {
	return NewBackgroundTask(t.ctx, func() T2 { return fn(t.fn()) })
}

func (t BackgroundTask[T]) PipeTo(pid *actor.PID) {
	go func() {
		result := t.fn()
		t.ctx.Send(pid, result)
	}()
}

func (t BackgroundTask[T]) Map(fn func(T) any) BackgroundTask[any] {
	return NewBackgroundTask(t.ctx, func() any { return fn(t.fn()) })
}

func TestActorSystem() *actor.ActorSystem {
	ll := logrus.New()
	ll.SetLevel(logrus.DebugLevel)
	return actor.NewActorSystem(actor.WithLoggerFactory(func(system *actor.ActorSystem) *slog.Logger {

		// create a new logger
		return slog.New(tint.NewHandler(ll.Out, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.DateTime,
		}))
	}))
}

func ActorLogger(actorName string, logger *logrus.Logger) *logrus.Entry {
	return logger.WithField("actor", actorName)
}

func ParsedMQTTCommandToCommand(cmd mqtt.ParsedMQTTCommand) (events.ParsedCommand, error) {
	if cmd.DeviceId == events.SWITCH_ID_BATTERY_HOLD {
		return events.BatteryControlHold{
			On: cmd.Payload == "on",
		}, nil
	} else if cmd.DeviceId == events.SWITCH_ID_BATTERY_CHARGE {
		return events.BatteryControlCharge{
			On: cmd.Payload == "on",
		}, nil
	} else if cmd.DeviceId == events.INPUT_NUMBER_ID_BATTERY_CHARGE_TARGET_SOC {
		value, err := strconv.ParseUint(cmd.Payload, 10, 8)
		if err != nil || value > 100 {
			return nil, err
		}
		return events.BatteryControlSetTargetSoC{
			TargetSoC: uint8(value),
		}, nil
	}
	return nil, nil
}
