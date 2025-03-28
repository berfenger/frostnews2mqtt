package mqtt

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"time"

	"github.com/berfenger/frostnews2mqtt/internal/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	MQTT_PAYLOAD_ONLINE  = "online"
	MQTT_PAYLOAD_OFFLINE = "offline"
	MQTT_PAYLOAD_ON      = "on"
	MQTT_PAYLOAD_OFF     = "off"
)

func OptsFromConfig(cfg *config.Config) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.MQTT.Host, cfg.MQTT.Port))
	opts.SetClientID(fmt.Sprintf("frostnews_%d", rand.IntN(1000)))
	if cfg.MQTT.Username != "" && cfg.MQTT.Password != "" {
		opts.SetUsername(cfg.MQTT.Username)
		opts.SetPassword(cfg.MQTT.Password)
	}
	opts.WillEnabled = true
	opts.WillPayload = []byte(MQTT_PAYLOAD_OFFLINE)
	opts.WillRetained = true
	opts.WillTopic = bridgeStateTopic(cfg.MQTT.BaseTopic)
	opts.WillQos = 0

	return opts
}

func CreateMQTTClient(cfg *config.Config, opts *mqtt.ClientOptions, onConnectHandler func(client mqtt.Client),
	onConnectionLostHandler func(mqtt.Client, error)) *MQTTClient {
	if onConnectHandler != nil {
		opts.OnConnect = onConnectHandler
	}
	if onConnectionLostHandler != nil {
		opts.OnConnectionLost = onConnectionLostHandler
	}
	return &MQTTClient{
		client:                   mqtt.NewClient(opts),
		cfg:                      cfg.MQTT,
		switchCommandRegexp:      switchCommandExtractor(cfg.MQTT.BaseTopic),
		inputNumberCommandRegexp: inputNumberCommandExtractor(cfg.MQTT.BaseTopic),
	}
}

type MQTTClient struct {
	client                   mqtt.Client
	cfg                      config.MQTTConfig
	switchCommandRegexp      *regexp.Regexp
	inputNumberCommandRegexp *regexp.Regexp
}

type ParsedMQTTCommand struct {
	DeviceId string
	Command  string
	Param    string
	Payload  string
}

func (c *MQTTClient) baseTopic() string {
	return c.cfg.BaseTopic
}

func (c *MQTTClient) BridgeStateTopic() string {
	return bridgeStateTopic(c.baseTopic())
}

func (c *MQTTClient) SensorStateTopic(sensorId string) string {
	return fmt.Sprintf("%s/sensor/%s/state", c.baseTopic(), sensorId)
}

func (c *MQTTClient) BinarySensorStateTopic(sensorId string) string {
	return fmt.Sprintf("%s/binary_sensor/%s/state", c.baseTopic(), sensorId)
}

func (c *MQTTClient) SwitchStateTopic(switchId string) string {
	return fmt.Sprintf("%s/switch/%s/state", c.baseTopic(), switchId)
}

func (c *MQTTClient) SwitchCommandTopic(switchId string) string {
	return fmt.Sprintf("%s/switch/%s/command", c.baseTopic(), switchId)
}

func (c *MQTTClient) InputNumberStateTopic(id string) string {
	return fmt.Sprintf("%s/number/%s/state", c.baseTopic(), id)
}

func (c *MQTTClient) InputNumberCommandTopic(id string) string {
	return fmt.Sprintf("%s/number/%s/set", c.baseTopic(), id)
}

func (c *MQTTClient) ParseMQTTCommand(msg mqtt.Message) (*ParsedMQTTCommand, error) {
	switchCmd, err := c.parseSwitchMQTTCommand(msg)
	if err == nil {
		return switchCmd, nil
	}
	inputNumberCmd, err := c.parseInputNumberMQTTCommand(msg)
	if err == nil {
		return inputNumberCmd, nil
	}
	return nil, err
}

func (c *MQTTClient) parseSwitchMQTTCommand(msg mqtt.Message) (*ParsedMQTTCommand, error) {
	topic := msg.Topic()
	matches := c.switchCommandRegexp.FindAllStringSubmatch(topic, 1)
	if len(matches) == 0 {
		return nil, errors.New("invalid command")
	}
	if len(matches[0]) != 2 {
		return nil, errors.New("invalid switch command")
	}
	return &ParsedMQTTCommand{
		DeviceId: matches[0][1],
		Command:  "switch",
		Payload:  string(msg.Payload()),
	}, nil
}

func (c *MQTTClient) parseInputNumberMQTTCommand(msg mqtt.Message) (*ParsedMQTTCommand, error) {
	topic := msg.Topic()
	matches := c.inputNumberCommandRegexp.FindAllStringSubmatch(topic, 1)
	if len(matches) == 0 {
		return nil, errors.New("invalid command")
	}
	if len(matches[0]) != 2 {
		return nil, errors.New("invalid switch command")
	}

	// try to parse a valid number
	_, err := strconv.ParseFloat(string(msg.Payload()), 64)
	if err != nil {
		return nil, err
	}

	return &ParsedMQTTCommand{
		DeviceId: matches[0][1],
		Command:  "number",
		Payload:  string(msg.Payload()),
	}, nil
}

func (c *MQTTClient) Publish(topic string, payload any, qos byte, retain bool, continuation func(error), timeout time.Duration) {
	token := c.client.Publish(topic, qos, retain, payload)
	go func() {
		didTO := token.WaitTimeout(timeout)
		if !didTO {
			continuation(errors.New("MQTT publish timed out"))
		} else {
			continuation(token.Error())
		}
	}()
}

func (c *MQTTClient) Subscribe(topic string, qos byte, handler mqtt.MessageHandler, continuation func(error), timeout time.Duration) {
	token := c.client.Subscribe(topic, qos, handler)
	go func() {
		didTO := token.WaitTimeout(timeout)
		if !didTO {
			continuation(errors.New("MQTT subscribe timed out"))
		} else {
			continuation(token.Error())
		}
	}()
}

func (c *MQTTClient) SubscribeToCommandTopic(handler mqtt.MessageHandler, continuation func(error), timeout time.Duration) {
	c.Subscribe(c.commandTopic(), 1, handler, continuation, timeout)
}

func (c *MQTTClient) Unsubscribe(topic string, continuation func(error), timeout time.Duration) {
	token := c.client.Unsubscribe(topic)
	go func() {
		didTO := token.WaitTimeout(timeout)
		if !didTO {
			continuation(errors.New("MQTT unsubscribe timed out"))
		} else {
			continuation(token.Error())
		}
	}()
}

func (c *MQTTClient) Connect(continuation func(error), timeout time.Duration) {
	token := c.client.Connect()
	go func() {
		didTO := token.WaitTimeout(timeout)
		if !didTO {
			continuation(errors.New("MQTT connect timed out"))
		} else {
			continuation(token.Error())
		}
	}()
}

func (c *MQTTClient) Disconnect(timeout time.Duration) {
	c.client.Disconnect(uint(timeout.Milliseconds()))
}

func (c *MQTTClient) commandTopic() string {
	return fmt.Sprintf("%s/#", c.baseTopic())
}

func switchCommandExtractor(baseTopic string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf("%s/switch/([a-zA-Z0-9_]+)/command", baseTopic))
}

func inputNumberCommandExtractor(baseTopic string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf("%s/number/([a-zA-Z0-9_]+)/set", baseTopic))
}

func bridgeStateTopic(baseTopic string) string {
	return fmt.Sprintf("%s/bridge/state", baseTopic)
}
