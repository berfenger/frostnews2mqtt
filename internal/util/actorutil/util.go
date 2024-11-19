package actorutil

import (
	"frostnews2mqtt/internal/core/domain"
	"frostnews2mqtt/internal/mqtt"
	"log/slog"
	"strconv"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/lmittmann/tint"
	"go.uber.org/zap"
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

// func OrCommandErrorResponse[T any](fn func() (*T, error)) any {
// 	value, err := fn()
// 	var resp any
// 	if err != nil {
// 		resp = CommandErrorResponse{
// 			Error: fmt.Sprintf("%s", err),
// 		}
// 	} else {
// 		resp = *value
// 	}
// 	return resp
// }

// func DoBackgroundTask[T any](ctx actor.Context, fn func() T) {
// 	self := ctx.Self()
// 	sender := ctx.Sender()
// 	go func() {
// 		result := fn()
// 		ctx.Send(self, backgroundTaskResult{
// 			message: result,
// 			replyTo: sender,
// 		})
// 	}()
// }

// type BackgroundTask[T any] struct {
// 	ctx actor.Context
// 	fn  func() T
// }

// func NewBackgroundTask[T any](ctx actor.Context, fn func() T) BackgroundTask[T] {
// 	return BackgroundTask[T]{
// 		ctx: ctx,
// 		fn:  fn,
// 	}
// }

// func MapBackgroundTask[T, T2 any](t BackgroundTask[T], fn func(T) T2) BackgroundTask[T2] {
// 	return NewBackgroundTask(t.ctx, func() T2 { return fn(t.fn()) })
// }

// func (t BackgroundTask[T]) PipeTo(pid *actor.PID) {
// 	go func() {
// 		result := t.fn()
// 		t.ctx.Send(pid, result)
// 	}()
// }

// func (t BackgroundTask[T]) Map(fn func(T) any) BackgroundTask[any] {
// 	return NewBackgroundTask(t.ctx, func() any { return fn(t.fn()) })
// }

// func TestActorSystem() *actor.ActorSystem {
// 	ll := logrus.New()
// 	ll.SetLevel(logrus.DebugLevel)
// 	return actor.NewActorSystem(actor.WithLoggerFactory(func(system *actor.ActorSystem) *slog.Logger {

// 		// create a new logger
// 		return slog.New(tint.NewHandler(ll.Out, &tint.Options{
// 			Level:      slog.LevelDebug,
// 			TimeFormat: time.DateTime,
// 		}))
// 	}))
// }

func NewActorSystemWithZapLogger(logger *zap.Logger) *actor.ActorSystem {
	stdOutLogger := zap.NewStdLog(logger)

	var slogLevel slog.Level = slog.LevelInfo

	switch logger.Level() {
	case zap.DebugLevel:
		slogLevel = slog.LevelDebug
	case zap.InfoLevel:
		slogLevel = slog.LevelInfo
	case zap.WarnLevel:
		slogLevel = slog.LevelWarn
	case zap.ErrorLevel:
		slogLevel = slog.LevelError
	case zap.PanicLevel:
		slogLevel = slog.LevelError
	}

	return actor.NewActorSystem(actor.WithLoggerFactory(func(system *actor.ActorSystem) *slog.Logger {

		// create a new logger
		return slog.New(tint.NewHandler(stdOutLogger.Writer(), &tint.Options{
			Level:      slogLevel,
			TimeFormat: time.DateTime,
		}))
	}))
}

func ActorLogger(actorName string, logger *zap.Logger) *zap.Logger {
	return logger.With(zap.String("actor", actorName))
}

func ParsedMQTTCommandToCommand(cmd mqtt.ParsedMQTTCommand) (domain.ActorRequest, error) {
	if cmd.DeviceId == domain.SWITCH_ID_BATTERY_HOLD {
		return domain.BatteryControlHoldRequest{
			Enable: cmd.Payload == "on",
		}, nil
	} else if cmd.DeviceId == domain.SWITCH_ID_BATTERY_CHARGE {
		return domain.BatteryControlChargeRequest{
			Enable: cmd.Payload == "on",
		}, nil
	} else if cmd.DeviceId == domain.INPUT_NUMBER_ID_BATTERY_CHARGE_TARGET_SOC {
		value, err := strconv.ParseUint(cmd.Payload, 10, 8)
		if err != nil || value > 100 {
			return nil, err
		}
		return domain.BatteryControlSetTargetSoCRequest{
			TargetSoC: uint(value),
		}, nil
	}
	return nil, nil
}
