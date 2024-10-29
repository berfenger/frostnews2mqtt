package actor

import (
	"errors"
	"frostnews2mqtt/internal/config"
	"frostnews2mqtt/internal/events"
	"frostnews2mqtt/pkg/sunspec_modbus"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestBatteryControlFlow(t *testing.T) {

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	as := TestActorSystem()

	context := as.Root

	cfg := config.Config{}
	cfg.MaxImportPower = 4000
	cfg.BatteryControlRevertTimeoutSeconds = 10

	// modbus actor
	modbusProps := actor.PropsFromProducer(func() actor.Actor {
		return NewModbusActor(&sunspec_modbus.TestInverterModbusReader{},
			sunspec_modbus.TestACMeterModbusReader{}, logger)
	})
	modbusActorPID := context.Spawn(modbusProps)

	// batteryControl actor
	battCtrlProps := actor.PropsFromProducer(func() actor.Actor {
		return NewBatteryControlActor(&cfg, modbusActorPID, &eventstream.EventStream{}, logger)
	})
	bcActorPID := context.Spawn(battCtrlProps)

	time.Sleep(2 * time.Second)

	// charging
	context.Send(bcActorPID, events.BatteryControlCharge{On: true})

	time.Sleep(200 * time.Millisecond)

	hcr, err := healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "charging", hcr.State, "actor state should be charging")

	// idle
	context.Send(bcActorPID, events.BatteryControlCharge{On: false})
	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "idle", hcr.State, "actor state should be idle")

	// holding
	context.Send(bcActorPID, events.BatteryControlHold{On: true})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "holding", hcr.State, "actor state should be holding")

	// idle
	context.Send(bcActorPID, events.BatteryControlHold{On: true})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "holding", hcr.State, "actor state should be holding")

	// hold + charge
	context.Send(bcActorPID, events.BatteryControlHold{On: true})
	context.Send(bcActorPID, events.BatteryControlCharge{On: true})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "charging", hcr.State, "actor state should be charging")

	// exit charging and return to holding
	context.Send(bcActorPID, events.BatteryControlCharge{On: false})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "holding", hcr.State, "actor state should be holding")

	// exit holding and return to idle
	context.Send(bcActorPID, events.BatteryControlHold{On: false})

	time.Sleep(200 * time.Millisecond)

	hcr, err = healthCheck(context, bcActorPID)
	if err != nil {
		t.Error(err)
		return
	}
	assert.True(t, hcr.Healthy, "actor should be healthy")
	assert.Equal(t, "idle", hcr.State, "actor state should be idle")

}

func healthCheck(ctx *actor.RootContext, pid *actor.PID) (*ActorHealthResponse, error) {
	resp, err := ctx.RequestFuture(pid, ActorHealthRequest{}, 2*time.Second).Result()
	if err != nil {
		return nil, err
	}
	hcr, ok := resp.(ActorHealthResponse)
	if !ok {
		return nil, errors.New("unexpcted response type")
	}
	return &hcr, nil
}
