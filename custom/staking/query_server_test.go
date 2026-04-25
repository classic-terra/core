package staking_test

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	apptesting "github.com/classic-terra/core/v4/app/testing"
	customstaking "github.com/classic-terra/core/v4/custom/staking"
	"github.com/classic-terra/core/v4/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/testutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"
)

type ValidatorDelegationsSuite struct {
	apptesting.KeeperTestHelper
}

func TestValidatorDelegationsSuite(t *testing.T) {
	suite.Run(t, new(ValidatorDelegationsSuite))
}

// seedValidatorWithDelegations creates `numVals` validators (so no single one
// exceeds the 20% voting-power cap enforced by the custom staking hook),
// then has `numDels` distinct delegators each delegate 1_000_000 uluna to the
// FIRST validator. Returns that validator's address.
func (s *ValidatorDelegationsSuite) seedValidatorWithDelegations(numVals, numDels int) sdk.ValAddress {
	// Pre-fund the not-bonded-pool with the total self-stake so
	// TestingUpdateValidator finds the tokens it expects.
	valOwners := s.RandomAccountAddresses(numVals)
	for _, o := range valOwners {
		s.FundAcc(o, sdk.NewCoins(sdk.NewInt64Coin("uluna", 1_000_000)))
		s.Require().NoError(s.App.BankKeeper.DelegateCoinsFromAccountToModule(
			s.Ctx, o, stakingtypes.NotBondedPoolName,
			sdk.NewCoins(sdk.NewInt64Coin("uluna", 1_000_000)),
		))
	}

	valAddrs := simtestutil.ConvertAddrsToValAddrs(valOwners)
	pks := simtestutil.CreateTestPubKeys(numVals)
	vals := make([]stakingtypes.Validator, numVals)
	for i := range vals {
		v := testutil.NewValidator(s.T(), valAddrs[i], pks[i])
		v, _ = v.AddTokensFromDel(math.NewInt(1_000_000))
		v = stakingkeeper.TestingUpdateValidator(s.App.StakingKeeper, s.Ctx, v, true)
		// Distribution rewards state is normally initialized in CreateValidator;
		// TestingUpdateValidator skips that path, so do it manually.
		s.Require().NoError(s.App.DistrKeeper.Hooks().AfterValidatorCreated(s.Ctx, valAddrs[i]))
		vals[i] = v
	}

	// Delegators all stake to vals[0]. Each delegation is 1M, total stake
	// across the chain ends up at numVals*1M + numDels*1M; the cap-hook
	// requires vals[0] tokens / total <= 20%, so caller should pick numVals
	// large enough to keep the ratio under the threshold.
	addrDels := s.RandomAccountAddresses(numDels)
	for _, d := range addrDels {
		s.FundAcc(d, sdk.NewCoins(sdk.NewInt64Coin("uluna", 1_000_000)))
		_, err := s.App.StakingKeeper.Delegate(s.Ctx, d, math.NewInt(1_000_000), stakingtypes.Unbonded, vals[0], true)
		s.Require().NoError(err)
	}

	_, err := s.App.StakingKeeper.ApplyAndReturnValidatorSetUpdates(s.Ctx)
	s.Require().NoError(err)

	return valAddrs[0]
}

// dropReverseIndex deletes every entry under the staking module's
// DelegationByValIndexKey (0x71) prefix.
//
// This simulates the IAVL state at heights *before* the cosmos-sdk staking
// v4→v5 migration ran (the migration that backfills 0x71 from the primary
// DelegationKey 0x31). Pre-migration archive state contains delegations under
// 0x31 but nothing under 0x71 — which is what causes the empty query result
// reported on the public archive LCDs at heights below 28214400.
func (s *ValidatorDelegationsSuite) dropReverseIndex() {
	storeKey := s.App.GetKey(stakingtypes.StoreKey)
	store := s.Ctx.KVStore(storeKey)

	iter := storetypes.KVStorePrefixIterator(store, stakingtypes.DelegationByValIndexKey)
	defer iter.Close()

	var keys [][]byte
	for ; iter.Valid(); iter.Next() {
		k := make([]byte, len(iter.Key()))
		copy(k, iter.Key())
		keys = append(keys, k)
	}
	for _, k := range keys {
		store.Delete(k)
	}
}

// TestValidatorDelegations_ReproducesArchiveBug reproduces the symptom seen on
// public Terra Classic archive LCDs at pre-v5-staking-migration heights:
// ValidatorDelegations returns an empty list because the SDK query iterates
// over the 0x71 reverse-index, which has no entries in pre-migration IAVL
// state.
//
// Pre-fix:  expect empty result (bug present).
// Post-fix: expect populated result (fix routes to a primary-key scan when the
//
//	queried height is below MainnetStakingV5Height for Columbus).
func (s *ValidatorDelegationsSuite) TestValidatorDelegations_ReproducesArchiveBug() {
	s.Setup(s.T(), types.ColumbusChainID)

	// 30 validators × 1M + 5 × 1M = 35M; vals[0] has 6M = 17.1% < 20% cap.
	valAddr := s.seedValidatorWithDelegations(30, 5)

	// Build the LegacyQueryServer the same way custom/staking/module.go does.
	querier := stakingkeeper.Querier{Keeper: s.App.StakingKeeper}
	ss := s.App.GetSubspace(stakingtypes.ModuleName)
	qs := customstaking.NewLegacyQueryServer(
		querier, ss, s.App.StakingKeeper,
		s.App.AppCodec(), s.App.GetKey(stakingtypes.StoreKey),
	)

	req := &stakingtypes.QueryValidatorDelegationsRequest{ValidatorAddr: valAddr.String()}

	// Use a query height above the v8 upgrade height so ensureLegacyParams
	// takes the LegacyHandlingNone path and doesn't try to read non-existent
	// legacy params from the subspace.
	queryCtx := s.Ctx.WithBlockHeight(28214399)

	// Sanity: with the reverse-index intact the query returns all 5 delegations.
	resp, err := qs.ValidatorDelegations(queryCtx, req)
	s.Require().NoError(err)
	s.Require().Len(resp.DelegationResponses, 5, "sanity: index intact, should return all delegations")

	// Simulate pre-migration archive state by wiping the 0x71 reverse-index.
	s.dropReverseIndex()

	resp, err = qs.ValidatorDelegations(queryCtx, req)
	s.Require().NoError(err)

	// THIS is the assertion that fails before the fix and passes after it.
	s.Require().Len(
		resp.DelegationResponses, 5,
		"pre-migration height must still return delegations (regression of archive-LCD bug)",
	)
}

// TestValidatorDelegations_PostMigrationUsesIndex ensures the fix doesn't change
// behavior at chain-head heights: with the reverse-index intact and queried at
// a post-v5-staking-migration height, the SDK's normal indexed path runs.
// (We assert this indirectly by dropping the index at a post-migration height
// and confirming the wrapper does NOT fall back to legacy iteration — i.e.
// returns empty, just as the unwrapped SDK query would.)
func (s *ValidatorDelegationsSuite) TestValidatorDelegations_PostMigrationUsesIndex() {
	s.Setup(s.T(), types.ColumbusChainID)

	valAddr := s.seedValidatorWithDelegations(30, 5)

	querier := stakingkeeper.Querier{Keeper: s.App.StakingKeeper}
	ss := s.App.GetSubspace(stakingtypes.ModuleName)
	qs := customstaking.NewLegacyQueryServer(
		querier, ss, s.App.StakingKeeper,
		s.App.AppCodec(), s.App.GetKey(stakingtypes.StoreKey),
	)

	req := &stakingtypes.QueryValidatorDelegationsRequest{ValidatorAddr: valAddr.String()}
	postCtx := s.Ctx.WithBlockHeight(28214400)

	s.dropReverseIndex()
	resp, err := qs.ValidatorDelegations(postCtx, req)
	s.Require().NoError(err)
	s.Require().Len(
		resp.DelegationResponses, 0,
		"at post-migration heights the legacy fallback must NOT trigger",
	)
}
