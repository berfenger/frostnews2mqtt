package sunspec_modbus

import (
	"errors"
	"math"

	"github.com/simonvetter/modbus"
)

func (inv InverterIntSFModbusReader) HasStorage() (bool, error) {
	storageConn, err := inv.readRegister(inv.blocks.status+3, modbus.HOLDING_REGISTER)
	if err != nil {
		return false, err
	}
	// check storage connected
	if storageConn&0x0001 == 0 {
		return false, nil
	}
	// check valid sunspec storage
	return inv.blocks.storage > 0, nil
}

func (inv InverterIntSFModbusReader) SetStorageChargeControl(maxDischargeRatePercent float64, maxChargeRatePercent float64, controlDischarge bool, controlCharge bool, rvrtTimeSeconds int32) error {

	if inv.blocks.storage == 0 {
		return errors.New("sunspec: storage block not supported")
	}

	inoutSF, err := inv.readRegister(inv.blocks.storage+25, modbus.HOLDING_REGISTER)
	if err != nil {
		return err
	}
	control := uint16(0)
	if controlDischarge {
		control = control | 0x02
	}
	if controlCharge {
		control = control | 0x01
	}

	outWRte := int16(inv.applySFfloat64Inv(maxDischargeRatePercent, inoutSF))
	inWRte := int16(inv.applySFfloat64Inv(maxChargeRatePercent, inoutSF))

	err = inv.writeRegisters(inv.blocks.storage+12, []uint16{uint16(outWRte), uint16(inWRte)})
	if err != nil {
		return err
	}
	err = inv.writeRegister(inv.blocks.storage+5, control)
	if err != nil {
		return err
	}
	if rvrtTimeSeconds >= 0 {
		err = inv.writeRegister(inv.blocks.storage+15, uint16(rvrtTimeSeconds))
		if err != nil {
			return err
		}
	}
	return nil
}

func (inv InverterIntSFModbusReader) SetStorageControl(params StorageControlParams) error {

	rawCapacity, err := inv.getStorageCapacity()
	if err != nil {
		return err
	}
	capacity := float64(rawCapacity)

	var outWRte float64 = 100
	var inWRte float64 = 100
	controlOut := false
	controlIn := false
	rvrtTime := int32(params.RevertTimeSeconds)
	if params.MinChargePowerWatt >= 0 {
		outWRte = -(float64(params.MinChargePowerWatt) / capacity) * 100
		controlOut = true
	}
	if params.MaxChargePowerWatt >= 0 {
		inWRte = (float64(params.MaxChargePowerWatt) / capacity) * 100
		controlIn = true
	}
	if params.MinDischargePowerWatt >= 0 {
		inWRte = -(float64(params.MinDischargePowerWatt) / capacity) * 100
		controlIn = true
	}
	if params.MaxDischargePowerWatt >= 0 {
		outWRte = (float64(params.MaxDischargePowerWatt) / capacity) * 100
		controlOut = true
	}

	return inv.SetStorageChargeControl(outWRte, inWRte, controlOut, controlIn, rvrtTime)
}

func (inv InverterIntSFModbusReader) DisableStorageControl() error {
	return inv.SetStorageChargeControl(100, 100, false, false, -1)
}

func (inv InverterIntSFModbusReader) SetStorageForceChargePower(watts uint16, revertTimeSeconds int32) error {

	return inv.SetStorageControl(StorageControlParams{
		MinChargePowerWatt:    int32(watts),
		MaxChargePowerWatt:    -1,
		MinDischargePowerWatt: -1,
		MaxDischargePowerWatt: -1,
		RevertTimeSeconds:     uint32(revertTimeSeconds),
	})
}

func (inv InverterIntSFModbusReader) SetStorageForceDischargePower(watts uint16, revertTimeSeconds int32) error {

	return inv.SetStorageControl(StorageControlParams{
		MinChargePowerWatt:    -1,
		MaxChargePowerWatt:    -1,
		MinDischargePowerWatt: int32(watts),
		MaxDischargePowerWatt: -1,
		RevertTimeSeconds:     uint32(revertTimeSeconds),
	})
}

func (inv InverterIntSFModbusReader) GetStorageState() (*StorageState, error) {
	if inv.blocks.storage == 0 {
		return nil, errors.New("sunspec: storage block not supported")
	}
	regs, err := inv.readRegisters(inv.blocks.storage+2, 24, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	soc := inv.applySF(regs[6], regs[20])
	// if state == off, soc = 0
	if regs[9] == StorageChargeStatusOff {
		soc = 0
	}
	maxCap := inv.applySF(regs[0], regs[17])

	return &StorageState{
		StateOfCharge:       soc,
		MaxCapacityWatt:     uint32(math.Round(maxCap)),
		CurrentCapacityWatt: uint32(math.Round(soc / 100 * maxCap)),
		ChargeStatus:        regs[9],
		ChargeStatusStr:     StorageChargeStatusToString(regs[9]),
	}, nil
}

func (inv InverterIntSFModbusReader) getStorageCapacity() (int, error) {

	if inv.blocks.storage == 0 {
		return 0, errors.New("sunspec: storage block not supported")
	}

	wChaMax, err := inv.readRegister(inv.blocks.storage+2, modbus.HOLDING_REGISTER)
	if err != nil {
		return 0, err
	}
	wChaMaxSF, err := inv.readRegister(inv.blocks.storage+18, modbus.HOLDING_REGISTER)
	if err != nil {
		return 0, err
	}
	return int(inv.applySF(wChaMax, wChaMaxSF)), nil
}
