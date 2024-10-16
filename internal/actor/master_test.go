package actor

import (
	"fmt"
	"frostnews2mqtt/internal/util"
	"frostnews2mqtt/pkg/sunspec_modbus"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMasterActor(t *testing.T) {

	as := actor.NewActorSystem()
	context := as.Root

	cfg := util.LoadTestConfig()
	logger := logrus.New()
	logger.SetLevel(cfg.LogLevel)

	props := actor.PropsFromProducer(func() actor.Actor {
		return NewMasterOfPuppetsActor(cfg, func() *ModbusActor {
			return NewModbusActor(sunspec_modbus.TestInverterModbusReader{}, sunspec_modbus.TestACMeterModbusReader{}, logger)
		}, func(es *eventstream.EventStream) *MQTTActor {
			return NewTestMQTTActor(&cfg, es, logger)
		}, logger)
	})
	pid, err := context.SpawnNamed(props, "master")
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	res, err := context.RequestFuture(pid, ActorHealthRequest{}, 10*time.Second).Result()
	if err != nil {
		t.Error(err)
		//return
	}
	healthResp, ok := res.(ActorHealthResponse)
	assert.True(t, ok)
	fmt.Printf("Health response: %+v\n", healthResp)
	assert.NotNil(t, healthResp)

	assert.True(t, healthResp.Healthy, "healthy is true")

	context.Stop(pid)

	as.Shutdown()
}
