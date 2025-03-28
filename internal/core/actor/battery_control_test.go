package actor

import (
	"errors"
	"testing"
	"time"

	adactor "github.com/berfenger/frostnews2mqtt/internal/adapter/actor"
	"github.com/berfenger/frostnews2mqtt/internal/config"
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/internal/core/service"
	"github.com/berfenger/frostnews2mqtt/internal/util/actorutil"
	"github.com/berfenger/frostnews2mqtt/pkg/sunspec_modbus"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestBatteryControlFlow(t *testing.T) {

	logger := zap.Must(zap.NewDevelopment())

	as := actorutil.NewActorSystemWithZapLogger(logger)

	context := as.Root

	cfg := config.Config{}
	cfg.GridConfig.MaxImportPower = 4000
	cfg.BatteryControlConfig.ControlIntervalMillis = 10000

	// modbus actor
	modbusProps := actor.PropsFromProducer(func() actor.Actor {
		return adactor.NewModbusActor(2*time.Second, &sunspec_modbus.TestInverterModbusReader{},
			sunspec_modbus.TestACMeterModbusReader{}, logger)
	})
	modbusActorPID := context.Spawn(modbusProps)

	// event actor
	evProps := actor.PropsFromProducer(func() actor.Actor { return adactor.NewTestMQTTActor(&cfg, logger) })
	eventActorPID := context.Spawn(evProps)

	// control logic
	control := &service.DefaultBatteryControlLogic{
		StartPowerThreshold:     0,
		MaxRatePowerIncrease:    800,
		MaxImportPower:          4000,
		PowerImportSafetyMargin: 200,
		Logger:                  logger,
	}

	// batteryControl actor
	battCtrlProps := actor.PropsFromProducer(func() actor.Actor {
		return NewBatteryControlActor(&cfg, modbusActorPID, eventActorPID, control, logger)
	})
	bcActorPID := context.Spawn(battCtrlProps)

	time.Sleep(2 * time.Second)

	// charging
	context.Send(bcActorPID, domain.BatteryControlChargeRequest{Enable: true})

	time.Sleep(200 * time.Millisecond)

	hcr, err := healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "charging", hcr.State, "actor state should be charging")

	// idle
	context.Send(bcActorPID, domain.BatteryControlChargeRequest{Enable: false})
	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "idle", hcr.State, "actor state should be idle")

	// holding
	context.Send(bcActorPID, domain.BatteryControlHoldRequest{Enable: true})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "holding", hcr.State, "actor state should be holding")

	// idle
	context.Send(bcActorPID, domain.BatteryControlHoldRequest{Enable: true})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "holding", hcr.State, "actor state should be holding")

	// hold + charge
	context.Send(bcActorPID, domain.BatteryControlHoldRequest{Enable: true})
	context.Send(bcActorPID, domain.BatteryControlChargeRequest{Enable: true})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "charging", hcr.State, "actor state should be charging")

	// exit charging and return to holding
	context.Send(bcActorPID, domain.BatteryControlChargeRequest{Enable: false})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "holding", hcr.State, "actor state should be holding")

	// exit holding and return to idle
	context.Send(bcActorPID, domain.BatteryControlHoldRequest{Enable: false})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "idle", hcr.State, "actor state should be idle")

}

func healthCheck(ctx *actor.RootContext, pid *actor.PID) (*domain.ActorHealthResponse, error) {
	resp, err := ctx.RequestFuture(pid, domain.ActorHealthRequest{}, 2*time.Second).Result()
	if err != nil {
		return nil, err
	}
	hcr, ok := resp.(domain.ActorHealthResponse)
	if !ok {
		return nil, errors.New("unexpcted response type")
	}
	return &hcr, nil
}
