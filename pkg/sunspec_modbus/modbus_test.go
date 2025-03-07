package sunspec_modbus

import (
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
)

const (
	USE_MOCKED_READER = true
)

func TestCapabilities(t *testing.T) {

	reader := InverterReader()

	err := reader.Open()
	if err != nil {
		t.Error(err)
	}

	st, err := reader.HasStorage()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Has storage: %v\n", st)

	pc, err := reader.SupportsPowerControl()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Supports power control: %v\n", pc)
}

func TestPowerLimit(t *testing.T) {

	reader := InverterReader()

	err := reader.Open()
	if err != nil {
		t.Error(err)
	}

	limit, err := reader.GetPowerLimit()
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("Power limit: %+v\n", limit)
}

func TestInverterInfo(t *testing.T) {

	reader := InverterReader()

	err := reader.Open()
	if err != nil {
		t.Error(err)
	}

	st, err := reader.GetPowerFlow()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Power flow: %+v\n", st)

}

func TestForceStorage(t *testing.T) {

	reader := InverterReader()

	err := reader.Open()
	if err != nil {
		t.Error(err)
	}

	st, err := reader.HasStorage()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Has storage: %v\n", st)

	err = reader.DisableStorageControl()

	if err != nil {
		t.Error(err)
	}

}

func TestInfoStorage(t *testing.T) {

	reader := InverterReader()

	err := reader.Open()
	if err != nil {
		t.Error(err)
	}
	err = reader.Validate()
	if err != nil {
		t.Error(err)
	}

	st, err := reader.HasStorage()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Has storage: %v\n", st)

	err = reader.SetStorageControl(StorageControlParams{
		MinChargePowerWatt:    -1,
		MaxChargePowerWatt:    -1,
		MinDischargePowerWatt: -1,
		MaxDischargePowerWatt: -1,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestInfoInverter(t *testing.T) {

	reader := InverterReader()

	err := reader.Open()
	if err != nil {
		t.Error(err)
	}
	err = reader.Validate()
	if err != nil {
		t.Error(err)
	}

	nfo, err := reader.GetInfo()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Inverter Info: %+v\n", nfo)
	state, err := reader.GetState()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Inverter State: %+v\n", state)
}

func TestMeter(t *testing.T) {

	reader := ACMeterReader()

	err := reader.Open()
	if err != nil {
		t.Error(err)
		return
	}
	err = reader.Validate()
	if err != nil {
		t.Error(err)
		return
	}

	info, err := reader.GetInfo()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Meter info: %+v\n", info)

	power, err := reader.GetPowerFlow()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Meter power flow: %+v\n", power)
	fmt.Printf("Meter imported: %f\n", power.TotalEnergyImportedKWh)
	fmt.Printf("Meter exported: %f\n", power.TotalEnergyExportedKWh)

}

func TestStorageState(t *testing.T) {

	reader := InverterReader()

	err := reader.Open()
	if err != nil {
		t.Error(err)
		return
	}
	err = reader.Validate()
	if err != nil {
		t.Error(err)
		return
	}

	stState, err := reader.GetStorageState()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Storage State: %+v\n", stState)

	invState, err := reader.GetState()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Inverter State: %+v\n", invState)

	flow, err := reader.GetPowerFlow()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Inverter Power Flow: %+v\n", flow)

}

func TestBatteryControl(t *testing.T) {

	reader := InverterReader()

	err := reader.Open()
	if err != nil {
		t.Error(err)
		return
	}
	err = reader.Validate()
	if err != nil {
		t.Error(err)
		return
	}

	err = reader.SetStorageControl(StorageControlParams{
		MinChargePowerWatt: 3000,
	})
	if err != nil {
		t.Error(err)
		return
	}

	flow, err := reader.GetPowerFlow()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Inverter Power Flow: %+v\n", flow)

}

func RealInverterReader() InverterModbusReader {
	logger := zap.Must(zap.NewDevelopment())
	reader, err := CreateInverterIntSFModbusReader("-.-.-.-", 502, 0, 1*time.Second, false, logger, nil)
	if err != nil {
		panic(err)
	}
	return reader
}

func MockedInverterReader() InverterModbusReader {
	reader, err := CreateTestInverterModbusReader()
	if err != nil {
		panic(err)
	}
	return reader
}

func RealACMeterReader() ACMeterModbusReader {
	logger := zap.Must(zap.NewDevelopment())
	reader, err := CreateACMeterIntSFModbusReader("-.-.-.-", 502, 240, 1*time.Second, false, logger, nil)
	if err != nil {
		panic(err)
	}
	return reader
}

func MockedACMeterReader() ACMeterModbusReader {
	reader, err := CreateTestACMeterModbusReader()
	if err != nil {
		panic(err)
	}
	return reader
}

func InverterReader() InverterModbusReader {
	if USE_MOCKED_READER {
		return MockedInverterReader()
	} else {
		return RealInverterReader()
	}
}

func ACMeterReader() ACMeterModbusReader {
	if USE_MOCKED_READER {
		return MockedACMeterReader()
	} else {
		return RealACMeterReader()
	}
}
