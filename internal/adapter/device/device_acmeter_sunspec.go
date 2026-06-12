package device

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/pkg/modbus"
	"github.com/berfenger/frostnews2mqtt/pkg/sunspec"
	"github.com/berfenger/frostnews2mqtt/pkg/util/logutil"
	"go.uber.org/zap"
)

type GenericACMeterClient struct {
	ctx          context.Context
	modbusClient modbus.ModbusReaderWriter
	device       *sunspec.DeviceACMeter
	isOpen       bool
}

func NewACMeterClient(ctx context.Context, ip string, port uint, acMeterAddress uint8,
	timeout time.Duration, instrumentations []modbus.ModbusInstrumentation) (domain.ACMeterModbusReader, error) {

	// create modbus client
	client, err := modbus.NewAutoReconnectModbusTCPReaderWriterClient("acMeter", ip, port, acMeterAddress, timeout, logutil.FromContext(ctx), instrumentations)
	if err != nil {
		return nil, err
	}

	// create reader instance
	dev := GenericACMeterClient{
		ctx:          ctx,
		modbusClient: client,
		isOpen:       false,
	}
	return &dev, nil
}

func (acMeter *GenericACMeterClient) Open() error {
	if err := acMeter.modbusClient.Open(); err != nil {
		return err
	}
	deviceModels, err := sunspec.SurveyDeviceAuto(acMeter.ctx, acMeter.modbusClient)
	if err != nil {
		return err
	}
	acMeter.device, _ = sunspec.NewSunSpecDeviceACMeter(acMeter.ctx, acMeter.modbusClient, *deviceModels)
	acMeter.isOpen = true
	return nil
}

func (acMeter *GenericACMeterClient) IsOpen() bool {
	return acMeter.isOpen
}

func (acMeter *GenericACMeterClient) Close() error {
	return acMeter.modbusClient.Close()
}

func (acMeter GenericACMeterClient) GetInfo() (*domain.ACMeterInfo, error) {
	data, err := acMeter.device.ReadCommonModel()
	if err != nil {
		return nil, err
	}

	manufacturer := data.Manufacturer()
	model := data.Model()
	version := data.Version()
	serial := data.Serial()

	return &domain.ACMeterInfo{
		Manufacturer: manufacturer,
		Model:        model,
		Version:      version,
		Serial:       serial,
	}, nil
}

func (acMeter GenericACMeterClient) GetCurrentPowerFlowWatt() (float64, error) {
	data, err := acMeter.device.ReadACMeterModel()
	if err != nil {
		return 0, err
	}
	if data.Events() != 0 {
		logutil.FromContext(acMeter.ctx).Warn("AC Meter reported events %s", zap.String("events", fmt.Sprintf("%X", data.Events())))
		return 0, fmt.Errorf("invalid ac meter read: %X", data.Events())
	}
	return data.GetCurrentPowerFlowWatt(), nil
}

func (acMeter GenericACMeterClient) GetPowerFlow() (*domain.ACMeterPowerFlow, error) {
	data, err := acMeter.device.ReadACMeterModel()
	if err != nil {
		return nil, err
	}
	if data.Events() != 0 {
		logutil.FromContext(acMeter.ctx).Warn("AC Meter reported events %s", zap.String("events", fmt.Sprintf("%X", data.Events())))
		return nil, fmt.Errorf("invalid ac meter read: %X", data.Events())
	}
	totalRealPower := data.GetCurrentPowerFlowWatt()
	totalEnergyExported := data.GetTotalEnergyExported()
	totalEnergyImported := data.GetTotalEnergyImported()
	freq := data.GetGridFrequency()
	phaseAVoltage := data.GetPhaseAVoltage()
	var importPower float64 = 0
	var exportPower float64 = 0
	if totalRealPower < 0 {
		exportPower = math.Abs(totalRealPower)
	} else {
		importPower = totalRealPower
	}

	return &domain.ACMeterPowerFlow{
		CurrentPowerFlowWatt:   totalRealPower,
		CurrentImportPowerWatt: importPower,
		CurrentExportPowerWatt: exportPower,
		TotalEnergyExportedKWh: totalEnergyExported,
		TotalEnergyImportedKWh: totalEnergyImported,
		Frequency:              freq,
		PhaseAVoltage:          phaseAVoltage,
	}, nil
}

// ensure interface compliance
var _ domain.ACMeterModbusReader = (*GenericACMeterClient)(nil)
