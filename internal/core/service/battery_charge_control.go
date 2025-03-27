package service

import (
	"frostnews2mqtt/internal/core/domain"
	"frostnews2mqtt/internal/core/port"
	"frostnews2mqtt/pkg/sunspec_modbus"
	"math"

	"go.uber.org/zap"
)

type DefaultBatteryControlLogic struct {
	StartPowerThreshold     uint32
	MaxRatePowerIncrease    uint32
	PowerImportSafetyMargin uint32
	MaxImportPower          uint32
	Logger                  *zap.Logger
}

func (cfg *DefaultBatteryControlLogic) Loop(prevPowerValue int32, storageState *sunspec_modbus.StorageState,
	acMeterPowerFlow *sunspec_modbus.ACMeterPowerFlow,
	inverterPowerFlow *sunspec_modbus.InverterPowerFlow,
	targetSoC uint8) domain.BatteryChargeControlTickResult {

	if storageState.StateOfCharge >= float64(targetSoC) {
		// when battery SoC target is reached, disable force charge
		cfg.Logger.Info("battery_control@charging: charge targetSoC is met. Turning off charging control.")
		return domain.BatteryChargeControlTickResult{
			NewPowerValue: -1,
			Exit:          true,
		}
	} else {

		realPVPower := math.Max(0, inverterPowerFlow.PVPowerWatt)

		newPowerValue := float64(prevPowerValue)
		if prevPowerValue == -1 {
			// track house power consumption
			houseConsumption := inverterPowerFlow.ACPowerWatt + acMeterPowerFlow.CurrentPowerFlowWatt

			/* powerBudget = grid.maxImportPower + PV power - House consumption
			 * Battery should be ignored because while charging the battery,
			 * it cannot be discharged at the same time to meet house power requirements
			 */
			powerBudget := float64(cfg.MaxImportPower) - float64(cfg.PowerImportSafetyMargin) // max import from grid
			powerBudget += realPVPower                                                        // + solar
			powerBudget -= houseConsumption                                                   // - house consumption

			// to start, powerBudget should be greater than solar production
			if powerBudget > realPVPower &&
				powerBudget > float64(cfg.StartPowerThreshold) {
				// newPowerValue = MIN(MAX(maxRate, PVPower), powerBudget)
				newPowerValue = math.Min(math.Max(float64(cfg.MaxRatePowerIncrease), realPVPower), powerBudget)
			} else {
				newPowerValue = -1
			}
		} else {
			// adjust charge power
			availablePower := float64(cfg.MaxImportPower) - float64(cfg.PowerImportSafetyMargin) - acMeterPowerFlow.CurrentImportPowerWatt
			newPowerValue += math.Min(float64(cfg.MaxRatePowerIncrease), availablePower)
			if newPowerValue < realPVPower {
				newPowerValue = -1
			}
		}

		// check bounds
		// max value. charge power cannot exceed battery max capacity (an error is returned otherwise)
		newPowerValue = math.Min(float64(storageState.MaxCapacityWatt), newPowerValue)
		// min value
		// if negative, temp disable
		if newPowerValue <= 0 {
			newPowerValue = -1
		}
		return domain.BatteryChargeControlTickResult{
			NewPowerValue: int32(newPowerValue),
			Exit:          false,
		}
	}
}

func (cfg *DefaultBatteryControlLogic) MaxGridImportPower() uint32 {
	return cfg.MaxImportPower
}

func (cfg *DefaultBatteryControlLogic) SetMaxGridImportPower(powerWatt uint32) {
	cfg.MaxImportPower = powerWatt
}

// ensure interface compliance
var _ port.BatteryChargeControlLogic = (*DefaultBatteryControlLogic)(nil)
