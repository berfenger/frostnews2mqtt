package actor

import (
	"context"
	"testing"
	"time"

	"github.com/berfenger/frostnews2mqtt/internal/core/component"
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/internal/util"
	"github.com/berfenger/frostnews2mqtt/internal/util/actorutil"
	"github.com/berfenger/frostnews2mqtt/pkg/util/logutil"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMQTTActor(t *testing.T) {

	cfg := util.LoadTestConfig()

	logger := zap.Must(zap.NewDevelopment())
	ctx := context.Background()
	ctx = logutil.WithLogger(ctx, logger)

	as := actorutil.NewActorSystem(ctx)

	context := as.Root

	props := actor.PropsFromProducer(func() actor.Actor { return NewTestMQTTActor(&cfg, logger) })
	pid := context.Spawn(props)

	time.Sleep(2 * time.Second)

	msg := domain.ActorHealthRequest{}
	result, err := context.RequestFuture(pid, msg, 2*time.Second).Result()
	if err != nil {
		t.Error(err)
		return
	}
	resp, ok := result.(domain.ActorHealthResponse)
	assert.True(t, ok)
	assert.NotNil(t, resp)

	context.Send(pid, domain.PublishSensorUpdateRequest{
		Event: domain.FloatSensorUpdateEvent{
			SensorUpdateEventMixIn: domain.SensorUpdateEventMixIn{
				Id: component.SENSOR_ID_INVERTER_AC_POWER_FLOW,
			},
			Value: 245,
		},
	})

	context.Send(pid, domain.PublishSensorUpdateRequest{
		Event: domain.FloatSensorUpdateEvent{
			SensorUpdateEventMixIn: domain.SensorUpdateEventMixIn{
				Id: component.SENSOR_ID_INVERTER_PV_POWER,
			},
			Value: 345.32,
		},
	})

	time.Sleep(1 * time.Second)

	context.Stop(pid)

	time.Sleep(1 * time.Second)

	as.Shutdown()
}
