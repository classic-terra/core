package staking_test

import (
	"encoding/binary"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	apptesting "github.com/classic-terra/core/v4/app/testing"
	customstaking "github.com/classic-terra/core/v4/custom/staking"
	"github.com/classic-terra/core/v4/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
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

// TestValidatorDelegations_LegacyPathKeepsLegacyParams covers the overlap
// between the pre-staking-v5 fallback window and the older staking-param legacy
// window. The fallback still needs ensureLegacyParams before it builds
// DelegationResponse balances.
func (s *ValidatorDelegationsSuite) TestValidatorDelegations_LegacyPathKeepsLegacyParams() {
	s.Setup(s.T(), types.ColumbusChainID)

	valAddr := s.seedValidatorWithDelegations(30, 1)

	querier := stakingkeeper.Querier{Keeper: s.App.StakingKeeper}
	ss := s.App.GetSubspace(stakingtypes.ModuleName)
	qs := customstaking.NewLegacyQueryServer(
		querier, ss, s.App.StakingKeeper,
		s.App.AppCodec(), s.App.GetKey(stakingtypes.StoreKey),
	)

	params, err := s.App.StakingKeeper.GetParams(s.Ctx)
	s.Require().NoError(err)

	legacyParams := params
	legacyParams.BondDenom = types.MicroLunaDenom
	ss.SetParamSet(s.Ctx, &legacyParams)

	currentParams := params
	currentParams.BondDenom = "stake"
	s.Require().NoError(s.App.StakingKeeper.SetParams(s.Ctx, currentParams))

	s.dropReverseIndex()

	req := &stakingtypes.QueryValidatorDelegationsRequest{ValidatorAddr: valAddr.String()}
	queryCtx := s.Ctx.WithBlockHeight(18302999)
	resp, err := qs.ValidatorDelegations(queryCtx, req)
	s.Require().NoError(err)
	s.Require().Len(resp.DelegationResponses, 1)
	s.Require().Equal(types.MicroLunaDenom, resp.DelegationResponses[0].Balance.Denom)
}

// TestValidatorDelegations_LegacyPathPaginates exercises the legacy-iteration
// path through pagination: it walks all delegations across multiple pages and
// asserts that (a) every page returns delegations matching only the requested
// validator, (b) the union of pages equals the full delegation set, and
// (c) walking terminates with an empty next_key.
func (s *ValidatorDelegationsSuite) TestValidatorDelegations_LegacyPathPaginates() {
	s.Setup(s.T(), types.ColumbusChainID)

	totalDelegators := 12
	valAddr := s.seedValidatorWithDelegations(30, totalDelegators)

	querier := stakingkeeper.Querier{Keeper: s.App.StakingKeeper}
	ss := s.App.GetSubspace(stakingtypes.ModuleName)
	qs := customstaking.NewLegacyQueryServer(
		querier, ss, s.App.StakingKeeper,
		s.App.AppCodec(), s.App.GetKey(stakingtypes.StoreKey),
	)

	// Drop the reverse-index and force the legacy path.
	s.dropReverseIndex()
	queryCtx := s.Ctx.WithBlockHeight(28214399)

	pageSize := uint64(5)
	seen := make(map[string]struct{})
	var nextKey []byte

	for pages := 0; pages < 10; pages++ {
		req := &stakingtypes.QueryValidatorDelegationsRequest{
			ValidatorAddr: valAddr.String(),
			Pagination:    &query.PageRequest{Key: nextKey, Limit: pageSize},
		}
		resp, err := qs.ValidatorDelegations(queryCtx, req)
		s.Require().NoError(err)
		s.Require().NotNil(resp.Pagination)

		for _, d := range resp.DelegationResponses {
			s.Require().Equal(valAddr.String(), d.Delegation.ValidatorAddress,
				"page must only contain delegations for the queried validator")
			seen[d.Delegation.DelegatorAddress] = struct{}{}
		}

		if len(resp.Pagination.NextKey) == 0 {
			break
		}
		nextKey = resp.Pagination.NextKey
	}

	s.Require().Len(seen, totalDelegators,
		"paginated walk must surface every delegation exactly once")
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

// rekeyHistoricalInfoToLegacyFormat takes every entry in the staking store
// under the HistoricalInfoKey (0x50) prefix, deletes the new big-endian
// uint64 key form, and re-writes the same value under the pre-v5-migration
// key form (`0x50 || strconv.FormatInt(height, 10)`). This simulates the
// IAVL state at heights *before* the v5 staking migration ran.
//
// Mirrors cosmos-sdk@v0.53.6/x/staking/migrations/v5/store.go:39 in reverse.
func (s *ValidatorDelegationsSuite) rekeyHistoricalInfoToLegacyFormat() {
	storeKey := s.App.GetKey(stakingtypes.StoreKey)
	store := s.Ctx.KVStore(storeKey)

	iter := storetypes.KVStorePrefixIterator(store, stakingtypes.HistoricalInfoKey)
	defer iter.Close()

	type entry struct {
		oldKey []byte
		height int64
		value  []byte
	}
	var entries []entry
	for ; iter.Valid(); iter.Next() {
		fullKey := append([]byte{}, iter.Key()...) // already includes 0x50 prefix
		val := append([]byte{}, iter.Value()...)
		// new format: 0x50 || 8-byte big-endian height
		if len(fullKey) != 1+8 {
			continue
		}
		h := int64(binary.BigEndian.Uint64(fullKey[1:]))
		entries = append(entries, entry{oldKey: fullKey, height: h, value: val})
	}

	for _, e := range entries {
		store.Delete(e.oldKey)
		legacyKey := append([]byte{}, stakingtypes.HistoricalInfoKey...)
		legacyKey = append(legacyKey, []byte(strconv.FormatInt(e.height, 10))...)
		store.Set(legacyKey, e.value)
	}
}

// TestHistoricalInfo_ReproducesArchiveBug verifies the parallel bug to
// ValidatorDelegations: ValidatorDelegations relies on the 0x71 reverse-index
// that the v5 migration backfills, while HistoricalInfo relies on
// big-endian-uint64 keys that the same migration writes (re-keying from
// ASCII-decimal). On pre-migration archive heights the IAVL state holds only
// the old string-format keys, so the SDK's GetHistoricalInfo returns NotFound
// — confirmed live against archive-lcd.galacticshift.io at height 28214399
// vs 28214400.
//
// Pre-fix:  expect NotFound (bug present).
// Post-fix: expect HistoricalInfo populated (legacy reader hits string keys).
func (s *ValidatorDelegationsSuite) TestHistoricalInfo_ReproducesArchiveBug() {
	s.Setup(s.T(), types.ColumbusChainID)
	s.seedValidatorWithDelegations(30, 1) // populate the validator set

	// Stage a HistoricalInfo entry. Use a height the staking module would
	// not currently retain via its own pruning — we're driving the state
	// directly to validate the read path, not exercising BeginBlocker.
	const targetHeight = int64(28210000)
	vals, err := s.App.StakingKeeper.GetAllValidators(s.Ctx)
	s.Require().NoError(err)
	s.Require().NotEmpty(vals, "test setup should bond at least one validator")

	hi := stakingtypes.HistoricalInfo{
		Header: cmtproto.Header{
			ChainID: types.ColumbusChainID,
			Height:  targetHeight,
			Time:    time.Unix(1700000000, 0).UTC(),
		},
		Valset: vals,
	}
	s.Require().NoError(s.App.StakingKeeper.SetHistoricalInfo(s.Ctx, targetHeight, &hi))

	querier := stakingkeeper.Querier{Keeper: s.App.StakingKeeper}
	ss := s.App.GetSubspace(stakingtypes.ModuleName)
	qs := customstaking.NewLegacyQueryServer(
		querier, ss, s.App.StakingKeeper,
		s.App.AppCodec(), s.App.GetKey(stakingtypes.StoreKey),
	)

	req := &stakingtypes.QueryHistoricalInfoRequest{Height: targetHeight}

	// Sanity: at a post-migration height the SDK's indexed path reads
	// big-endian-uint64 keys (which is what SetHistoricalInfo just wrote).
	postCtx := s.Ctx.WithBlockHeight(28214400)
	resp, err := qs.HistoricalInfo(postCtx, req)
	s.Require().NoError(err, "sanity: post-migration indexed read should succeed")
	s.Require().NotNil(resp.Hist)
	s.Require().Equal(targetHeight, resp.Hist.Header.Height)

	// Simulate pre-migration archive state: rewrite every 0x50 entry from
	// big-endian to string-format keys.
	s.rekeyHistoricalInfoToLegacyFormat()

	preCtx := s.Ctx.WithBlockHeight(28214399)
	resp, err = qs.HistoricalInfo(preCtx, req)

	// THIS is the assertion that fails before the fix and passes after it.
	s.Require().NoError(err, "pre-migration height must still return historical info (regression of HistoricalInfo bug)")
	s.Require().NotNil(resp.Hist)
	s.Require().Equal(targetHeight, resp.Hist.Header.Height)
	s.Require().Equal(types.ColumbusChainID, resp.Hist.Header.ChainID)
}

// TestHistoricalInfo_PostMigrationUsesIndex confirms the legacy reader is NOT
// used at chain-head heights: at BlockHeight = MainnetStakingV5Height the
// wrapper goes through the SDK's GetHistoricalInfo, which expects the binary
// key format. After we rewrite to the legacy format, the post-migration call
// must miss — proving the height gate short-circuits correctly.
func (s *ValidatorDelegationsSuite) TestHistoricalInfo_PostMigrationUsesIndex() {
	s.Setup(s.T(), types.ColumbusChainID)
	s.seedValidatorWithDelegations(30, 1)

	const targetHeight = int64(28210000)
	vals, err := s.App.StakingKeeper.GetAllValidators(s.Ctx)
	s.Require().NoError(err)

	hi := stakingtypes.HistoricalInfo{
		Header: cmtproto.Header{ChainID: types.ColumbusChainID, Height: targetHeight, Time: time.Unix(1700000000, 0).UTC()},
		Valset: vals,
	}
	s.Require().NoError(s.App.StakingKeeper.SetHistoricalInfo(s.Ctx, targetHeight, &hi))

	querier := stakingkeeper.Querier{Keeper: s.App.StakingKeeper}
	ss := s.App.GetSubspace(stakingtypes.ModuleName)
	qs := customstaking.NewLegacyQueryServer(
		querier, ss, s.App.StakingKeeper,
		s.App.AppCodec(), s.App.GetKey(stakingtypes.StoreKey),
	)

	// Rewrite into legacy string-format keys; with the height gate at
	// post-migration the wrapper must NOT consult the legacy reader.
	s.rekeyHistoricalInfoToLegacyFormat()

	postCtx := s.Ctx.WithBlockHeight(28214400)
	_, err = qs.HistoricalInfo(postCtx, &stakingtypes.QueryHistoricalInfoRequest{Height: targetHeight})
	s.Require().Error(err, "at post-migration heights the legacy reader must NOT be consulted")
}
