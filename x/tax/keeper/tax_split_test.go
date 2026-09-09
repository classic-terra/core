package keeper_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	apphelpers "github.com/classic-terra/core/v4/app/testing"
	core "github.com/classic-terra/core/v4/types"
	markettypes "github.com/classic-terra/core/v4/x/market/types"
	oracletypes "github.com/classic-terra/core/v4/x/oracle/types"
	treasurytypes "github.com/classic-terra/core/v4/x/treasury/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"
)

func TestProcessTaxSplits_RedirectToMarketAccumulator(t *testing.T) {
	// Setup app and context
	chainID := "tax-redirect-test"
	app := apphelpers.SetupApp(t, chainID)
	ctx := app.NewUncachedContext(false, tmproto.Header{Height: 1, ChainID: chainID, Time: time.Now().UTC()})

	// Configure distribution params: community tax = 0 to simplify
	distrParams := distrtypes.DefaultParams()
	distrParams.CommunityTax = sdkmath.LegacyZeroDec()
	app.DistrKeeper.Params.Set(ctx, distrParams)

	// Configure treasury params: BurnSplit=1.0, OracleSplit=0.5, TaxRedirectRate=1.0
	tparams := treasurytypes.DefaultParams()
	tparams.BurnTaxSplit = sdkmath.LegacyOneDec()                // 100% goes to distribution (no remainder to final burn)
	tparams.OracleSplit = sdkmath.LegacyNewDecWithPrec(5, 1)     // 0.5 to oracle
	tparams.TaxRedirectRate = sdkmath.LegacyNewDecWithPrec(5, 1) // 50% of post-oracle base to market accumulator
	app.TreasuryKeeper.SetParams(ctx, tparams)

	// Prepare taxes to split
	taxAmt := sdkmath.NewInt(1_000_000)
	taxes := sdk.NewCoins(sdk.NewCoin(core.MicroUSDDenom, taxAmt))

	// Fund FeeCollector: mint to treasury (has Minter) and transfer to FeeCollector
	require.NoError(t, app.BankKeeper.MintCoins(ctx, treasurytypes.ModuleName, taxes))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, treasurytypes.ModuleName, authtypes.FeeCollectorName, taxes))

	// Execute split
	require.NoError(t, app.TaxKeeper.ProcessTaxSplits(ctx, taxes))

	// Expected splits with new semantics (redirect to market accumulator first):
	// community=0, BurnTaxSplit=1.0, OracleSplit=0.5, TaxRedirectRate=0.5
	// Let T be full taxes; redirect M = 0.5*T to market accumulator; remaining T1 = 0.5*T.
	// DistributionDelta = BurnTaxSplit * T1 = 1.0 * T1 = T1; CommunityTax=0; Oracle gets 0.5*T1 = 0.25*T.
	// Remaining 'taxes' at end is zero (we subtracted DistributionDelta fully), so burn = 0.
	expectedMarket := sdkmath.LegacyNewDecFromInt(taxAmt).Mul(sdkmath.LegacyNewDecWithPrec(5, 1)).TruncateInt()  // 50% of T
	expectedOracle := sdkmath.LegacyNewDecFromInt(taxAmt).Mul(sdkmath.LegacyNewDecWithPrec(25, 2)).TruncateInt() // 25% of T

	// Module addresses
	oracleAddr := app.AccountKeeper.GetModuleAddress(oracletypes.ModuleName)
	marketAccumAddr := app.AccountKeeper.GetModuleAddress(markettypes.AccumulatorModuleName)
	burnAddr := app.AccountKeeper.GetModuleAddress(treasurytypes.BurnModuleName)

	// Balances
	oracleBal := app.BankKeeper.GetBalance(ctx, oracleAddr, core.MicroUSDDenom).Amount
	marketBal := app.BankKeeper.GetBalance(ctx, marketAccumAddr, core.MicroUSDDenom).Amount
	burnBal := app.BankKeeper.GetBalance(ctx, burnAddr, core.MicroUSDDenom).Amount

	require.Equal(t, expectedOracle, oracleBal, "oracle split mismatch")
	require.Equal(t, expectedMarket, marketBal, "market redirect mismatch")
	require.True(t, burnBal.IsZero(), "burn should be zero with burnSplit=1.0 and redirect=1.0")
}
