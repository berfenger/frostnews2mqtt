package modbus

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

type ModbusInstrumentation struct {
	RecordTime func(fnName string, readTime time.Duration)
}

func RecordModbusTimer(name string, addr uint16, quantity uint16, unitId uint8, instrument []ModbusInstrumentation) func() {
	return RecordTimer(fmt.Sprintf("%s addr=%d quantity=%d unitId=%d", name, addr, quantity, unitId), instrument)
}

func RecordTimer(name string, instrument []ModbusInstrumentation) func() {
	if instrument == nil {
		return func() {}
	}

	start := time.Now()
	return func() {
		duration := time.Since(start)
		for i := range instrument {
			instrument[i].RecordTime(name, duration)
		}
	}
}

func CreateModbusInstrumentation(target string, unitId uint8, logger *zap.Logger, instrumentation []ModbusInstrumentation) []ModbusInstrumentation {
	var insts []ModbusInstrumentation
	logInst := traceLoggerInstrumentation(logger.With(zap.String("target", target)).With(zap.Uint8("unitId", unitId)))
	if logInst != nil {
		insts = append(insts, *logInst)
	}
	if len(instrumentation) > 0 {
		insts = append(insts, instrumentation...)
	}
	return insts
}

func traceLoggerInstrumentation(logger *zap.Logger) *ModbusInstrumentation {
	return &ModbusInstrumentation{
		RecordTime: func(fnName string, readTime time.Duration) {
			logger.Sugar().Debugf("modbus [%s]: %d millis", fnName, readTime.Milliseconds())
		},
	}
}
