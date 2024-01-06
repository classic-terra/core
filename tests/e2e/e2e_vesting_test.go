package e2e

import (
	"encoding/json"
	"math/rand"
	// "path/filepath"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	delayedVestingKey    = "delayed_vesting"
	continuousVestingKey = "continuous_vesting"
	lockedVestingKey     = "locker_vesting"
	periodicVestingKey   = "periodic_vesting"

	vestingPeriodFile = "test_period.json"
	vestingTxDelay    = 5
)

type (
	vestingPeriod struct {
		StartTime int64    `json:"start_time"`
		Periods   []period `json:"periods"`
	}
	period struct {
		Coins  string `json:"coins"`
		Length int64  `json:"length_seconds"`
	}
)

var (
	genesisVestingKeys      = []string{continuousVestingKey, delayedVestingKey, lockedVestingKey, periodicVestingKey}
	vestingAmountVested     = sdk.NewCoin(ulunaDenom, sdk.NewInt(99900000000))
	vestingAmount           = sdk.NewCoin(ulunaDenom, sdk.NewInt(350000))
	vestingBalance          = sdk.NewCoins(vestingAmountVested).Add(vestingAmount)
	vestingDelegationAmount = sdk.NewCoin(ulunaDenom, sdk.NewInt(500000000))
	vestingDelegationFees   = sdk.NewCoin(ulunaDenom, sdk.NewInt(1))
)

// generateVestingPeriod generate the vesting period file
func generateVestingPeriod() ([]byte, error) {
	p := vestingPeriod{
		StartTime: time.Now().Add(time.Duration(rand.Intn(20)+95) * time.Second).Unix(),
		Periods: []period{
			{
				Coins:  "850000000" + ulunaDenom,
				Length: 35,
			},
			{
				Coins:  "2000000000" + ulunaDenom,
				Length: 35,
			},
		},
	}
	return json.Marshal(p)
}
