package keeper

import (
	"testing"

	core "github.com/classic-terra/core/v4/types"
	"github.com/classic-terra/core/v4/x/market/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSwap_InvalidPair(t *testing.T) {
	input := CreateTestInput(t)
	server := NewMsgServerImpl(input.MarketKeeper)

	// uluna -> ukrw is not allowed by guard
	msg := types.NewMsgSwap(Addrs[0], sdk.NewInt64Coin(core.MicroLunaDenom, 1_000_000), core.MicroKRWDenom)
	_, err := server.Swap(sdk.WrapSDKContext(input.Ctx), msg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidSwapPair)
}

func TestSwap_OracleGuard_NoUSDMeta(t *testing.T) {
	input := CreateTestInput(t)
	server := NewMsgServerImpl(input.MarketKeeper)

	// Allowed pair but missing oracle meta USD rate -> guard should fail
	msg := types.NewMsgSwap(Addrs[0], sdk.NewInt64Coin(core.MicroLunaDenom, 1_000_000), core.MicroUSDDenom)
	_, err := server.Swap(sdk.WrapSDKContext(input.Ctx), msg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrNoEffectivePrice)
}
