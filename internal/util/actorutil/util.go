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

func NewActorSystemWithZapLogger(logger *zap.Logger) *actor.ActorSystem {
	stdOutLogger := zap.NewStdLog(logger)

	slogLevel := slog.LevelInfo

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
	switch cmd.DeviceId {
	case domain.SWITCH_ID_BATTERY_HOLD:
		return domain.BatteryControlHoldRequest{
			Enable: cmd.Payload == "on",
		}, nil
	case domain.SWITCH_ID_BATTERY_CHARGE:
		return domain.BatteryControlChargeRequest{
			Enable: cmd.Payload == "on",
		}, nil
	case domain.INPUT_NUMBER_ID_BATTERY_CHARGE_TARGET_SOC:
		value, err := strconv.ParseUint(cmd.Payload, 10, 8)
		if err != nil || value > 100 {
			return nil, err
		}
		return domain.BatteryControlSetTargetSoCRequest{
			TargetSoC: uint(value),
		}, nil
	default:
		return nil, nil
	}
}
