package simulation

//DONTCOVER

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/classic-terra/core/x/feeshare/types"
)

// Simulation parameter constants
const (
	enableFeeShareKey  = "enable_fee_share"
	developerSharesKey = "developer_shares"
	allowedDenomsKey   = "allowed_denoms"
)

// GenEnableFeeShare
func GenEnableFeeShare(r *rand.Rand) bool {
	rand.Seed(time.Now().UnixNano())

	return rand.Intn(2) == 1
}

// GenDeveloperShares
func GenDeveloperShares(r *rand.Rand) sdk.Dec {
	return sdk.NewDec(50000000000000).Add(sdk.NewDec(int64(r.Intn(10000000000))))
}

// GenAllowedDenoms
func GenAllowedDenoms(r *rand.Rand) []string {
	return []string(nil)
}

// RandomizedGenState generates a random GenesisState for gov
func RandomizedGenState(simState *module.SimulationState) {
	var enableFeeShare bool
	simState.AppParams.GetOrGenerate(
		simState.Cdc, enableFeeShareKey, &enableFeeShare, simState.Rand,
		func(r *rand.Rand) { enableFeeShare = GenEnableFeeShare(r) },
	)

	var developerShares sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, developerSharesKey, &developerShares, simState.Rand,
		func(r *rand.Rand) { developerShares = GenDeveloperShares(r) },
	)

	var allowedDenoms []string
	simState.AppParams.GetOrGenerate(
		simState.Cdc, allowedDenomsKey, &allowedDenoms, simState.Rand,
		func(r *rand.Rand) { allowedDenoms = GenAllowedDenoms(r) },
	)

	feeshareGenesis := types.NewGenesisState(
		types.Params{
			EnableFeeShare:  enableFeeShare,
			DeveloperShares: developerShares,
			AllowedDenoms:   allowedDenoms,
		},
		[]types.FeeShare(nil),
	)

	bz, err := json.MarshalIndent(&feeshareGenesis.Params, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Selected randomly generated feeshare parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(feeshareGenesis)
}
