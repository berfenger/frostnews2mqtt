package device

import (
	"context"
	"strings"
	"time"

	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/pkg/fronius"
	"github.com/berfenger/frostnews2mqtt/pkg/modbus"
)

// Fronius Inverter

type FroniusInverterClient struct {
	GenericInverterClient
	Model *fronius.FroniusModel
}

func NewFroniusInverterClient(
	ctx context.Context, ip string, port uint, inverterAddress uint8,
	timeout time.Duration, instrumentations []modbus.ModbusInstrumentation,
) (*FroniusInverterClient, error) {

	baseReader, err := NewSunspecInverterClient(ctx, ip, port, inverterAddress, timeout, instrumentations)
	if err != nil {
		return nil, err
	}

	fron := &FroniusInverterClient{
		GenericInverterClient: *baseReader,
	}

	return fron, nil
}

func (reader *FroniusInverterClient) GetInfo() (*domain.InverterInfo, error) {

	state, err := reader.GenericInverterClient.GetInfo()
	if err != nil {
		return nil, err
	}
	state.HasVendorProfile = true
	if reader.Model == nil {
		model := parseFroniusModel(state.Model)
		reader.Model = &model
	}

	return state, nil
}

func (fron *FroniusInverterClient) GetVendorState() (*domain.VendorInverterState, error) {

	state, err := fron.GetState()
	if err != nil {
		return nil, err
	}

	fronState := fronius.FroniusInverterStatus(state.VendorOperatingState)

	var model fronius.FroniusModel
	if fron.Model != nil {
		model = *fron.Model
	} else {
		model = fronius.UnknownModel
	}

	return &domain.VendorInverterState{
		VendorOperatingStateStr: fronState.String(),
		VendorDeviceEvents:      fronius.ParseEvents(state.SunspecVendorEvent1, state.SunspecVendorEvent2, state.SunspecVendorEvent3, model),
	}, nil
}

func parseFroniusModel(name string) fronius.FroniusModel {
	lwrName := strings.ToLower(name)
	if strings.Contains(lwrName, "primo") {
		return fronius.Primo
	} else if strings.Contains(lwrName, "symo") {
		return fronius.Symo
	} else if strings.Contains(lwrName, "galvo") {
		return fronius.Galvo
	} else if strings.Contains(lwrName, "igplus") {
		return fronius.IGPlus
	} else {
		return fronius.UnknownModel
	}
}

var _ domain.VendorInverterDevice = (*FroniusInverterClient)(nil)
