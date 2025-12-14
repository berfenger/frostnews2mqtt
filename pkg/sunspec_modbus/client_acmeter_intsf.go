package sunspec_modbus

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/simonvetter/modbus"
	"go.uber.org/zap"
)

type acMeterIntSFModbusBlocks struct {
	common  uint16
	acMeter uint16
}

func (blk *acMeterIntSFModbusBlocks) AllBlocksDefined() bool {
	return blk.common > 0 && blk.acMeter > 0
}

type ACMeterIntSFModbusReader struct {
	ModbusClient
	blocks        acMeterIntSFModbusBlocks
	ignoreFronius bool
	isOpen        bool
}

func CreateACMeterIntSFModbusReader(ip string, port uint, acMeterAddress uint8, timeout time.Duration,
	ignoreFronius bool, logger *zap.Logger, instrumentation *ModbusInstrument) (ACMeterModbusReader, error) {
	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     fmt.Sprintf("tcp://%s:%d", ip, port),
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	// instrumentation
	var inst []ModbusInstrument
	logInst := traceLoggerInstrumentation(logger.With(zap.String("target", "acMeter")).With(zap.Uint8("acMeter", acMeterAddress)))
	if logInst != nil {
		inst = append(inst, *logInst)
	}
	if instrumentation != nil {
		inst = append(inst, *instrumentation)
	}

	// set ac meter address
	err = client.SetUnitId(acMeterAddress)
	if err != nil {
		return nil, err
	}
	// create reader instance
	fron := ACMeterIntSFModbusReader{
		ModbusClient: ModbusClient{
			client:     client,
			instrument: inst,
		},
		ignoreFronius: ignoreFronius,
		isOpen:        false,
	}
	return &fron, nil
}

func (reader *ACMeterIntSFModbusReader) Open() error {
	if err := reader.client.Open(); err != nil {
		return err
	}
	if err := reader.survey(); err != nil {
		return err
	}
	reader.isOpen = true
	return nil
}

func (reader *ACMeterIntSFModbusReader) IsOpen() bool {
	return reader.isOpen
}

func (reader ACMeterIntSFModbusReader) Close() error {
	return reader.client.Close()
}

func (reader ACMeterIntSFModbusReader) Validate() error {
	str, err := reader.readString(40000, 4)
	if err != nil {
		return err
	}
	if str != "SunS" {
		return errors.New("could not find a SunSpec smart meter")
	}
	str, err = reader.readString(40004, 32)
	if err != nil {
		return err
	}
	if !reader.ignoreFronius {
		if str != "Fronius" {
			return errors.New("could not find a Fronius smart meter")
		}
	}
	return nil
}

func (reader ACMeterIntSFModbusReader) GetInfo() (*ACMeterInfo, error) {
	manufacturer, err := reader.readString(reader.blocks.common+2, 32)
	if err != nil {
		return nil, err
	}
	model, err := reader.readString(reader.blocks.common+18, 32)
	if err != nil {
		return nil, err
	}
	version, err := reader.readString(reader.blocks.common+42, 16)
	if err != nil {
		return nil, err
	}
	serial, err := reader.readString(reader.blocks.common+50, 32)
	if err != nil {
		return nil, err
	}

	return &ACMeterInfo{
		Manufacturer: manufacturer,
		Model:        model,
		Version:      version,
		Serial:       serial,
	}, nil
}

func (reader ACMeterIntSFModbusReader) GetCurrentPowerFlowWatt() (float64, error) {
	regs, err := reader.readRegisters(reader.blocks.acMeter+18, 5, modbus.HOLDING_REGISTER)
	if err != nil {
		return 0, err
	}
	totalRealPower := regs[0]   // reader.blocks.acMeter+18
	totalRealPowerSF := regs[4] // reader.blocks.acMeter+22
	return reader.applySFint16(int16(totalRealPower), totalRealPowerSF), nil
}

func (reader ACMeterIntSFModbusReader) getGridFrequency() (float64, error) {
	freq, err := reader.readRegisters(reader.blocks.acMeter+16, 2, modbus.HOLDING_REGISTER)
	if err != nil {
		return 0, err
	}
	return reader.applySF(freq[0], freq[1]), nil
}

func (reader ACMeterIntSFModbusReader) getEnergyCounterValues() (totalEnergyExported float64, totalEnergyImported float64, err error) {
	regs, err := reader.client.ReadRawBytes(reader.blocks.acMeter+38, 34, modbus.HOLDING_REGISTER)
	if err != nil {
		return 0, 0, err
	}
	rawTotalEnergyExported := reader.bytesToUint32(regs[0:4])   // reader.blocks.acMeter+38
	rawTotalEnergyImported := reader.bytesToUint32(regs[16:20]) // reader.blocks.acMeter+46
	totWh_SF := reader.bytesToUint16(regs[32:34])               // reader.blocks.acMeter+54
	totalEnergyExported = reader.applySFuint32(rawTotalEnergyExported, totWh_SF) / 1000
	totalEnergyImported = reader.applySFuint32(rawTotalEnergyImported, totWh_SF) / 1000
	err = nil
	return
}

func (reader ACMeterIntSFModbusReader) getPhaseAVoltage() (float64, error) {
	regs, err := reader.readRegisters(reader.blocks.acMeter+8, 8, modbus.HOLDING_REGISTER)
	if err != nil {
		return 0, err
	}
	phaseAVoltage := regs[0]    // reader.blocks.acMeter+8
	phaseAVoltage_SF := regs[7] // reader.blocks.acMeter+15
	return reader.applySF(phaseAVoltage, phaseAVoltage_SF), nil
}

func (reader ACMeterIntSFModbusReader) GetPowerFlow() (*ACMeterPowerFlow, error) {
	totalRealPower, err := reader.GetCurrentPowerFlowWatt()
	if err != nil {
		return nil, err
	}
	totalEnergyExported, totalEnergyImported, err := reader.getEnergyCounterValues()
	if err != nil {
		return nil, err
	}
	freq, err := reader.getGridFrequency()
	if err != nil {
		return nil, err
	}
	phaseAVoltage, err := reader.getPhaseAVoltage()
	if err != nil {
		return nil, err
	}
	var importPower float64 = 0
	var exportPower float64 = 0
	if totalRealPower < 0 {
		exportPower = math.Abs(totalRealPower)
	} else {
		importPower = totalRealPower
	}

	return &ACMeterPowerFlow{
		CurrentPowerFlowWatt:   totalRealPower,
		CurrentImportPowerWatt: importPower,
		CurrentExportPowerWatt: exportPower,
		TotalEnergyExportedKWh: totalEnergyExported,
		TotalEnergyImportedKWh: totalEnergyImported,
		Frequency:              freq,
		PhaseAVoltage:          phaseAVoltage,
	}, nil
}

func (inv *ACMeterIntSFModbusReader) survey() error {

	// check SunSpec
	str, err := inv.readString(40000, 4)
	if err != nil {
		return err
	}
	if str != "SunS" {
		return errors.New("could not find a SunSpec smart meter")
	}

	// survey blocks
	blocks := acMeterIntSFModbusBlocks{}
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
		switch block.id {
		case 1:
			blocks.common = block.baseAddr
		case 201, 202, 203, 204:
			blocks.acMeter = block.baseAddr
		}
		baseAddr = baseAddr + block.length + 2
		// ensure the loop has an ending
		if blocks.AllBlocksDefined() || n > 10 {
			break
		}
		n++
	}
	if blocks.common > 0 && blocks.acMeter > 0 {
		inv.blocks = blocks
		return nil
	}
	return errors.New("could not find all required sunspec blocks (common, ac_meter)")
}
