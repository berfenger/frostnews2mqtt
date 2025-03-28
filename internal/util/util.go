package util

import (
	"github.com/berfenger/frostnews2mqtt/internal/config"

	"go.uber.org/zap"
)

func LoadTestConfig() config.Config {
	return config.Config{
		LogLevel: zap.DebugLevel,
		InverterModbusTcp: config.InverterModbusTCPConfig{
			Host:       "-.-.-.-",
			Port:       502,
			MeterId:    200,
			InverterId: 0,
		},
		MQTT: config.MQTTConfig{
			Host: "localhost",
			Port: 1883,
		},
		BatteryControlConfig: config.BatteryControlConfig{
			ControlIntervalMillis: 10000,
			StartPowerThreshold:   500,
			MaxRatePowerIncrease:  800,
		},
		GridConfig: config.GridConfig{
			MaxImportPower: 4000,
		},
		MonitorConfig: config.MonitorConfig{
			TrackHousePower:    true,
			PollIntervalMillis: 5000,
		},
		Port: 8080,
	}
}
