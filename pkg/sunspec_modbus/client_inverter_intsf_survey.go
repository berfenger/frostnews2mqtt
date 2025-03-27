package sunspec_modbus

import (
	"errors"

	"github.com/simonvetter/modbus"
)

const (
	SUNSPEC_WK_COMMON        = 1
	SUNSPEC_WK_INVERTERS_MIN = 101
	SUNSPEC_WK_INVERTERS_MAX = 103
	SUNSPEC_WK_STATUS        = 122
	SUNSPEC_WK_CONTROLS      = 123
	SUNSPEC_WK_STORAGE       = 124
	SUNSPEC_WK_MPPT          = 160
)

// inverter
type inverterIntSFModbusBlocks struct {
	common   uint16
	inverter uint16
	controls uint16
	status   uint16
	mppt     uint16
	storage  uint16
}

func (blk *inverterIntSFModbusBlocks) AllBlocksDefined() bool {
	return blk.common > 0 && blk.inverter > 0 && blk.controls > 0 &&
		blk.status > 0 && blk.mppt > 0 && blk.storage > 0
}

func (inv *InverterIntSFModbusReader) survey() error {

	// check SunSpec
	str, err := inv.readString(40000, 4)
	if err != nil {
		return err
	}
	if str != "SunS" {
		return errors.New("could not find a SunSpec inverter")
	}

	// survey blocks
	blocks := inverterIntSFModbusBlocks{}
	var baseAddr uint16 = 40002
	n := 0
	for {
		block, err := surveyModbusBlock(inv.client, baseAddr)
		if err != nil {
			return err
		}
		if block.isEndBlock() {
			break
		}
		// identify block
		if block.id >= SUNSPEC_WK_INVERTERS_MIN && block.id <= SUNSPEC_WK_INVERTERS_MAX {
			blocks.inverter = block.baseAddr
		} else {
			switch block.id {
			case SUNSPEC_WK_COMMON:
				blocks.common = block.baseAddr
			case SUNSPEC_WK_STATUS:
				blocks.status = block.baseAddr
			case SUNSPEC_WK_CONTROLS:
				blocks.controls = block.baseAddr
			case SUNSPEC_WK_STORAGE:
				blocks.storage = block.baseAddr
			case SUNSPEC_WK_MPPT:
				blocks.mppt = block.baseAddr
			}
		}
		baseAddr = baseAddr + block.length + 2
		// ensure the loop has an ending
		if blocks.AllBlocksDefined() || n > 20 {
			break
		}
		n++
	}
	if blocks.common > 0 && blocks.inverter > 0 &&
		blocks.status > 0 && blocks.mppt > 0 {
		inv.blocks = blocks
		return nil
	}
	return errors.New("could not find all required sunspec blocks (common, inverter, status, mppt)")
}

// common

type modbusBlock struct {
	id       uint16
	baseAddr uint16
	length   uint16
}

func (block *modbusBlock) isEndBlock() bool {
	return block.id == 0xFFFF
}

func surveyModbusBlock(client *modbus.ModbusClient, baseAddr uint16) (*modbusBlock, error) {
	wellKnownValue, err := client.ReadRegister(baseAddr, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	length, err := client.ReadRegister(baseAddr+1, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	return &modbusBlock{
		id:       wellKnownValue,
		length:   length,
		baseAddr: baseAddr,
	}, nil
}
