package ustc_staking_test

import (
	"testing"

	sdklog "cosmossdk.io/log"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	apphelpers "github.com/classic-terra/core/v4/app/testing"
	ustcstakingupgrade "github.com/classic-terra/core/v4/app/upgrades/ustc_staking"
	ustcstaking "github.com/classic-terra/core/v4/x/ustcstaking"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
)

func TestUpgradeAddsOnlyUSTCStakingStore(t *testing.T) {
	require.Equal(t, "v14_5", ustcstakingupgrade.UpgradeName)
	require.Equal(t, []string{ustcstakingtypes.StoreKey}, ustcstakingupgrade.Upgrade.StoreUpgrades.Added)
	require.Empty(t, ustcstakingupgrade.Upgrade.StoreUpgrades.Deleted)
	require.Empty(t, ustcstakingupgrade.Upgrade.StoreUpgrades.Renamed)
}

func TestUpgradeStoreLoaderPreservesOldStoreAndPersistsNewStoreAcrossRestart(t *testing.T) {
	db := dbm.NewMemDB()
	oldKey := storetypes.NewKVStoreKey("bank")
	oldStores := store.NewCommitMultiStore(db, sdklog.NewNopLogger(), storemetrics.NewNoOpMetrics())
	oldStores.MountStoreWithDB(oldKey, storetypes.StoreTypeIAVL, nil)
	require.NoError(t, oldStores.LoadLatestVersion())
	oldCtx := sdk.NewContext(oldStores, tmproto.Header{Height: 1, ChainID: "pre-ustc-chain"}, false, sdklog.NewNopLogger())
	oldCtx.KVStore(oldKey).Set([]byte("existing-account-balance"), []byte("2500000stake"))
	oldCommit := oldStores.Commit()
	require.EqualValues(t, 1, oldCommit.Version)

	newKey := storetypes.NewKVStoreKey(ustcstakingtypes.StoreKey)
	upgradedStores := store.NewCommitMultiStore(db, sdklog.NewNopLogger(), storemetrics.NewNoOpMetrics())
	upgradedStores.MountStoreWithDB(oldKey, storetypes.StoreTypeIAVL, nil)
	upgradedStores.MountStoreWithDB(newKey, storetypes.StoreTypeIAVL, nil)
	require.NoError(t, upgradedStores.LoadLatestVersionAndUpgrade(&ustcstakingupgrade.Upgrade.StoreUpgrades))
	upgradedCtx := sdk.NewContext(upgradedStores, tmproto.Header{Height: 2, ChainID: "pre-ustc-chain"}, false, sdklog.NewNopLogger())
	require.Equal(t, []byte("2500000stake"), upgradedCtx.KVStore(oldKey).Get([]byte("existing-account-balance")))
	newStoreIterator := upgradedCtx.KVStore(newKey).Iterator(nil, nil)
	require.False(t, newStoreIterator.Valid())
	newStoreIterator.Close()
	upgradedCommit := upgradedStores.Commit()
	require.EqualValues(t, 2, upgradedCommit.Version)

	// Reopen the same database to model process restart after the store upgrade.
	restartedStores := store.NewCommitMultiStore(db, sdklog.NewNopLogger(), storemetrics.NewNoOpMetrics())
	restartedStores.MountStoreWithDB(oldKey, storetypes.StoreTypeIAVL, nil)
	restartedStores.MountStoreWithDB(newKey, storetypes.StoreTypeIAVL, nil)
	require.NoError(t, restartedStores.LoadLatestVersion())
	restartedCtx := sdk.NewContext(restartedStores, tmproto.Header{Height: 2, ChainID: "pre-ustc-chain"}, false, sdklog.NewNopLogger())
	require.Equal(t, []byte("2500000stake"), restartedCtx.KVStore(oldKey).Get([]byte("existing-account-balance")))
	newStoreIterator = restartedCtx.KVStore(newKey).Iterator(nil, nil)
	require.False(t, newStoreIterator.Valid())
	newStoreIterator.Close()
}

func TestUpgradeHandlerRunsWithRealAppKeepers(t *testing.T) {
	var suite apphelpers.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-staking-upgrade")

	store := suite.Ctx.KVStore(suite.App.GetKey(ustcstakingtypes.StoreKey))
	store.Delete(ustcstakingtypes.ParamsKey)
	store.Delete(ustcstakingtypes.RewardStateKey)
	store.Delete(ustcstakingtypes.NextPositionIDKey)
	for _, moduleName := range []string{ustcstakingtypes.PrincipalPoolName, ustcstakingtypes.RewardPoolName} {
		account, _ := suite.App.AccountKeeper.GetModuleAccountAndPermissions(suite.Ctx, moduleName)
		require.NotNil(t, account)
		suite.App.AccountKeeper.RemoveAccount(suite.Ctx, account)
	}
	initialSupply := suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom)

	manager := module.NewManager(ustcstaking.NewAppModule(suite.App.AppCodec(), suite.App.UstcStakingKeeper))
	configurator := module.NewConfigurator(suite.App.AppCodec(), suite.App.MsgServiceRouter(), suite.App.GRPCQueryRouter())
	handler := ustcstakingupgrade.CreateUSTCStakingUpgradeHandler(manager, configurator, nil, suite.App.AppKeepers)

	versionMap, err := handler(suite.Ctx, upgradetypes.Plan{Name: ustcstakingupgrade.UpgradeName}, module.VersionMap{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), versionMap[ustcstakingtypes.ModuleName])

	for _, moduleName := range []string{ustcstakingtypes.PrincipalPoolName, ustcstakingtypes.RewardPoolName} {
		address, _ := suite.App.AccountKeeper.GetModuleAddressAndPermissions(moduleName)
		account := suite.App.AccountKeeper.GetAccount(suite.Ctx, address)
		require.NotNil(t, account)
		_, permissions := suite.App.AccountKeeper.GetModuleAccountAndPermissions(suite.Ctx, moduleName)
		require.Empty(t, permissions)
	}
	require.Equal(t, uint64(1), suite.App.UstcStakingKeeper.GetNextPositionID(suite.Ctx))
	require.NotNil(t, store.Get(ustcstakingtypes.ParamsKey))
	require.NotNil(t, store.Get(ustcstakingtypes.RewardStateKey))
	require.NotNil(t, store.Get(ustcstakingtypes.NextPositionIDKey))
	require.True(t, suite.App.UstcStakingKeeper.GetRewardState(suite.Ctx).RewardIndex.IsZero())
	require.Zero(t, suite.App.UstcStakingKeeper.GetRewardState(suite.Ctx).TotalShares.Int64())
	require.Equal(t, initialSupply, suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom))
	require.NoError(t, suite.App.UstcStakingKeeper.ValidateState(suite.Ctx))
}
