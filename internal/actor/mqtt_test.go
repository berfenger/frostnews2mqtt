package actor

import (
	"fmt"
	"frostnews2mqtt/internal/config"
	"frostnews2mqtt/internal/events"
	"frostnews2mqtt/internal/mqtt"
	"frostnews2mqtt/internal/util"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMQTTActor(t *testing.T) {

	cfg := util.LoadTestConfig()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	as := TestActorSystem()

	context := as.Root

	es := eventstream.EventStream{}

	props := actor.PropsFromProducer(func() actor.Actor { return NewTestMQTTActor(&cfg, &es, logger) })
	pid := context.Spawn(props)

	time.Sleep(2 * time.Second)

	msg := ActorHealthRequest{}
	result, err := context.RequestFuture(pid, msg, 2*time.Second).Result()
	if err != nil {
		t.Error(err)
		return
	}
	resp, ok := result.(ActorHealthResponse)
	assert.True(t, ok)
	assert.NotNil(t, resp)

	es.Publish(events.SensorUpdateEvent{
		GenericSensorUpdateEvent: events.GenericSensorUpdateEvent{
			Id: events.SENSOR_ID_INVERTER_AC_POWER_FLOW,
		},
		Value: 245,
	})
	es.Publish(events.SensorUpdateEvent{
		GenericSensorUpdateEvent: events.GenericSensorUpdateEvent{
			Id: events.SENSOR_ID_INVERTER_PV_POWER,
		},
		Value: 345.32,
	})

	time.Sleep(1 * time.Second)

	context.Stop(pid)

	time.Sleep(1 * time.Second)

	as.Shutdown()
}

func NewTestMQTTActor(config *config.Config, eventStream *eventstream.EventStream, logger *logrus.Logger) *MQTTActor {
	act := &MQTTActor{
		config:      config,
		behavior:    actor.NewBehavior(),
		stash:       &Stash{},
		logger:      ActorLogger("mqtt", logger),
		eventStream: eventStream,
	}
	act.behavior.Become(act.DummyReceive)
	return act
}

func (state *MQTTActor) DummyReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.client = mqtt.CreateMQTTClient(state.config, mqtt.OptsFromConfig(state.config), nil, nil)
		state.eventStreamSub = state.eventStream.Subscribe(func(value any) {
			ctx.Send(ctx.Self(), OnEventStreamMessage{
				message: value,
			})
		})
	case ActorHealthRequest:
		state.logger.Debug("mqtt@default ActorHealthRequest")
		// respond health check request
		ctx.Respond(ActorHealthResponse{
			Id:      MQTT_ACTOR_ID,
			Healthy: true,
		})
	case OnEventStreamMessage:
		// receive message from event bus and publish to MQTT if needed
		state.logger.Debug("mqtt@default OnEventStreamMessage", "value", fmt.Sprintf("%T", msg.message))
		if rawMsg := state.event2MQTTMessage(msg.message); rawMsg != nil {
			state.logger.Debug("mqtt@default publish", "value", rawMsg.message, "topic", rawMsg.topic)
		}
	}
}
