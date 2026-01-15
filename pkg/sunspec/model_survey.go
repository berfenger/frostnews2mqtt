package sunspec

import (
	"context"

	"github.com/berfenger/frostnews2mqtt/pkg/modbus"
	"github.com/berfenger/frostnews2mqtt/pkg/util/logutil"
	"go.uber.org/zap"
)

func SurveyDeviceAuto(ctx context.Context, client modbus.ModbusReaderWriter) (*SunspecDeviceModels, error) {
	logutil.Debug(ctx, "Starting auto SunSpec device survey")

	probes := buildClientProbes(ctx, client)

	for _, probe := range probes {
		dev, err := probe()
		if err == nil {
			return dev, nil
		}
	}

	logutil.Error(ctx, "SunSpec device not found after all address tests")
	return nil, ErrDeviceNotFound
}

func SurveyDeviceAtAddress(ctx context.Context, client modbus.ModbusReaderWriter, sunsBaseAddr uint16) (*SunspecDeviceModels, error) {
	logutil.Debug(ctx, "Surveying SunSpec device at address", zap.Uint16("base_addr", sunsBaseAddr))

	// check SunSpec
	str, err := client.ReadString(sunsBaseAddr, 4)
	if err != nil {
		logutil.Warn(ctx, "Failed to read SunSpec identifier", zap.Uint16("base_addr", sunsBaseAddr), zap.Error(err))
		return nil, err
	}
	if str != SUNS_DEVICE_WELL_KNOWN_ID {
		logutil.Debug(ctx, "Invalid SunSpec identifier",
			zap.String("expected", SUNS_DEVICE_WELL_KNOWN_ID),
			zap.String("found", str))
		return nil, ErrDeviceNotFound
	}

	logutil.Debug(ctx, "Valid SunSpec device found", zap.Uint16("base_addr", sunsBaseAddr))

	// survey models
	models := []ModelHeader{}
	baseAddr := sunsBaseAddr + 2
	n := 0
	for {
		block, err := surveyModelBlock(client, baseAddr)
		if err != nil {
			logutil.Warn(ctx, "Failed to survey model block", zap.Error(err))
			return nil, err
		}
		logutil.Debug(ctx, "Found SunSpec model",
			zap.Uint16("model_id", block.id),
			zap.Uint16("base_addr", block.baseAddr),
			zap.Uint16("length", block.length))
		if block.isEndBlock() {
			logutil.Debug(ctx, "Reached end block")
			break
		}
		// identify block
		if block.id >= SUNS_MODEL_ID_ACMETER_MIN && block.id <= SUNS_MODEL_ID_ACMETER_MAX {
			models = append(models, *block)
		} else if block.id >= SUNS_MODEL_ID_INVERTER_MIN && block.id <= SUNS_MODEL_ID_INVERTER_MAX {
			models = append(models, *block)
		} else {
			switch block.id {
			case SUNS_MODEL_ID_COMMON:
				models = append(models, *block)
			case SUNS_MODEL_ID_NAMEPLATE:
				models = append(models, *block)
			case SUNS_MODEL_ID_STATUS:
				models = append(models, *block)
			case SUNS_MODEL_ID_CONTROLS:
				models = append(models, *block)
			case SUNS_MODEL_ID_STORAGE:
				models = append(models, *block)
			case SUNS_MODEL_ID_MPPT:
				models = append(models, *block)
			}
		}
		baseAddr = baseAddr + block.length + 2
		// ensure the loop has an ending
		if n > 20 {
			break
		}
		n++
	}
	return NewDeviceModels(models), nil
}

func buildClientProbes(ctx context.Context, client modbus.ModbusReaderWriter) []func() (*SunspecDeviceModels, error) {
	probes := []func() (*SunspecDeviceModels, error){}
	for _, baseAddr := range SUNS_BASE_ADDRESSES {
		probes = append(probes, func() (*SunspecDeviceModels, error) {
			return SurveyDeviceAtAddress(ctx, client, baseAddr)
		})
	}
	return probes
}

func surveyModelBlock(client modbus.ModbusReaderWriter, baseAddr uint16) (*ModelHeader, error) {
	regs, err := client.ReadRegisters(baseAddr, 2, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, err
	}
	return &ModelHeader{
		id:       regs[0], // well-known model ID
		length:   regs[1], // model length in registers
		baseAddr: baseAddr + 2,
	}, nil
}
