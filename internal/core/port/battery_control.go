package port

import (
	"github.com/berfenger/frostnews2mqtt/internal/core/domain"
)

type BatteryChargeControlLogic interface {
	Loop(prevPowerValue int32, storageState *domain.StorageState,
		acMeterPowerFlow *domain.ACMeterPowerFlow,
		inverterPowerFlow *domain.InverterPowerFlow,
		targetSoC uint8) domain.BatteryChargeControlTickResult
	SetMaxGridImportPower(powerWatt uint32)
	MaxGridImportPower() uint32
}
