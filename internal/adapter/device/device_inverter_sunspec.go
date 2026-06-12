package device

import (
	"context"
	"errors"
	"time"

	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/pkg/modbus"
	"github.com/berfenger/frostnews2mqtt/pkg/sunspec"
	"github.com/berfenger/frostnews2mqtt/pkg/util/logutil"
)

type GenericInverterClient struct {
	ctx          context.Context
	modbusClient modbus.ModbusReaderWriter
	device       *sunspec.DeviceInverter
	isOpen       bool
}

func NewSunspecInverterClient(ctx context.Context, ip string, port uint, inverterAddress uint8, timeout time.Duration,
	instrumentations []modbus.ModbusInstrumentation) (*GenericInverterClient, error) {

	// create modbus client
	client, err := modbus.NewAutoReconnectModbusTCPReaderWriterClient("inverter", ip, port, inverterAddress, timeout, logutil.FromContext(ctx), instrumentations)
	if err != nil {
		return nil, err
	}

	// create reader instance
	dev := GenericInverterClient{
		ctx:          ctx,
		modbusClient: client,
		isOpen:       false,
	}
	return &dev, nil
}

func (inverter *GenericInverterClient) Open() error {
	if err := inverter.modbusClient.Open(); err != nil {
		return err
	}
	deviceModels, err := sunspec.SurveyDeviceAuto(inverter.ctx, inverter.modbusClient)
	if err != nil {
		return err
	}
	inverter.device, err = sunspec.NewSunSpecDeviceInverter(inverter.ctx, inverter.modbusClient, *deviceModels)
	if err != nil {
		return err
	}
	inverter.isOpen = true
	return nil
}

func (inverter *GenericInverterClient) IsOpen() bool {
	return inverter.isOpen
}

func (inverter *GenericInverterClient) Close() error {
	return inverter.modbusClient.Close()
}

func (inverter *GenericInverterClient) GetInfo() (*domain.InverterInfo, error) {

	commonData, err := inverter.device.ReadCommonModel()
	if err != nil {
		return nil, err
	}

	statusData, err := inverter.device.ReadStatusModel()
	if err != nil {
		return nil, err
	}

	nameplateData, err := inverter.device.ReadNameplateModel()
	if err != nil {
		return nil, err
	}

	return &domain.InverterInfo{
		Manufacturer:      commonData.Manufacturer(),
		Model:             commonData.Model(),
		Version:           commonData.Version(),
		Serial:            commonData.Serial(),
		MaxRatedPowerWatt: nameplateData.MaxRatedACPowerWatt(),
		HasStorage:        statusData.StorageConnection().Connected() && inverter.device.Storage != nil,
	}, nil
}

func (inverter *GenericInverterClient) GetState() (*domain.InverterState, error) {

	inverterData, err := inverter.device.ReadInverterModel()
	if err != nil {
		return nil, err
	}

	return &domain.InverterState{
		CabinetTemperature:   inverterData.CabinetTemperature(),
		OperatingState:       inverterData.OperatingState(),
		OperatingStateStr:    inverterData.OperatingStateString(),
		VendorOperatingState: inverterData.VendorOperatingState(),
		SunspecEvent1:        inverterData.Events1(),
		SunspecDeviceEvents:  inverterData.Events1Parsed(),
		SunspecVendorEvent1:  inverterData.VendorDefinedEvents1(),
		SunspecVendorEvent2:  inverterData.VendorDefinedEvents2(),
		SunspecVendorEvent3:  inverterData.VendorDefinedEvents3(),
		SunspecVendorEvent4:  inverterData.VendorDefinedEvents4(),
	}, nil
}

func (inverter *GenericInverterClient) GetPowerFlow() (*domain.InverterPowerFlow, error) {

	inverterData, err := inverter.device.ReadInverterModel()
	if err != nil {
		return nil, err
	}

	mpptData, err := inverter.device.ReadMPPTModel()
	if err != nil {
		return nil, err
	}

	modules := mpptData.MPPTModulesData()
	var dcpower float64 = 0
	var chargeDCPower float64 = 0
	var dischargeDCPower float64 = 0

	if len(modules) > 0 { // 1 MPPT + Battery or just 1 MPPT
		dcpower += modules[0].DCPower
	}
	if len(modules) == 2 || len(modules) == 4 { // 2 MPPT + Battery or just 2 MPPT
		dcpower += modules[1].DCPower
	}
	if len(modules) == 3 || len(modules) == 4 { // 1 or 2 MPPT + Battery
		chargeDCPower = modules[len(modules)-2].DCPower
		dischargeDCPower = modules[len(modules)-1].DCPower
	}

	return &domain.InverterPowerFlow{
		ACPowerWatt:               inverterData.ACPowerWatt(),
		PVPowerWatt:               dcpower,
		BatteryChargePowerWatt:    chargeDCPower,
		BatteryDischargePowerWatt: dischargeDCPower,
		BatteryDCPowerFlowWatt:    dischargeDCPower - chargeDCPower,
	}, nil
}

func (inverter *GenericInverterClient) SetPowerLimit(powerLimit domain.InverterPowerLimit) error {

	if inverter.device.Controls == nil {
		return errors.New("controls model not supported")
	}

	return inverter.device.WritableControlsModel().SetPowerLimit(powerLimit.Enabled, powerLimit.Percent, powerLimit.RevertTimeSeconds)
}

func (inverter *GenericInverterClient) GetPowerLimit() (*domain.InverterPowerLimit, error) {

	if inverter.device.Controls == nil {
		return nil, errors.New("controls model not supported")
	}

	data, err := inverter.device.Controls.Read()
	if err != nil {
		return nil, err
	}

	return &domain.InverterPowerLimit{
		Enabled:           data.PowerLimitEnabled(),
		Percent:           data.PowerLimitPercent(),
		RevertTimeSeconds: data.RevertTimeSeconds(),
	}, nil
}

func (inverter *GenericInverterClient) HasStorage() (bool, error) {

	statusData, err := inverter.device.ReadStatusModel()
	if err != nil {
		return false, err
	}

	return statusData.StorageConnection().Connected() && inverter.device.Storage != nil, nil
}

func (inverter *GenericInverterClient) SupportsPowerControl() (bool, error) {
	return inverter.device.Controls != nil, nil
}

func (inverter *GenericInverterClient) SetStorageControl(params domain.StorageControlParams) error {
	if inverter.device.Storage == nil {
		return errors.New("storage model not supported")
	}

	return inverter.device.WritableStorageModel().SetStorageControl(
		params.MinChargePowerWatt,
		params.MaxChargePowerWatt,
		params.MinDischargePowerWatt,
		params.MaxDischargePowerWatt,
		params.RevertTimeSeconds,
	)
}

func (inverter *GenericInverterClient) SetStorageForceChargePower(watts uint16, revertTimeSeconds int32) error {

	return inverter.SetStorageControl(domain.StorageControlParams{
		MinChargePowerWatt:    int32(watts),
		MaxChargePowerWatt:    -1,
		MinDischargePowerWatt: -1,
		MaxDischargePowerWatt: -1,
		RevertTimeSeconds:     uint32(revertTimeSeconds),
	})
}

func (inverter *GenericInverterClient) SetStorageForceDischargePower(watts uint16, revertTimeSeconds int32) error {

	return inverter.SetStorageControl(domain.StorageControlParams{
		MinChargePowerWatt:    -1,
		MaxChargePowerWatt:    -1,
		MinDischargePowerWatt: int32(watts),
		MaxDischargePowerWatt: -1,
		RevertTimeSeconds:     uint32(revertTimeSeconds),
	})
}

func (inverter *GenericInverterClient) DisableStorageControl() error {

	if inverter.device.Storage == nil {
		return errors.New("storage control not supported")
	}

	return inverter.device.Storage.DisableStorageControl()
}

func (inverter *GenericInverterClient) GetStorageState() (*domain.StorageState, error) {

	if inverter.device.Storage == nil {
		return nil, errors.New("storage model not available")
	}

	data, err := inverter.device.Storage.Read()
	if err != nil {
		return nil, err
	}

	return &domain.StorageState{
		StateOfCharge:       data.StateOfCharge(),
		MaxCapacityWatt:     data.MaxCapacityWatt(),
		CurrentCapacityWatt: data.CurrentCapacityWatt(),
		ChargeStatus:        data.ChargeStatus(),
		ChargeStatusStr:     data.ChargeStatusString(),
	}, nil
}

// ensure interface compliance
var _ domain.InverterDevice = (*GenericInverterClient)(nil)
