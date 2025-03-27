package service

import (
	"fmt"
	"frostnews2mqtt/internal/core/domain"
	"frostnews2mqtt/internal/core/port"
	"frostnews2mqtt/pkg/sunspec_modbus"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	BATTERY_MAX_CAP = 5120
	MAX_LOOP_INTER  = 100
)

func TestSequencesLowSolar(t *testing.T) {

	require := require.New(t)

	// check power increments until there is no change
	c := runIncrements(require, ctrl, 200, 2300, 5)
	assert.GreaterOrEqual(t, c, 1)

	// check power decrements until the battery stops charging
	// house power is increasing to force battery charge to decrease
	c = runDecrements(require, ctrl, 200, 1700, 5, 100-float64(ctrl.PowerImportSafetyMargin))
	assert.GreaterOrEqual(t, c, 2)
}

func TestSequencesHighSolar(t *testing.T) {

	require := require.New(t)

	// check power increments until there is no change
	c := runIncrements(require, ctrl, 2000, 300, 5)
	assert.GreaterOrEqual(t, c, 1)

	// check power decrements until the battery stops charging
	// house power is increasing to force battery charge to decrease
	c = runDecrements(require, ctrl, 2000, 3000, 5, 100-float64(ctrl.PowerImportSafetyMargin))
	assert.GreaterOrEqual(t, c, 2)
}

func TestStopWhenTargetSoCMet(t *testing.T) {

	require := require.New(t)

	r := callControllerAndCheckPower(require, ctrl, ss(100, BATTERY_MAX_CAP), pf(2000, 570, -4000), 4000, 100)
	require.True(r.Exit, "must exit when target SoC is met")
}

func TestDontStartWhenTargetSoCIsMet(t *testing.T) {

	require := require.New(t)

	r := callControllerAndCheckPower(require, ctrl, ss(100, BATTERY_MAX_CAP), pf(2000, 570, 0), -1, 100)
	require.True(r.Exit, "must exit when target SoC is met")
}

func TestEnsureChargePowerGTEPVPower(t *testing.T) {

	require := require.New(t)

	// charge power must be >= pvPower, otherwise the inverter will keep
	// working on DC-AC mode and the battery will not be charged from the grid

	pvPower := float64(ctrl.MaxGridImportPower()) * 0.5
	r := callControllerAndCheckPower(require, ctrl, ss(50, BATTERY_MAX_CAP), pf(pvPower, float64(ctrl.MaxGridImportPower())*0.2, 0), -1, 100)
	require.GreaterOrEqual(pvPower, float64(r.NewPowerValue))
	require.False(r.Exit)
}

func TestDontStartWhenNotEnoughPower(t *testing.T) {

	require := require.New(t)

	// charge power must be >= pvPower

	pvPower := 2000.0
	r := callControllerAndCheckPower(require, ctrl, ss(50, BATTERY_MAX_CAP), pf(pvPower, float64(ctrl.MaxGridImportPower()), 0), -1, 100)
	require.EqualValues(-1, r.NewPowerValue)
	require.False(r.Exit)
}

// Helper function to simulate multiple power computations assuming the house consumption
// is stable to let the controller achieve maximum charge power over time
func runIncrements(require *require.Assertions, ctrl port.BatteryChargeControlLogic, pvPower, housePower, batterySoC float64) int {
	var powerValue int32 = -1
	var batteryFlow float64 = 0
	count := 0
	for {
		r := callControllerAndCheckPower(require, ctrl, ss(batterySoC, BATTERY_MAX_CAP), pf(pvPower, housePower, batteryFlow), powerValue, 100)
		if r.Exit || r.NewPowerValue == powerValue || r.NewPowerValue < 0 {
			break
		}
		powerValue = r.NewPowerValue
		batteryFlow = -float64(r.NewPowerValue)
		count++
		// avoid infinite loop
		require.LessOrEqual(count, int(MAX_LOOP_INTER), "possible infinite loop avoided")
	}
	return count
}

// Helper function to simulate multiple power computations assuming the house consumption
// increases over time to force the charge power to go down
func runDecrements(require *require.Assertions, ctrl port.BatteryChargeControlLogic, pvPower float64, startingBatteryChargePower int32,
	batterySoC, housePowerIncrease float64) int {
	powerValue := startingBatteryChargePower
	batteryFlow := -float64(startingBatteryChargePower)
	count := 0
	for {
		housePower := float64(ctrl.MaxGridImportPower()) + pvPower + batteryFlow + housePowerIncrease
		r := callControllerAndCheckPower(require, ctrl, ss(batterySoC, BATTERY_MAX_CAP), pf(pvPower, housePower, batteryFlow), powerValue, 100)
		if r.Exit || r.NewPowerValue == powerValue || r.NewPowerValue < 0 {
			break
		}
		powerValue = r.NewPowerValue
		batteryFlow = -float64(r.NewPowerValue)
		count++
		// avoid infinite loop
		require.LessOrEqual(count, int(MAX_LOOP_INTER), "possible infinite loop avoided")
	}
	return count
}

// Generic function to check if the calculated power values meet the power constraints
// Does not check the rate of power increase/decrease or other behaviour that depends on controller implementation
func callControllerAndCheckPower(require *require.Assertions, ctrl port.BatteryChargeControlLogic,
	ss *sunspec_modbus.StorageState, pf PFPair, prevPowerValue int32, targetSoC uint8) domain.BatteryChargeControlTickResult {

	acpf := pf.A
	ipf := pf.B

	r := ctrl.Loop(prevPowerValue, ss, acpf, ipf, targetSoC)

	fmt.Printf("Check control tick: %d => %d (soc = %d)\n", prevPowerValue, r.NewPowerValue, targetSoC)

	require.NotEqualValues(0, r.NewPowerValue, "NewPowerValue cannot be zero")

	if ss.StateOfCharge >= float64(targetSoC) {
		// stop condition
		require.True(r.Exit, "must exist when targetSoC is met")
		require.EqualValues(-1, r.NewPowerValue)
	} else {
		pvPower := math.Max(0, ipf.PVPowerWatt)
		house := acpf.CurrentPowerFlowWatt + ipf.ACPowerWatt
		maxImport := float64(ctrl.MaxGridImportPower())
		batteryChargePower := float64(r.NewPowerValue)

		if r.NewPowerValue > 0 {
			// check new value bounds
			require.LessOrEqual(house+batteryChargePower-pvPower, maxImport, "house + batteryChargePower - pvPower <= maxImport")
			require.GreaterOrEqual(batteryChargePower, pvPower, "batteryChargePower >= PV power")
		} else if prevPowerValue > 0 {
			// temporal pause when previously was running
			require.GreaterOrEqual(house, maxImport-pvPower, "house >= maxImport - pvPower")
		}
	}
	return r
}

func genStorageState(soc float64, maxCapacity uint32) *sunspec_modbus.StorageState {
	return &sunspec_modbus.StorageState{
		StateOfCharge:   soc,
		MaxCapacityWatt: maxCapacity,
	}
}

func genACMeterPF(currentPFWatt float64) *sunspec_modbus.ACMeterPowerFlow {
	_import := 0.0
	_export := 0.0
	if currentPFWatt > 0 {
		_import += currentPFWatt
	} else {
		_export += math.Abs(currentPFWatt)
	}

	return &sunspec_modbus.ACMeterPowerFlow{
		CurrentPowerFlowWatt:   currentPFWatt,
		CurrentImportPowerWatt: _import,
		CurrentExportPowerWatt: _export,
	}
}

func genInverterPF(batteryPF, pvPower float64) *sunspec_modbus.InverterPowerFlow {
	bcharge := 0.0
	bdischarge := 0.0
	if batteryPF > 0 {
		bdischarge = batteryPF
	} else {
		bcharge = math.Abs(batteryPF)
	}
	return &sunspec_modbus.InverterPowerFlow{
		ACPowerWatt:               pvPower + batteryPF,
		PVPowerWatt:               pvPower,
		BatteryDCPowerFlowWatt:    batteryPF,
		BatteryChargePowerWatt:    bcharge,
		BatteryDischargePowerWatt: bdischarge,
	}
}

type Pair[A, B any] struct {
	A A
	B B
}

type PFPair = Pair[*sunspec_modbus.ACMeterPowerFlow, *sunspec_modbus.InverterPowerFlow]

func genDepententPF(pvPower, housePower, batteryFlow float64) PFPair {
	invPF := ipf(batteryFlow, pvPower)
	return PFPair{
		acpf(housePower - invPF.ACPowerWatt),
		invPF}
}

var ctrl = &DefaultBatteryControlLogic{
	StartPowerThreshold:     200,
	MaxRatePowerIncrease:    800,
	PowerImportSafetyMargin: 200,
	MaxImportPower:          4000,
	Logger:                  zap.Must(zap.NewDevelopment()),
}

var ss = genStorageState
var acpf = genACMeterPF
var ipf = genInverterPF
var pf = genDepententPF
