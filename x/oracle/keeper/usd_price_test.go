package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	core "github.com/classic-terra/core/v3/types"
	"github.com/classic-terra/core/v3/x/oracle/types"
)

// Tests USD price calculation using feeder-reported USD/Luna (R) and generic denom rates
func TestGetUSDPrice_UlunaAndGeneric(t *testing.T) {
	input := CreateTestInput(t)
	ctx := input.Ctx

	// Setup: denom per 1 LUNA and R=USD per 1 LUNA
	sdrPerLuna := sdk.NewDec(1_000)               // 1 LUNA = 1,000 uSDR
	usdPerLunaR := sdk.MustNewDecFromStr("0.023") // R: 1 LUNA = 0.023 USD (from core.MicroUSDDenom)

	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroSDRDenom, sdrPerLuna)
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, usdPerLunaR)

	// For uluna: USD/Luna = R directly
	gotUsdPerLuna, err := input.OracleKeeper.GetUSDPrice(ctx, core.MicroLunaDenom)
	require.NoError(t, err)
	require.True(t, gotUsdPerLuna.Equal(usdPerLunaR))

	// For uSDR: USD/SDR = R / (SDR/Luna)
	usdPerSDRExpected := usdPerLunaR.Quo(sdrPerLuna)
	usdPerSDR, err := input.OracleKeeper.GetUSDPrice(ctx, core.MicroSDRDenom)
	require.NoError(t, err)
	require.True(t, usdPerSDR.Equal(usdPerSDRExpected))
}

// Tests that uusd returns the meta USTC/USD price (U)
func TestGetUSDPrice_UUSD_ReturnsMeta(t *testing.T) {
	input := CreateTestInput(t)
	ctx := input.Ctx

	U := sdk.MustNewDecFromStr("0.021") // USD per 1 USTC
	input.OracleKeeper.SetLunaExchangeRate(ctx, types.MetaUSDDenom, U)

	// uusd should equal U, independent of R
	price, err := input.OracleKeeper.GetUSDPrice(ctx, core.MicroUSDDenom)
	require.NoError(t, err)
	require.True(t, price.Equal(U))
}

func TestGetUSDPrice_MissingMetaOrBaseRates(t *testing.T) {
	input := CreateTestInput(t)
	ctx := input.Ctx

	// No rates set: uluna should fail
	_, err := input.OracleKeeper.GetUSDPrice(ctx, core.MicroLunaDenom)
	require.Error(t, err)

	// Set uluna price (in USD) only: uluna works, uusd fails, generic fails until its E_d is set
	usdPerLuna := sdk.MustNewDecFromStr("0.02")
	input.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroUSDDenom, usdPerLuna)

	price, err := input.OracleKeeper.GetUSDPrice(ctx, core.MicroLunaDenom)
	require.NoError(t, err)
	require.True(t, price.Equal(usdPerLuna))

	// uusd without U should fail
	_, err = input.OracleKeeper.GetUSDPrice(ctx, core.MicroUSDDenom)
	require.Error(t, err)

	// Set U: uusd works now
	U := sdk.MustNewDecFromStr("0.021")
	input.OracleKeeper.SetLunaExchangeRate(ctx, types.MetaUSDDenom, U)
	price, err = input.OracleKeeper.GetUSDPrice(ctx, core.MicroUSDDenom)
	require.NoError(t, err)
	require.True(t, price.Equal(U))

	// Generic denom should fail without its rate
	_, err = input.OracleKeeper.GetUSDPrice(ctx, core.MicroSDRDenom)
	require.Error(t, err)
}

func TestQuerier_USDPriceAndUSDPrices(t *testing.T) {
	input := CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	q := NewQuerier(input.OracleKeeper)

	lunaPerKRW := sdk.NewDec(1_500)     // 1 LUNA = 1500 uKRW
	R := sdk.MustNewDecFromStr("0.01")  // USD/Luna via core.MicroUSDDenom
	U := sdk.MustNewDecFromStr("0.021") // USD per 1 USTC for uusd pricing

	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroKRWDenom, lunaPerKRW)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, R)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, types.MetaUSDDenom, U)

	// Single denom
	resp, err := q.USDPrice(ctx, &types.QueryUSDPriceRequest{Denom: core.MicroKRWDenom})
	require.NoError(t, err)
	require.Equal(t, R.Quo(lunaPerKRW), resp.UsdPrice)

	// All denoms
	resps, err := q.USDPrices(ctx, &types.QueryUSDPricesRequest{})
	require.NoError(t, err)
	require.Equal(t, R, resps.UsdPrices.AmountOf(core.MicroLunaDenom))
	require.Equal(t, R.Quo(lunaPerKRW), resps.UsdPrices.AmountOf(core.MicroKRWDenom))
}
