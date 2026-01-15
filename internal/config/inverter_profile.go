package config

import (
	"fmt"
	"strings"
)

type InverterProfile string

const (
	InverterProfileFronius InverterProfile = "fronius"
	InverterProfileSunspec InverterProfile = "sunspec"
)

func (p InverterProfile) String() string {
	return string(p)
}

func (p InverterProfile) IsValid() bool {
	switch p {
	case InverterProfileFronius, InverterProfileSunspec:
		return true
	}
	return false
}

func ParseInverterProfile(s string) (InverterProfile, error) {
	profile := InverterProfile(strings.ToLower(s))
	if !profile.IsValid() {
		return "", fmt.Errorf("invalid inverter profile: %s (valid values: %s, %s)", s, InverterProfileFronius, InverterProfileSunspec)
	}
	return profile, nil
}
