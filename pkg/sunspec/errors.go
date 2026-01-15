package sunspec

import "errors"

var (
	ErrDeviceNotFound   = errors.New("could not find a SunSpec device")
	ErrInverterNotFound = errors.New("could not find all required sunspec inverter blocks")
	ErrAcMeterNotFound  = errors.New("could not find all required sunspec ac meter blocks")
)
