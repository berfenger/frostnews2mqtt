package actor

import (
	"fmt"
	"testing"
	"time"

	adactor "github.com/berfenger/frostnews2mqtt/internal/adapter/actor"
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/internal/util"
	"github.com/berfenger/frostnews2mqtt/pkg/sunspec_modbus"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMasterActor(t *testing.T) {

	as := actor.NewActorSystem()
	context := as.Root

	cfg := util.LoadTestConfig()
	logCfg := zap.NewDevelopmentConfig()
	logCfg.Level = zap.NewAtomicLevelAt(cfg.LogLevel)
	logger := zap.Must(logCfg.Build())

	props := actor.PropsFromProducer(func() actor.Actor {
		return NewMasterOfPuppetsActor(cfg, func() *adactor.ModbusActor {
			return adactor.NewModbusActor(2*time.Second, &sunspec_modbus.TestInverterModbusReader{}, sunspec_modbus.TestACMeterModbusReader{}, logger)
		}, func() *adactor.MQTTActor {
			return adactor.NewTestMQTTActor(&cfg, logger)
		}, logger)
	})
	pid, err := context.SpawnNamed(props, "master")
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	res, err := context.RequestFuture(pid, domain.ActorHealthRequest{}, 10*time.Second).Result()
	if err != nil {
		t.Error(err)
		//return
	}
	healthResp, ok := res.(domain.ActorHealthResponse)
	assert.True(t, ok)
	fmt.Printf("Health response: %+v\n", healthResp)
	assert.NotNil(t, healthResp)

	assert.True(t, healthResp.Healthy, "healthy is true")

	context.Stop(pid)

	as.Shutdown()
}
