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
	return nil
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
	totalRealPower, err := reader.readRegister(reader.blocks.acMeter+18, modbus.HOLDING_REGISTER)
	if err != nil {
		return 0, err
	}
	totalRealPowerSF, err := reader.readRegister(reader.blocks.acMeter+22, modbus.HOLDING_REGISTER)
	if err != nil {
		return 0, err
	}
	return reader.applySFint16(int16(totalRealPower), totalRealPowerSF), nil
}

func (reader ACMeterIntSFModbusReader) GetPowerFlow() (*ACMeterPowerFlow, error) {
	totalRealPower, err := reader.GetCurrentPowerFlowWatt()
	if err != nil {
		return nil, err
	}
	totalEnergyExported, err := reader.readUint32(reader.blocks.acMeter+38, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	totalEnergyImported, err := reader.readUint32(reader.blocks.acMeter+46, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	totWh_SF, err := reader.readRegister(reader.blocks.acMeter+54, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	freq, err := reader.readRegisters(reader.blocks.acMeter+16, 2, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	phaseAVoltage, err := reader.readRegister(reader.blocks.acMeter+8, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	phaseAVoltage_SF, err := reader.readRegister(reader.blocks.acMeter+15, modbus.HOLDING_REGISTER)
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
		TotalEnergyExportedKWh: reader.applySFuint32(totalEnergyExported, totWh_SF) / 1000,
		TotalEnergyImportedKWh: reader.applySFuint32(totalEnergyImported, totWh_SF) / 1000,
		Frequency:              reader.applySF(freq[0], freq[1]),
		PhaseAVoltage:          reader.applySF(phaseAVoltage, phaseAVoltage_SF),
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
