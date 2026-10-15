package simulation

import (
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// RandomizedGenesis supplies simulation fixtures only, without changing chain
// defaults. All USTC is accounted for in genesis balances and supply; reward
// funding still has to use governance and the native community-pool API.
func RandomizedGenesis(state *module.SimulationState) {
	genesis := types.DefaultGenesisState()
	duration := time.Duration(1+state.Rand.Intn(60)) * time.Second
	genesis.Params.LockTiers = []types.LockTier{{Id: 1, Duration: &duration, Multiplier: math.LegacyNewDec(int64(1 + state.Rand.Intn(3)))}}
	genesis.Params.RewardRatePerDay = math.NewInt(1_000_000) // 1 USTC/day, simulation only
	genesis.RewardState.LastRewardTime = timePtr(state.GenTimestamp)
	state.GenState[types.ModuleName] = state.Cdc.MustMarshalJSON(genesis)

	var bank banktypes.GenesisState
	state.Cdc.MustUnmarshalJSON(state.GenState[banktypes.ModuleName], &bank)
	amount := sdk.NewInt64Coin(types.BondDenom, 1_000_000_000)
	for _, account := range state.Accounts {
		addBalance(&bank, account.Address.String(), amount)
	}
	// Seed the existing distribution community pool, never the native reward pool.
	reserve := sdk.NewInt64Coin(types.BondDenom, 1_000_000_000_000)
	addBalance(&bank, authtypes.NewModuleAddress(distrtypes.ModuleName).String(), reserve)
	state.GenState[banktypes.ModuleName] = state.Cdc.MustMarshalJSON(&bank)
	var distribution distrtypes.GenesisState
	state.Cdc.MustUnmarshalJSON(state.GenState[distrtypes.ModuleName], &distribution)
	distribution.FeePool.CommunityPool = distribution.FeePool.CommunityPool.Add(sdk.NewDecCoinFromCoin(reserve))
	state.GenState[distrtypes.ModuleName] = state.Cdc.MustMarshalJSON(&distribution)
}

func timePtr(value time.Time) *time.Time { return &value }

func addBalance(bank *banktypes.GenesisState, address string, amount sdk.Coin) {
	for i := range bank.Balances {
		if bank.Balances[i].Address == address {
			bank.Balances[i].Coins = bank.Balances[i].Coins.Add(amount)
			bank.Supply = bank.Supply.Add(amount)
			return
		}
	}
	bank.Balances = append(bank.Balances, banktypes.Balance{Address: address, Coins: sdk.NewCoins(amount)})
	bank.Supply = bank.Supply.Add(amount)
}
