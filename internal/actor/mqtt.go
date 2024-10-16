package actor

import (
	"encoding/json"
	"fmt"
	"frostnews2mqtt/internal/config"
	"frostnews2mqtt/internal/events"
	"frostnews2mqtt/internal/mqtt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

const (
	MQTT_ACTOR_ID = "mqtt"
)

type MQTTActor struct {
	config         *config.Config
	behavior       actor.Behavior
	stash          *Stash
	client         *mqtt.MQTTClient
	eventStream    *eventstream.EventStream
	eventStreamSub *eventstream.Subscription
	logger         *logrus.Entry
}

type OnEventStreamMessage struct {
	message any
}

type MQTTConnected struct {
}

type MQTTSubscribed struct {
}

type MQTTConnectionLost struct {
	Error error
}

type PublishHADiscovery struct {
	Sensors      []events.GenericSensor
	Switches     []events.GenericSwitch
	InputNumbers []events.GenericInputNumber
}

type publishResult struct {
	Error error
}

type parsedCommand struct {
	command *mqtt.ParsedMQTTCommand
}

type rawMessage struct {
	topic   string
	message string
	retain  bool
}

func NewMQTTActor(config *config.Config, eventStream *eventstream.EventStream, logger *logrus.Logger) *MQTTActor {
	act := &MQTTActor{
		config:      config,
		behavior:    actor.NewBehavior(),
		stash:       &Stash{},
		logger:      ActorLogger("mqtt", logger),
		eventStream: eventStream,
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

		// subscribe to eventStream
		state.eventStreamSub = state.eventStream.Subscribe(func(value any) {
			ctx.Send(ctx.Self(), OnEventStreamMessage{
				message: value,
			})
		})

		// subscribe to MQTT command topic
		state.client.SubscribeToCommandTopic(func(c pahomqtt.Client, m pahomqtt.Message) {
			cmd, err := state.client.ParseMQTTCommand(m)
			if err == nil && cmd != nil {
				ctx.Send(ctx.Self(), parsedCommand{command: cmd})
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
		state.logger.Debug("mqtt@starting connection lost", "error", msg.Error)
		panic(msg.Error)
	case *actor.Restarting:
		state.stop()
	default:
		state.logger.Trace("mqtt@starting stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

func (state *MQTTActor) DefaultReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Restarting:
		state.stop()
	case *actor.Stopping:
		state.stop()
	case ActorHealthRequest:
		state.logger.Debug("mqtt@default ActorHealthRequest")
		// respond health check request
		ctx.Respond(ActorHealthResponse{
			Id:      MQTT_ACTOR_ID,
			Healthy: true,
		})
	case parsedCommand:
		// route command to parent
		state.logger.Debug("mqtt@default parsedCommand", "value", fmt.Sprintf("%+v", msg.command))
		ctx.Send(ctx.Parent(), msg)
	case OnEventStreamMessage:
		// receive message from event bus and publish to MQTT if needed
		state.logger.Debug("mqtt@default OnEventStreamMessage", "value", fmt.Sprintf("%T", msg.message))
		state.publishSensorValue(ctx, msg.message)
	case PublishHADiscovery:
		state.logger.Debug("mqtt@default PublishHADiscovery")
		state.PublishHomeAssistantDiscovery(ctx, msg.Sensors, msg.Switches, msg.InputNumbers)
	case MQTTConnectionLost:
		// if connection lost, stop actor and let supervisor decide
		state.logger.Debug("mqtt@default connection lost", "error", msg.Error)
		panic(msg.Error)
	default:
		state.logger.Trace("mqtt@default stash", "type", fmt.Sprintf("%T", msg))
	}
}

func (state *MQTTActor) event2MQTTMessage(event any) *rawMessage {
	switch msg := event.(type) {
	case events.SensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.SensorStateTopic(msg.Id),
			message: fmt.Sprintf(fmt.Sprintf("%%.%df", msg.Decimals), msg.Value),
		}
	case events.BinarySensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.BinarySensorStateTopic(msg.Id),
			message: bool2MQTTPayload(msg.Value),
		}
	case events.SwitchSensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.SwitchStateTopic(msg.Id),
			message: bool2MQTTPayload(msg.Value),
			retain:  true,
		}
	case events.InputNumberSensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.InputNumberStateTopic(msg.Id),
			message: fmt.Sprintf(fmt.Sprintf("%%.%df", msg.Decimals), msg.Value),
			retain:  true,
		}
	case events.TextSensorUpdateEvent:
		return &rawMessage{
			topic:   state.client.SensorStateTopic(msg.Id),
			message: msg.Value,
		}
	case events.BridgeStateUpdateEvent:
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

func (state *MQTTActor) publishSensorValue(ctx actor.Context, event any) {
	msg := state.event2MQTTMessage(event)
	if msg != nil {
		state.logger.Trace("mqtt@publish: sensor publish", "message", msg.message, "topic", msg.topic)
		state.client.Publish(msg.topic, msg.message, 1, msg.retain, func(err error) {
			ctx.Send(ctx.Self(), publishResult{Error: err})
		}, 5*time.Second)
		state.behavior.BecomeStacked(state.PublishingReceive)
	}
}

func (state *MQTTActor) PublishingReceive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case publishResult:
		// log error and return to default state
		if msg.Error != nil {
			state.logger.Error("mqtt@publishing could not publish a message", "error", msg.Error)
		}
		state.behavior.UnbecomeStacked()
		state.stash.UnstashOldest(ctx)
	default:
		state.logger.Trace("mqtt@publishing stash", "type", fmt.Sprintf("%T", msg))
		state.stash.Stash(ctx, msg)
	}
}

func (state *MQTTActor) PublishHomeAssistantDiscovery(ctx actor.Context, sensors []events.GenericSensor,
	switches []events.GenericSwitch, inputNumbers []events.GenericInputNumber) error {
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
	if state.eventStreamSub != nil {
		state.eventStream.Unsubscribe(state.eventStreamSub)
		state.eventStreamSub = nil
	}
}

func bool2MQTTPayload(value bool) string {
	if value {
		return mqtt.MQTT_PAYLOAD_ON
	} else {
		return mqtt.MQTT_PAYLOAD_OFF
	}
}
