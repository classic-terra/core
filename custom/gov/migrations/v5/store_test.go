package v5_test

import (
	"testing"
	"time"

	v5 "github.com/classic-terra/core/v3/custom/gov/migrations/v5"
	"github.com/classic-terra/core/v3/custom/gov/types/v2custom"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func TestMigrateStore(t *testing.T) {
	// Initialize codec and store keys
	cdc := moduletestutil.MakeTestEncodingConfig(gov.AppModuleBasic{}, bank.AppModuleBasic{}).Codec

	// Create KV store keys
	govKey := storetypes.NewKVStoreKey("gov")
	transientKey := storetypes.NewTransientStoreKey("transient_test")

	// Create context and KV store
	ctx := testutil.DefaultContext(govKey, transientKey)

	store := ctx.KVStore(govKey)

	storeService := storetypes.StoreKey(govKey)

	maxDepositPeriod := (time.Hour * 24 * 2)
	votingPeriod := (time.Hour * 24 * 3)

	// Setup initial governance params in the store (old params before migration)
	oldParams := v1.Params{
		MinDeposit:                 sdk.Coins{sdk.NewCoin("uluna", sdk.NewInt(100))},
		MaxDepositPeriod:           &maxDepositPeriod,
		VotingPeriod:               &votingPeriod,
		Quorum:                     sdk.NewDecWithPrec(334, 3).String(), // 33.4%
		Threshold:                  sdk.NewDecWithPrec(500, 3).String(), // 50%
		VetoThreshold:              sdk.NewDecWithPrec(334, 3).String(), // 33.4%
		MinInitialDepositRatio:     sdk.NewDecWithPrec(100, 3).String(), // 10%
		BurnProposalDepositPrevote: true,
		BurnVoteQuorum:             true,
		BurnVoteVeto:               false,
	}

	// Marshal and set the old params into the KV store
	bz, err := cdc.Marshal(&oldParams)
	require.NoError(t, err)
	store.Set(govtypes.ParamsKey, bz)

	// Ensure the params are correctly stored before migration
	var params v1.Params
	bz = store.Get(govtypes.ParamsKey)
	require.NoError(t, cdc.Unmarshal(bz, &params))
	t.Logf("params: %v", params)
	require.Equal(t, sdk.NewDecWithPrec(500, 3).String(), params.Threshold)
	// require.Equal(t, (*time.Duration(time.Hour * 2)), params.VotingPeriod) // Expecting nil before migration

	// Run the migration function
	err = v5.MigrateStore(ctx, storeService, cdc)
	require.NoError(t, err)

	// Fetch the params after migration
	bz = store.Get(govtypes.ParamsKey)

	var newParams v2custom.Params
	require.NoError(t, cdc.Unmarshal(bz, &newParams))

	t.Logf("params2: %v", newParams)

	require.Equal(t, oldParams.MinDeposit, newParams.MinDeposit)
	require.Equal(t, oldParams.MaxDepositPeriod, newParams.MaxDepositPeriod)
	require.Equal(t, oldParams.VotingPeriod, newParams.VotingPeriod)
	require.Equal(t, oldParams.Quorum, newParams.Quorum)
	require.Equal(t, oldParams.Threshold, newParams.Threshold)
	require.Equal(t, oldParams.VetoThreshold, newParams.VetoThreshold)
	require.Equal(t, oldParams.MinInitialDepositRatio, newParams.MinInitialDepositRatio)
	require.Equal(t, oldParams.BurnProposalDepositPrevote, newParams.BurnProposalDepositPrevote)
	require.Equal(t, oldParams.BurnVoteQuorum, newParams.BurnVoteQuorum)
	require.Equal(t, oldParams.BurnVoteVeto, newParams.BurnVoteVeto)

	// Check any new fields from the `v2custom.Params`
	require.Equal(t, v2custom.DefaultParams().MinUsdDeposit, newParams.MinUsdDeposit)
}
