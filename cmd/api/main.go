package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"frostnews2mqtt/internal/actor"
	"frostnews2mqtt/internal/config"
	"frostnews2mqtt/internal/server"
	"frostnews2mqtt/pkg/sunspec_modbus"

	pactor "github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {

	// load and print config
	cfg, err := initConfig()
	if err != nil {
		slog.Error("config errors", "error", err)
		return
	}
	safePrintConfig(*cfg)

	// init actor system
	as := pactor.NewActorSystem()
	ctx := as.Root
	logger := logrus.New()
	logger.SetLevel(cfg.LogLevel)

	// init Modbus actor provider
	modbusProv, err := modbusActorProvider(cfg, logger)
	if err != nil {
		panic(err)
	}

	props := pactor.PropsFromProducer(func() pactor.Actor {
		return actor.NewMasterOfPuppetsActor(*cfg, modbusProv, mqttActorProvider(cfg, logger), logger)
	})
	pid, err := ctx.SpawnNamed(props, "master")
	if err != nil {
		return
	}

	server := server.NewServer(*cfg, ctx, pid)
	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(server, done)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")

	ctx.Stop(pid)
	as.Shutdown()
}

func initConfig() (*config.Config, error) {

	setConfigDefaults()

	viper.SetEnvPrefix("frostnews")
	viper.AutomaticEnv()

	// if defined, try to load config from yaml file
	if cfgFile := os.Getenv("CONFIG_FILE"); cfgFile != "" {
		if _, err := os.Stat(cfgFile); err == nil {
			slog.Info("Using config", "file", cfgFile)
			viper.SetConfigFile(cfgFile)

			err = viper.ReadInConfig()
			if err != nil {
				slog.Error("Error reading config file", "error", err)
			}
		}
	}

	var cfg config.Config

	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	// parse log level
	switch viper.GetString("log_level") {
	case "trace":
		cfg.LogLevel = logrus.TraceLevel
	case "debug":
		cfg.LogLevel = logrus.DebugLevel
	case "info":
		cfg.LogLevel = logrus.InfoLevel
	case "error":
		cfg.LogLevel = logrus.ErrorLevel
	case "warn":
		cfg.LogLevel = logrus.WarnLevel
	case "fatal":
		cfg.LogLevel = logrus.FatalLevel
	default:
		cfg.LogLevel = logrus.InfoLevel
	}

	// check and fix base topic
	baseTopic, err := config.CheckMQTTTopic(cfg.MQTT.BaseTopic)
	if err != nil {
		return nil, errors.New("invalid base topic. can only contain letters, numbers and underscores")
	}
	cfg.MQTT.BaseTopic = baseTopic

	// check and fix homeassistant discovery topic
	hadBaseTopic, err := config.CheckMQTTTopic(cfg.MQTT.HADiscoveryTopic)
	if err != nil {
		return nil, errors.New("invalid homeassistant discovery topic. can only contain letters, numbers and underscores")
	}
	cfg.MQTT.HADiscoveryTopic = hadBaseTopic

	// check bounds
	if cfg.BatteryControlRevertTimeoutSeconds <= 0 {
		return nil, errors.New("config param battery_control_revert_timeout_seconds should be greater than zero")
	}
	if cfg.FeedInControlRevertTimeoutSeconds <= 0 {
		return nil, errors.New("config param feedin_control_revert_timeout_seconds should be greater than zero")
	}
	if cfg.MaxImportPower <= 0 {
		return nil, errors.New("config param max_import_power should be greater than zero")
	}

	return &cfg, nil
}

func modbusActorProvider(cfg *config.Config, logger *logrus.Logger) (actor.ModbusActorProvider, error) {

	inv, err := sunspec_modbus.CreateInverterIntSFModbusReader(cfg.InverterModbusTcp.Host,
		cfg.InverterModbusTcp.Port, uint8(cfg.InverterModbusTcp.InverterId), 1*time.Second,
		cfg.InverterModbusTcp.IgnoreFronius, logger, nil)

	if err != nil {
		return nil, err
	}

	acMeter, err := sunspec_modbus.CreateACMeterIntSFModbusReader(cfg.InverterModbusTcp.Host,
		cfg.InverterModbusTcp.Port, uint8(cfg.InverterModbusTcp.MeterId), 1*time.Second,
		cfg.InverterModbusTcp.IgnoreFronius, logger, nil)

	if err != nil {
		return nil, err
	}

	return func() *actor.ModbusActor {
		return actor.NewModbusActor(inv, acMeter, logger)
	}, nil
}

func mqttActorProvider(cfg *config.Config, logger *logrus.Logger) actor.MQTTActorProvider {
	return func(es *eventstream.EventStream) *actor.MQTTActor {
		return actor.NewMQTTActor(cfg, es, logger)
	}
}

func setConfigDefaults() {
	viper.SetDefault("log_level", "warn")
	viper.SetDefault("mqtt.ha_discovery_enable", false)
	viper.SetDefault("mqtt.base_topic", "frostnews")
	viper.SetDefault("mqtt.ha_discovery_topic", "homeassistant")
	viper.SetDefault("power_flow_poll_interval_millis", 5000)
	viper.SetDefault("track_house_power", false)
	viper.SetDefault("max_import_power", 0)
	viper.SetDefault("battery_control_revert_timeout_seconds", 30)
	viper.SetDefault("feedin_control_revert_timeout_seconds", 30)
	viper.SetDefault("port", 8080)
}

func safePrintConfig(cfg config.Config) {
	cfg.MQTT.Username = "*redacted*"
	cfg.MQTT.Password = "*redacted*"
	slog.Info("Using", "config", cfg)
}
