package sunspec_modbus

import (
	"errors"
	"fmt"
	"time"

	"github.com/simonvetter/modbus"
	log "github.com/sirupsen/logrus"
)

type InverterIntSFModbusReader struct {
	ModbusClient

	logger        *log.Logger
	blocks        inverterIntSFModbusBlocks
	ignoreFronius bool
}

func (inv *InverterIntSFModbusReader) Open() error {
	if err := inv.client.Open(); err != nil {
		return err
	}
	if err := inv.survey(); err != nil {
		return err
	}
	return nil
}

func (inv InverterIntSFModbusReader) Close() error {
	return inv.client.Close()
}

func (inv InverterIntSFModbusReader) Validate() error {
	// check manufacturer
	if !inv.ignoreFronius {
		str, err := inv.readString(inv.blocks.common+2, 32)
		if err != nil {
			return err
		}
		if str != "Fronius" {
			return errors.New("could not find a Fronius inverter")
		}
	}
	return nil
}

func (inv InverterIntSFModbusReader) GetInfo() (*InverterInfo, error) {
	manufacturer, err := inv.readString(inv.blocks.common+2, 32)
	if err != nil {
		return nil, err
	}
	model, err := inv.readString(inv.blocks.common+18, 32)
	if err != nil {
		return nil, err
	}
	version, err := inv.readString(inv.blocks.common+42, 16)
	if err != nil {
		return nil, err
	}
	serial, err := inv.readString(inv.blocks.common+50, 32)
	if err != nil {
		return nil, err
	}

	pow, err := inv.readRegister(inv.blocks.inverter+82, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	powSF, err := inv.readRegister(inv.blocks.inverter+102, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	hasStorage, err := inv.HasStorage()
	if err != nil {
		return nil, err
	}

	return &InverterInfo{
		Manufacturer:      manufacturer,
		Model:             model,
		Version:           version,
		Serial:            serial,
		MaxRatedPowerWatt: uint32(inv.applySF(pow, powSF)),
		HasStorage:        hasStorage,
	}, nil
}
func (inv InverterIntSFModbusReader) GetState() (*InverterState, error) { // TODO
	temp, err := inv.readRegister(inv.blocks.inverter+33, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	tempSF, err := inv.readRegister(inv.blocks.inverter+37, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	temperature := inv.applySF(temp, tempSF)
	state, err := inv.readRegister(inv.blocks.inverter+38, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}

	return &InverterState{
		CabinetTemperature: temperature,
		OperatingState:     state,
		OperatingStateStr:  InverterStatusToString(state),
	}, nil
}

func (inv InverterIntSFModbusReader) SetPowerLimit(powerLimit InverterPowerLimit) error {
	if inv.blocks.controls == 0 {
		return errors.New("sunspec: controls block not supported")
	}
	// write 0 to WMaxLim_Ena. A new value won't be accepted without this step
	err := inv.writeRegister(inv.blocks.controls+9, uint16(0))
	if err != nil {
		return err
	}
	if powerLimit.Enabled {
		// get scale factor to write percent
		sf, err := inv.readRegister(inv.blocks.controls+23, modbus.HOLDING_REGISTER)
		if err != nil {
			return err
		}
		c := uint16(1)
		// build and write data array [WMaxLimPct, WMaxLimPct_WinTms, WMaxLimPct_RvrtTms, WMaxLimPct_RmpTms, WMaxLim_Ena]
		data := []uint16{uint16(inv.applySFInv(uint16(powerLimit.Percent), sf)), 0, uint16(powerLimit.RevertTimeSeconds), 0, c}
		return inv.writeRegisters(inv.blocks.controls+5, data)
	} else {
		return err
	}
}
func (inv InverterIntSFModbusReader) GetPowerLimit() (*InverterPowerLimit, error) {
	if inv.blocks.controls == 0 {
		return nil, errors.New("sunspec: controls block not supported")
	}
	regs, err := inv.readRegisters(inv.blocks.controls+5, 5, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	sf, err := inv.readRegister(inv.blocks.controls+23, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}

	ena := regs[4] == 1
	pct := inv.applySF(regs[0], sf)
	rvtTimeout := regs[2]

	return &InverterPowerLimit{
		Enabled:           ena,
		Percent:           pct,
		RevertTimeSeconds: uint32(rvtTimeout),
	}, nil
}

func (inv InverterIntSFModbusReader) GetPowerFlow() (*InverterPowerFlow, error) {
	// ac power
	acpower, err := inv.readRegisters(inv.blocks.inverter+14, 2, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	// dc power sf
	dcPowerSF, err := inv.readRegister(inv.blocks.mppt+4, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	// pv dc power
	var dcpower float64 = 0
	var chargeDCPower float64 = 0
	var dischargeDCPower float64 = 0
	nMods, err := inv.readRegister(inv.blocks.mppt+8, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	if nMods > 0 { // 1 MPPT + Battery or just 1 MPPT
		mpptPower, err := inv.readMPPTPower(0)
		if err != nil {
			return nil, err
		}
		dcpower += inv.applySF(mpptPower, dcPowerSF)
	}
	if nMods == 2 || nMods == 4 { // 2 MPPT + Battery or just 2 MPPT
		mpptPower, err := inv.readMPPTPower(1)
		if err != nil {
			return nil, err
		}
		dcpower += inv.applySF(mpptPower, dcPowerSF)
	}
	if nMods == 3 || nMods == 4 { // 1 or 2 MPPT + Battery
		chargeDCPowerRaw, err := inv.readMPPTPower(uint8(nMods - 2))
		if err != nil {
			return nil, err
		}
		chargeDCPower = inv.applySF(chargeDCPowerRaw, dcPowerSF)
		dischargeDCPowerRaw, err := inv.readMPPTPower(uint8(nMods - 1))
		if err != nil {
			return nil, err
		}
		dischargeDCPower = inv.applySF(dischargeDCPowerRaw, dcPowerSF)
	}

	return &InverterPowerFlow{
		ACPowerWatt:               inv.applySFint16(int16(acpower[0]), acpower[1]),
		PVPowerWatt:               dcpower,
		BatteryChargePowerWatt:    chargeDCPower,
		BatteryDischargePowerWatt: dischargeDCPower,
		BatteryDCPowerFlowWatt:    dischargeDCPower - chargeDCPower,
	}, nil
}

func (inv InverterIntSFModbusReader) readMPPTPower(index uint8) (uint16, error) {
	baseAddr := uint16(inv.blocks.mppt + 10 + 20*uint16(index))
	// dc power
	dcpower, err := inv.readRegister(baseAddr+11, modbus.HOLDING_REGISTER)
	if err != nil {
		return 0, err
	}
	if int16(dcpower) == -1 {
		dcpower = 0
	}
	return dcpower, nil
}

func (inv InverterIntSFModbusReader) SupportsPowerControl() (bool, error) {
	return inv.blocks.controls > 0, nil
}

func traceLoggerInstrumentation(logger *log.Entry) *ModbusInstrument {
	return &ModbusInstrument{
		RecordTime: func(fnName string, readTime time.Duration) {
			logger.Tracef("modbus [%s]: %d millis", fnName, readTime.Milliseconds())
		},
	}
}

func CreateInverterIntSFModbusReader(ip string, port uint, inverterAddress uint8, timeout time.Duration,
	ignoreFronius bool, logger *log.Logger, instrumentation *ModbusInstrument) (InverterModbusReader, error) {
	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     fmt.Sprintf("tcp://%s:%d", ip, port),
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}

	// instrumentation
	var inst []ModbusInstrument
	logInst := traceLoggerInstrumentation(logger.WithField("target", "inverter").WithField("inverter", inverterAddress))
	if logInst != nil {
		inst = append(inst, *logInst)
	}
	if instrumentation != nil {
		inst = append(inst, *instrumentation)
	}

	// set inverter address
	if inverterAddress > 0 {
		err = client.SetUnitId(inverterAddress)
		if err != nil {
			return nil, err
		}
	}

	// create reader instance
	fron := InverterIntSFModbusReader{
		ModbusClient: ModbusClient{
			client:     client,
			instrument: inst,
		},
		logger:        logger,
		ignoreFronius: ignoreFronius,
	}
	return &fron, nil
}
