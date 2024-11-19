package util

import (
	"frostnews2mqtt/internal/config"

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
		PowerFlowPollIntervalMillis:        5000,
		TrackHousePower:                    true,
		MaxImportPower:                     4000,
		BatteryControlRevertTimeoutSeconds: 10,
		FeedInControlRevertTimeoutSeconds:  10,
		Port:                               8080,
	}
}
