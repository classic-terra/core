package simulation

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	core "github.com/classic-terra/core/v3/types"
)

// RandomGenesisBalances returns a slice of account balances. Each account has
// a balance of simState.InitialStake for sdk.DefaultBondDenom and core.MicroLunaDenom.
func RandomGenesisBalances(simState *module.SimulationState) []types.Balance {
	genesisBalances := []types.Balance{}

	for _, acc := range simState.Accounts {
		genesisBalances = append(genesisBalances, types.Balance{
			Address: acc.Address.String(),
			Coins: sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, simState.InitialStake),
				sdk.NewCoin(core.MicroLunaDenom, simState.InitialStake),
			),
		})
	}

	return genesisBalances
}

// RandomizedGenState generates a random GenesisState for bank
func RandomizedGenState(simState *module.SimulationState) {
	var sendEnabledParams []types.SendEnabled
	simState.AppParams.GetOrGenerate(
		string(types.KeySendEnabled), &sendEnabledParams, simState.Rand,
		func(r *rand.Rand) {
			sendEnabledParams = []types.SendEnabled{
				{Denom: "uluna", Enabled: true},
				{Denom: "uusd", Enabled: true},
			}
		},
	)

	var defaultSendEnabledParam bool
	simState.AppParams.GetOrGenerate(
		string(types.KeyDefaultSendEnabled), &defaultSendEnabledParam, simState.Rand,
		func(r *rand.Rand) { defaultSendEnabledParam = r.Intn(2) == 1 },
	)

	numAccs := int64(len(simState.Accounts))
	totalSupply := simState.InitialStake.Mul(math.NewInt(numAccs + simState.NumBonded))
	totalLunaSupply := simState.InitialStake.Mul(math.NewInt(numAccs))
	supply := sdk.NewCoins(
		sdk.NewCoin(sdk.DefaultBondDenom, totalSupply),
		sdk.NewCoin(core.MicroLunaDenom, totalLunaSupply),
	)

	bankGenesis := types.GenesisState{
		Params:      types.NewParams(defaultSendEnabledParam),
		Balances:    RandomGenesisBalances(simState),
		Supply:      supply,
		SendEnabled: sendEnabledParams,
	}

	paramsBytes, err := json.MarshalIndent(&bankGenesis.Params, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated bank parameters:\n%s\n", paramsBytes)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&bankGenesis)
}
