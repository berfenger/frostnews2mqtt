package port

import (
	"frostnews2mqtt/internal/core/domain"
	"frostnews2mqtt/pkg/sunspec_modbus"
)

type BatteryChargeControlLogic interface {
	Loop(prevPowerValue int32, storageState *sunspec_modbus.StorageState,
		acMeterPowerFlow *sunspec_modbus.ACMeterPowerFlow,
		inverterPowerFlow *sunspec_modbus.InverterPowerFlow,
		targetSoC uint8) domain.BatteryChargeControlTickResult
	SetMaxGridImportPower(powerWatt uint32)
	MaxGridImportPower() uint32
}
