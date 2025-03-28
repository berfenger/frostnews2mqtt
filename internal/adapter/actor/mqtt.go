package actor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/berfenger/frostnews2mqtt/internal/config"
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/internal/mqtt"
	"github.com/berfenger/frostnews2mqtt/internal/util/actorutil"

	"github.com/asynkron/protoactor-go/actor"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

type MQTTActor struct {
	config   *config.Config
	behavior actor.Behavior
	stash    *actorutil.Stash
	client   *mqtt.MQTTClient
	logger   *zap.Logger
}

type MQTTConnected struct {
}

type MQTTSubscribed struct {
}

type MQTTConnectionLost struct {
	Error error
}

type publishResult struct {
	ReplyTo *actor.PID
	Error   error
}

type ParsedCommand struct {
	Command *mqtt.ParsedMQTTCommand
}

type rawMessage struct {
	topic   string
	message string
	retain  bool
}

func NewMQTTActor(config *config.Config, logger *zap.Logger) *MQTTActor {
	act := &MQTTActor{
		config:   config,
		behavior: actor.NewBehavior(),
		stash:    &actorutil.Stash{},
		logger:   actorutil.ActorLogger(domain.ACTOR_ID_MQTT, logger),
	}
	act.behavior.Become(act.StartingReceive)
	return act
}

func (state *MQTTActor) Receive(context actor.Context) {
	state.behavior.Receive(context)
}

func (state *MQTTActor) StartingReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.logger.Debug("mqtt@starting started")

		// create MQTT client
		state.client = mqtt.CreateMQTTClient(state.config, mqtt.OptsFromConfig(state.config), func(_ pahomqtt.Client) {
		}, func(_ pahomqtt.Client, err error) {
			ctx.Send(ctx.Self(), MQTTConnectionLost{Error: err})
		})

		// connect to MQTT server
		state.client.Connect(func(err error) {
			if err != nil {
				ctx.Send(ctx.Self(), MQTTConnectionLost{Error: err})
			} else {
				ctx.Send(ctx.Self(), MQTTConnected{})
			}
		}, 10*time.Second)

	case MQTTConnected:
		state.logger.Debug("mqtt@starting connected")

		state.client.Publish(state.client.BridgeStateTopic(), mqtt.MQTT_PAYLOAD_ONLINE, 0, true, func(error) {}, 500*time.Millisecond)

		// subscribe to MQTT command topic
		state.client.SubscribeToCommandTopic(func(c pahomqtt.Client, m pahomqtt.Message) {
			cmd, err := state.client.ParseMQTTCommand(m)
			if err == nil && cmd != nil {
				ctx.Send(ctx.Self(), ParsedCommand{Command: cmd})
			}
		}, func(err error) {
			if err != nil {
				ctx.Send(ctx.Self(), MQTTConnectionLost{Error: err})
			} else {
				ctx.Send(ctx.Self(), MQTTSubscribed{})
			}
		}, 1*time.Second)
	case MQTTSubscribed:
		// init completed, transition to default state
		state.logger.Debug("mqtt@starting subscribed")
		state.behavior.Become(state.DefaultReceive)
		state.stash.UnstashAll(ctx)
	case MQTTConnectionLost:
		// if connection lost, stop actor and let supervisor decide
		state.logger.Error("mqtt@starting connection lost", zap.Error(msg.Error))
		panic(msg.Error)
	case *actor.Restarting:
		state.stop()
	default:
		state.logger.Debug("mqtt@starting stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *MQTTActor) DefaultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Restarting:
		state.stop()
	case *actor.Stopping:
		state.stop()
	case domain.ActorHealthRequest:
		state.logger.Debug("mqtt@default ActorHealthRequest")
		// respond health check request
		ctx.Respond(domain.ActorHealthResponse{
			Id:      domain.ACTOR_ID_MQTT,
			Healthy: true,
			State:   "idle",
		})
	case ParsedCommand:
		// route command to parent
		state.logger.Debug("mqtt@default parsedCommand", zap.Any("command", msg.Command))
		ctx.Send(ctx.Parent(), msg)
	case domain.PublishMessageRequest:
		state.logger.Debug("mqtt@default PublishMessageRequest", zap.Any("message", msg))
		state.publishMessage(ctx, msg.Topic, msg.Payload, msg.Retain, actorutil.ForRequest(msg).ReplyTo(ctx))
	case domain.PublishSensorUpdateRequest:
		// receive message from event bus and publish to MQTT if needed
		state.logger.Debug("mqtt@default PublishSensorUpdateRequest", zap.String("type", fmt.Sprintf("%T", msg.Event)))
		state.publishSensorValue(ctx, msg.Event, msg.Retain)
	case domain.PublishDiscoveryRequest:
		state.logger.Debug("mqtt@default PublishHADiscovery")
		err := state.PublishHomeAssistantDiscovery(ctx, msg.Sensors, msg.Switches, msg.InputNumbers)
		if err != nil {
			state.logger.Error("mqtt@default PublishHADiscovery error", zap.Error(err))
		}
	case MQTTConnectionLost:
		// if connection lost, stop actor and let supervisor decide
		state.logger.Error("mqtt@default connection lost", zap.Error(msg.Error))
		panic(msg.Error)
	default:
		state.logger.Debug("mqtt@default stash", zap.String("type", fmt.Sprintf("%T", msg)))
	}
}

func (state *MQTTActor) event2MQTTMessage(event any) *rawMessage {
	switch msg := event.(type) {
	case domain.FloatSensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.SensorStateTopic(msg.Id),
			message: fmt.Sprintf(fmt.Sprintf("%%.%df", msg.Decimals), msg.Value),
		}
	case domain.BinarySensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.BinarySensorStateTopic(msg.Id),
			message: bool2MQTTPayload(msg.Value),
		}
	case domain.SwitchSensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.SwitchStateTopic(msg.Id),
			message: bool2MQTTPayload(msg.Value),
			retain:  true,
		}
	case domain.InputNumberSensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.InputNumberStateTopic(msg.Id),
			message: fmt.Sprintf(fmt.Sprintf("%%.%df", msg.Decimals), msg.Value),
			retain:  true,
		}
	case domain.TextSensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.SensorStateTopic(msg.Id),
			message: msg.Value,
		}
	case domain.BridgeStateUpdateEvent:
		var stringMessage string
		if msg.Value {
			stringMessage = mqtt.MQTT_PAYLOAD_ONLINE
		} else {
			stringMessage = mqtt.MQTT_PAYLOAD_OFFLINE
		}
		return &rawMessage{
			topic:   state.client.BridgeStateTopic(),
			message: stringMessage,
		}
	default:
		return nil
	}
}

func (state *MQTTActor) publishSensorValue(ctx actor.Context, event domain.SensorUpdateEvent, retain bool) {
	msg := state.event2MQTTMessage(event)
	if msg != nil {
		state.logger.Sugar().Debugf("mqtt@publish: sensor publish %s => %s", msg.topic, msg.message)
		state.client.Publish(msg.topic, msg.message, 1, msg.retain || retain, func(err error) {
			ctx.Send(ctx.Self(), publishResult{Error: err})
		}, 5*time.Second)
		state.behavior.BecomeStacked(state.EventPublishResultReceive)
	}
}

func (state *MQTTActor) publishMessage(ctx actor.Context, topic, payload string, retain bool, replyTo *actor.PID) {
	state.logger.Sugar().Debugf("mqtt@publish: message publish %s => %s", topic, payload)
	state.client.Publish(topic, payload, 1, retain, func(err error) {
		ctx.Send(ctx.Self(), publishResult{ReplyTo: replyTo, Error: err})
	}, 5*time.Second)
	state.behavior.BecomeStacked(state.MessagePublishResultReceive)
}

func (state *MQTTActor) MessagePublishResultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case publishResult:
		// log error and return to default state
		if msg.Error != nil {
			state.logger.Error("mqtt@publishing could not publish a message", zap.Error(msg.Error))
		}
		if msg.ReplyTo != nil {
			ctx.Send(msg.ReplyTo, domain.PublishMessageResponse{
				ActorResponseMixIn: domain.ActorResponseMixIn{
					ResponseError: msg.Error,
				},
			})
		}
		state.behavior.UnbecomeStacked()
		state.stash.UnstashOldest(ctx)
	default:
		state.logger.Debug("mqtt@publishing stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *MQTTActor) EventPublishResultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case publishResult:
		// log error and return to default state
		if msg.Error != nil {
			state.logger.Error("mqtt@publishing could not publish a message", zap.Error(msg.Error))
		}
		if msg.ReplyTo != nil {
			ctx.Send(msg.ReplyTo, domain.PublishSensorUpdateResponse{
				ActorResponseMixIn: domain.ActorResponseMixIn{
					ResponseError: msg.Error,
				},
			})
		}
		state.behavior.UnbecomeStacked()
		state.stash.UnstashOldest(ctx)
	default:
		state.logger.Debug("mqtt@publishing stash", zap.String("type", fmt.Sprintf("%T", msg)))
		state.stash.Stash(ctx, msg)
	}
}

func (state *MQTTActor) PublishHomeAssistantDiscovery(ctx actor.Context, sensors []domain.GenericSensor,
	switches []domain.GenericSwitch, inputNumbers []domain.GenericInputNumber) error {
	for i := range sensors {
		msg := mqtt.GenericSensorToHADiscoveryMessage(state.client, sensors[i])
		payload, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		topic := mqtt.HADiscoverySensorTopic(sensors[i])
		state.client.Publish(topic, payload, 0, true, func(error) {}, 1*time.Second)
	}
	for i := range switches {
		msg := mqtt.GenericSwitchToHADiscoveryMessage(state.client, switches[i])
		payload, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		topic := mqtt.HADiscoverySwitchTopic(switches[i])
		state.client.Publish(topic, payload, 0, true, func(error) {}, 1*time.Second)
	}
	for i := range inputNumbers {
		msg := mqtt.GenericInputNumberToHADiscoveryMessage(state.client, inputNumbers[i])
		payload, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		topic := mqtt.HADiscoveryInputNumberTopic(inputNumbers[i])
		state.client.Publish(topic, payload, 0, true, func(error) {}, 1*time.Second)
	}
	return nil
}

func (state *MQTTActor) stop() {
	state.logger.Debug("mqtt: disconnect")
	state.client.Publish(state.client.BridgeStateTopic(), mqtt.MQTT_PAYLOAD_OFFLINE, 0, true, func(error) {}, 500*time.Millisecond)
	if state.client != nil {
		state.client.Disconnect(500 * time.Millisecond)
	}
}

func bool2MQTTPayload(value bool) string {
	if value {
		return mqtt.MQTT_PAYLOAD_ON
	} else {
		return mqtt.MQTT_PAYLOAD_OFF
	}
}

// Dummy actor
func NewTestMQTTActor(config *config.Config, logger *zap.Logger) *MQTTActor {
	act := &MQTTActor{
		config:   config,
		behavior: actor.NewBehavior(),
		stash:    &actorutil.Stash{},
		logger:   actorutil.ActorLogger("mqtt", logger),
	}
	act.behavior.Become(act.DummyReceive)
	return act
}

func (state *MQTTActor) DummyReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.client = mqtt.CreateMQTTClient(state.config, mqtt.OptsFromConfig(state.config), nil, nil)
	case domain.ActorHealthRequest:
		state.logger.Debug("mqtt@default ActorHealthRequest")
		// respond health check request
		ctx.Respond(domain.ActorHealthResponse{
			Id:      domain.ACTOR_ID_MQTT,
			Healthy: true,
			State:   "idle",
		})
	case domain.PublishSensorUpdateRequest:
		if msg.ReplyToRef != nil {
			ctx.Respond(domain.PublishSensorUpdateResponse{})
		}
	case domain.PublishMessageRequest:
		if msg.ReplyToRef != nil {
			ctx.Respond(domain.PublishMessageResponse{})
		}
	}
}
