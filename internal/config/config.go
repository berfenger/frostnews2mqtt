package config

import (
	"errors"
	"regexp"
	"strings"

	"go.uber.org/zap/zapcore"
)

type Config struct {
	LogLevel                           zapcore.Level
	InverterModbusTcp                  InverterModbusTCPConfig `mapstructure:"inverter_modbus_tcp"`
	MQTT                               MQTTConfig              `mapstructure:"mqtt"`
	PowerFlowPollIntervalMillis        uint32                  `mapstructure:"power_flow_poll_interval_millis"`
	TrackHousePower                    bool                    `mapstructure:"track_house_power"`
	MaxImportPower                     uint                    `mapstructure:"max_import_power"`
	BatteryControlRevertTimeoutSeconds uint                    `mapstructure:"battery_control_revert_timeout_seconds"`
	FeedInControlRevertTimeoutSeconds  uint                    `mapstructure:"feedin_control_revert_timeout_seconds"`
	Port                               uint                    `mapstructure:"port"`
	HttpLog                            bool                    `mapstructure:"http_log"`
}

type InverterModbusTCPConfig struct {
	Host          string
	Port          uint
	MeterId       uint `mapstructure:"meter_id"`
	InverterId    uint `mapstructure:"inverter_id"`
	IgnoreFronius bool `mapstructure:"ignore_fronius"`
}

type MQTTConfig struct {
	Host              string
	Port              int
	Username          string
	Password          string
	BaseTopic         string `mapstructure:"base_topic"`
	HADiscoveryEnable bool   `mapstructure:"ha_discovery_enable"`
	HADiscoveryTopic  string `mapstructure:"ha_discovery_topic"`
}

func CheckMQTTTopic(baseTopic string) (string, error) {
	// check and fix base topic
	lowerBaseTopic := strings.ToLower(baseTopic)
	baseTopicRegexp := regexp.MustCompile("^[a-z0-9_]+$")
	matches := baseTopicRegexp.FindAllStringSubmatch(lowerBaseTopic, 1)
	if len(matches) <= 0 {
		return "", errors.New("invalid topic. can only contain letters, numbers and underscores")
	}
	return lowerBaseTopic, nil
}
