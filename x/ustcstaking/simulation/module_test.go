package simulation_test

import (
	"cosmossdk.io/math"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	ustcstaking "github.com/classic-terra/core/v4/x/ustcstaking"
	"github.com/classic-terra/core/v4/x/ustcstaking/keeper"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"
)

func TestNativeLifecycleSimulationRegistered(t *testing.T) {
	am := ustcstaking.NewAppModule(simulationCodec(), keeper.Keeper{})
	require.Len(t, am.WeightedOperations(module.SimulationState{AppParams: simtypes.AppParams{}}), 4)
}

func TestSimulationGenesisFundsRealAccountsAndCommunityPool(t *testing.T) {
	config := simulationCodec()
	random := rand.New(rand.NewSource(7))
	state := module.SimulationState{Cdc: config, Rand: random, Accounts: simtypes.RandomAccounts(random, 2), GenState: map[string]json.RawMessage{}, GenTimestamp: time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)}
	state.GenState[banktypes.ModuleName] = config.MustMarshalJSON(banktypes.DefaultGenesisState())
	state.GenState[distrtypes.ModuleName] = config.MustMarshalJSON(distrtypes.DefaultGenesisState())
	am := ustcstaking.NewAppModule(config, keeper.Keeper{})
	am.GenerateGenesisState(&state)
	var genesis types.GenesisState
	config.MustUnmarshalJSON(state.GenState[types.ModuleName], &genesis)
	require.NotEmpty(t, genesis.Params.LockTiers)
	require.Equal(t, math.NewInt(1_000_000), genesis.Params.RewardRatePerDay)
	require.Equal(t, state.GenTimestamp, *genesis.RewardState.LastRewardTime)
	require.NoError(t, genesis.Params.Validate())
	var bank banktypes.GenesisState
	config.MustUnmarshalJSON(state.GenState[banktypes.ModuleName], &bank)
	var distribution distrtypes.GenesisState
	config.MustUnmarshalJSON(state.GenState[distrtypes.ModuleName], &distribution)
	require.True(t, distribution.FeePool.CommunityPool.AmountOf(types.BondDenom).IsPositive())
	total := math.ZeroInt()
	for _, balance := range bank.Balances {
		total = total.Add(balance.Coins.AmountOf(types.BondDenom))
	}
	require.Equal(t, total, bank.Supply.AmountOf(types.BondDenom))
	for _, account := range state.Accounts {
		found := false
		for _, balance := range bank.Balances {
			if balance.Address == account.Address.String() {
				found = balance.Coins.AmountOf(types.BondDenom).IsPositive()
			}
		}
		require.True(t, found, "simulation account must have spendable USTC")
	}
}

func simulationCodec() codec.Codec {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	return codec.NewProtoCodec(registry)
}

func TestGovernanceProposalMessagesUseFixedAuthority(t *testing.T) {
	am := ustcstaking.NewAppModule(simulationCodec(), keeper.Keeper{})
	proposals := am.ProposalMsgs(module.SimulationState{})
	require.Len(t, proposals, 3)
	for _, proposal := range proposals {
		msg := proposal.MsgSimulatorFn()(rand.New(rand.NewSource(7)), sdk.Context{}, nil)
		switch value := msg.(type) {
		case *types.MsgFundRewards:
			require.Equal(t, keeper.Keeper{}.GetAuthority(), value.Authority)
			require.NoError(t, value.ValidateBasic())
		case *types.MsgReturnRewardDust:
			require.Equal(t, keeper.Keeper{}.GetAuthority(), value.Authority)
			require.NoError(t, value.ValidateBasic())
		case *types.MsgUpdateParams:
			require.Equal(t, keeper.Keeper{}.GetAuthority(), value.Authority)
			require.NoError(t, value.ValidateBasic())
		default:
			t.Fatalf("unexpected proposal message %T", msg)
		}
	}
}
