package types

import (
	"gopkg.in/yaml.v2"

	sdkmath "cosmossdk.io/math"
)

// String implements fmt.Stringer interface
func (pc PolicyConstraints) String() string {
	out, _ := yaml.Marshal(pc)
	return string(out)
}

// Clamp constrains a policy variable update within the policy constraints
func (pc PolicyConstraints) Clamp(prevRate sdkmath.LegacyDec, newRate sdkmath.LegacyDec) (clampedRate sdkmath.LegacyDec) {
	delta := newRate.Sub(prevRate)
	if newRate.GT(prevRate) {
		if delta.GT(pc.ChangeRateMax) {
			newRate = prevRate.Add(pc.ChangeRateMax)
		}
	} else {
		if delta.Abs().GT(pc.ChangeRateMax) {
			newRate = prevRate.Sub(pc.ChangeRateMax)
		}
	}

	if newRate.LT(pc.RateMin) {
		newRate = pc.RateMin
	} else if newRate.GT(pc.RateMax) {
		newRate = pc.RateMax
	}

	return newRate
}
