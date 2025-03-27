package config

import (
	"errors"
	"regexp"
	"strings"

	"go.uber.org/zap/zapcore"
)

type Config struct {
	LogLevel          zapcore.Level
	InverterModbusTcp InverterModbusTCPConfig `mapstructure:"inverter_modbus_tcp"`
	MQTT              MQTTConfig              `mapstructure:"mqtt"`

	GridConfig           GridConfig           `mapstructure:"grid"`
	BatteryControlConfig BatteryControlConfig `mapstructure:"battery_control"`
	MonitorConfig        MonitorConfig        `mapstructure:"monitor"`
	Port                 uint                 `mapstructure:"port"`
	HttpLog              bool                 `mapstructure:"http_log"`
}

type InverterModbusTCPConfig struct {
	Host                       string
	Port                       uint
	MeterId                    uint   `mapstructure:"meter_id"`
	InverterId                 uint   `mapstructure:"inverter_id"`
	IgnoreFronius              bool   `mapstructure:"ignore_fronius"`
	ReadDelayAfterChangeMillis uint32 `mapstructure:"read_delay_after_change_millis"`
}

type MonitorConfig struct {
	TrackHousePower    bool   `mapstructure:"track_house_power"`
	PollIntervalMillis uint32 `mapstructure:"poll_interval_millis"`
}

type GridConfig struct {
	MaxImportPower uint32 `mapstructure:"max_import_power"`
}

type BatteryControlConfig struct {
	ControlIntervalMillis uint32 `mapstructure:"control_interval_millis"`
	StartPowerThreshold   uint32 `mapstructure:"start_power_threshold"`
	MaxRatePowerIncrease  uint32 `mapstructure:"max_rate_power_increase"`
	SafetyMarginPower     uint32 `mapstructure:"safety_margin_power"`
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
