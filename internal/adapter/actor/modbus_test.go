package actor

import (
	"frostnews2mqtt/internal/core/domain"
	"frostnews2mqtt/internal/util/actorutil"
	"frostnews2mqtt/pkg/sunspec_modbus"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGetDevicesInfoModbusActor(t *testing.T) {

	assert := assert.New(t)

	inv, err := sunspec_modbus.CreateTestInverterModbusReader()
	if err != nil {
		t.Error(err)
		return
	}

	acMeter, err := sunspec_modbus.CreateTestACMeterModbusReader()
	if err != nil {
		t.Error(err)
		return
	}

	logger := zap.Must(zap.NewDevelopment())

	as := actorutil.NewActorSystemWithZapLogger(logger)

	context := as.Root

	props := actor.PropsFromProducer(func() actor.Actor { return NewModbusActor(inv, acMeter, logger) })
	pid := context.Spawn(props)

	time.Sleep(1 * time.Second)

	msg := domain.GetDevicesInfoRequest{}
	result, err := context.RequestFuture(pid, msg, 15*time.Second).Result()
	if err != nil {
		t.Error(err)
		return
	}
	resp := result.(domain.GetDevicesInfoResponse)

	assert.Equal(resp.Inverter.Manufacturer, "Frostnews", "Inverter manufacturer")
	assert.Equal(resp.Inverter.Model, "Primo GEN24 4.0", "Inverter model")
	assert.Equal(resp.Inverter.Version, "1.30.7-1", "Inverter version")
	assert.Equal(resp.ACMeter.Manufacturer, "Frostnews", "Inverter manufacturer")
	assert.Equal(resp.ACMeter.Model, "Smart Meter TS 100A-1", "Inverter model")
	assert.Equal(resp.ACMeter.Version, "1.2", "Inverter version")

	context.Stop(pid)

	as.Shutdown()
}

func TestGetPowerFlowoModbusActor(t *testing.T) {

	assert := assert.New(t)

	inv, err := sunspec_modbus.CreateTestInverterModbusReader()
	if err != nil {
		t.Error(err)
		return
	}

	acMeter, err := sunspec_modbus.CreateTestACMeterModbusReader()
	if err != nil {
		t.Error(err)
		return
	}

	logger := zap.Must(zap.NewDevelopment())

	as := actorutil.NewActorSystemWithZapLogger(logger)
	context := as.Root

	props := actor.PropsFromProducer(func() actor.Actor { return NewModbusActor(inv, acMeter, logger) })
	pid := context.Spawn(props)

	time.Sleep(1 * time.Second)

	msg := domain.GetPowerFlowRequest{}

	result, err := context.RequestFuture(pid, msg, 15*time.Second).Result()
	if err != nil {
		t.Error(err)
		return
	}
	resp := result.(domain.GetPowerFlowResponse)

	assert.True(resp.Inverter.ACPowerWatt > 0, "ACPowerWatt bounds")
	assert.True(resp.Inverter.BatteryChargePowerWatt >= 0, "BatteryChargePowerWatt bounds")
	assert.True(resp.Inverter.BatteryDischargePowerWatt >= 0, "BatteryDischargePowerWatt bounds")
	assert.Equal(resp.Inverter.BatteryDCPowerFlowWatt, resp.Inverter.BatteryDischargePowerWatt-resp.Inverter.BatteryChargePowerWatt, "BatteryDCPowerFlowWatt value")

	context.Stop(pid)

	as.Shutdown()
}
