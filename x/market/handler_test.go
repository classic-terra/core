package market

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/market/keeper"
	"github.com/classic-terra/core/v3/x/market/types"
)

func TestMarketFilters(t *testing.T) {
	input, h := setup(t)

	// Case 2: Normal MsgSwap submission goes through
	offerCoin := sdk.NewCoin(core.MicroLunaDenom, sdkmath.NewInt(10))
	prevoteMsg := types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroSDRDenom)
	_, err := h.Swap(sdk.WrapSDKContext(input.Ctx), prevoteMsg)
	require.NoError(t, err)
}

func TestSwapMsg_FailZeroReturn(t *testing.T) {
	input, h := setup(t)

	params := input.MarketKeeper.GetParams(input.Ctx)
	params.MinStabilitySpread = sdkmath.LegacyOneDec()
	input.MarketKeeper.SetParams(input.Ctx, params)

	amt := sdkmath.NewInt(10)
	offerCoin := sdk.NewCoin(core.MicroLunaDenom, amt)
	swapMsg := types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroSDRDenom)
	_, err := h.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.Error(t, err)
}

func TestSwapMsg(t *testing.T) {
	input, h := setup(t)

	params := input.MarketKeeper.GetParams(input.Ctx)
	params.MinStabilitySpread = sdkmath.LegacyZeroDec()
	input.MarketKeeper.SetParams(input.Ctx, params)

	beforeTerraPoolDelta := input.MarketKeeper.GetTerraPoolDelta(input.Ctx)

	amt := sdkmath.NewInt(10)
	offerCoin := sdk.NewCoin(core.MicroLunaDenom, amt)
	swapMsg := types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroSDRDenom)
	_, err := h.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.NoError(t, err)

	afterTerraPoolDelta := input.MarketKeeper.GetTerraPoolDelta(input.Ctx)
	diff := beforeTerraPoolDelta.Sub(afterTerraPoolDelta)

	// calculate estimation
	basePool := input.MarketKeeper.GetParams(input.Ctx).BasePool
	price, _ := input.OracleKeeper.GetLunaExchangeRate(input.Ctx, core.MicroSDRDenom)
	cp := basePool.Mul(basePool)

	terraPool := basePool.Add(beforeTerraPoolDelta)
	lunaPool := cp.Quo(terraPool)
	estmiatedDiff := terraPool.Sub(cp.Quo(lunaPool.Add(price.MulInt(amt))))
	require.True(t, estmiatedDiff.Sub(diff.Abs()).LTE(sdkmath.LegacyNewDecWithPrec(1, 6)))

	// invalid recursive swap
	swapMsg = types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroLunaDenom)

	_, err = h.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.Error(t, err)

	// valid zero tobin tax test
	input.OracleKeeper.SetTobinTax(input.Ctx, core.MicroKRWDenom, sdkmath.LegacyZeroDec())
	input.OracleKeeper.SetTobinTax(input.Ctx, core.MicroSDRDenom, sdkmath.LegacyZeroDec())
	offerCoin = sdk.NewCoin(core.MicroSDRDenom, amt)
	swapMsg = types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroKRWDenom)
	_, err = h.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.NoError(t, err)
}

func TestSwapSendMsg(t *testing.T) {
	input, h := setup(t)

	amt := sdkmath.NewInt(10)
	offerCoin := sdk.NewCoin(core.MicroLunaDenom, amt)
	retCoin, spread, err := input.MarketKeeper.ComputeSwap(input.Ctx, offerCoin, core.MicroSDRDenom)
	require.NoError(t, err)

	expectedAmt := retCoin.Amount.Mul(sdkmath.LegacyOneDec().Sub(spread)).TruncateInt()

	swapSendMsg := types.NewMsgSwapSend(keeper.Addrs[0], keeper.Addrs[1], offerCoin, core.MicroSDRDenom)
	_, err = h.SwapSend(sdk.WrapSDKContext(input.Ctx), swapSendMsg)
	require.NoError(t, err)

	balance := input.BankKeeper.GetBalance(input.Ctx, keeper.Addrs[1], core.MicroSDRDenom)
	require.Equal(t, expectedAmt, balance.Amount)
}
