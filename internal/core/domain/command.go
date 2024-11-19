package domain

import "fmt"

// BatteryControlRequest

type BatteryControlRequest interface {
	ActorRequest
	BatteryControlCommand() string
}

type BatteryControlRequestMixIn struct {
	ActorRequestMixIn
}

func (r BatteryControlRequestMixIn) BatteryControlCommand() string {
	return fmt.Sprintf("%T", r)
}

// BatteryControlResponse

type BatteryControlResponse interface {
	ActorResponse
	BatteryControlResponse() string
}

type BatteryControlResponseMixIn struct {
	ActorResponse
}

func (r BatteryControlResponseMixIn) BatteryControlResponse() string {
	return fmt.Sprintf("%T", r)
}

// BatteryControl commands

type BatteryControlHoldRequest struct {
	BatteryControlRequestMixIn
	Enable bool
}

type BatteryControlHoldResponse struct {
	BatteryControlResponseMixIn
	Changed bool
}

type BatteryControlChargeRequest struct {
	BatteryControlRequestMixIn
	Enable bool
}

type BatteryControlChargeResponse struct {
	BatteryControlResponseMixIn
	Changed bool
}

type BatteryControlSetTargetSoCRequest struct {
	BatteryControlRequestMixIn
	TargetSoC uint
}

type BatteryControlSetTargetSoCResponse struct {
	BatteryControlResponseMixIn
	TargetSoC uint
}

type BatteryControlGetChargeStateRequest struct {
	BatteryControlRequestMixIn
}

type BatteryControlGetChargeStateResponse struct {
	BatteryControlResponseMixIn
	State bool
}

type BatteryControlGetHoldStateRequest struct {
	BatteryControlRequestMixIn
}

type BatteryControlGetHoldStateResponse struct {
	BatteryControlResponseMixIn
	State bool
}

// ensure interface compliance
var _ BatteryControlRequest = (*BatteryControlHoldRequest)(nil)
