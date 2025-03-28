package port

import (
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
	"github.com/berfenger/frostnews2mqtt/pkg/sunspec_modbus"
)

type BatteryChargeControlLogic interface {
	Loop(prevPowerValue int32, storageState *sunspec_modbus.StorageState,
		acMeterPowerFlow *sunspec_modbus.ACMeterPowerFlow,
		inverterPowerFlow *sunspec_modbus.InverterPowerFlow,
		targetSoC uint8) domain.BatteryChargeControlTickResult
	SetMaxGridImportPower(powerWatt uint32)
	MaxGridImportPower() uint32
}
