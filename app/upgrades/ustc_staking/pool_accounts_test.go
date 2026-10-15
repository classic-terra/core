package ustc_staking_test

import (
	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"errors"
	apphelpers "github.com/classic-terra/core/v4/app/testing"
	upgrade "github.com/classic-terra/core/v4/app/upgrades/ustc_staking"
	staking "github.com/classic-terra/core/v4/x/ustcstaking"
	keeper "github.com/classic-terra/core/v4/x/ustcstaking/keeper"
	types "github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	vesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktest "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type failedMigrationModule struct{ staking.AppModule }

func (failedMigrationModule) Name() string             { return "failed_migration" }
func (failedMigrationModule) ConsensusVersion() uint64 { return 2 }

func TestUSTCStakingUpgradeRollsBackNormalizationOnMigrationFailure(t *testing.T) {
	var suite apphelpers.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "upgrade-rollback")
	ctx := suite.Ctx
	name := types.PrincipalPoolName
	addr := auth.NewModuleAddress(name)
	suite.App.AccountKeeper.GetModuleAccount(ctx, name)
	coins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 1))
	require.NoError(t, banktest.FundModuleAccount(ctx, suite.App.BankKeeper, name, coins))
	existing := suite.App.AccountKeeper.GetAccount(ctx, addr)
	suite.App.AccountKeeper.SetAccount(ctx, auth.NewBaseAccount(addr, nil, existing.GetAccountNumber(), existing.GetSequence()))
	before, err := suite.App.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	am := staking.NewAppModule(suite.App.AppCodec(), suite.App.UstcStakingKeeper)
	manager := module.NewManager(am, failedMigrationModule{am})
	cfg := module.NewConfigurator(suite.App.AppCodec(), suite.App.MsgServiceRouter(), suite.App.GRPCQueryRouter())
	failure := errors.New("injected migration failure")
	require.NoError(t, cfg.RegisterMigration("failed_migration", 1, func(sdk.Context) error { return failure }))
	handler := upgrade.CreateUSTCStakingUpgradeHandler(manager, cfg, nil, suite.App.AppKeepers)
	_, err = handler(ctx, upgradetypes.Plan{Name: upgrade.UpgradeName}, module.VersionMap{"failed_migration": 1})
	require.ErrorContains(t, err, failure.Error())
	_, isModule := suite.App.AccountKeeper.GetAccount(ctx, addr).(sdk.ModuleAccountI)
	require.False(t, isModule)
	require.Equal(t, coins, suite.App.BankKeeper.GetAllBalances(ctx, addr))
	after, err := suite.App.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestUSTCStakingRepeatHandlerPreservesLiveCustody(t *testing.T) {
	var suite apphelpers.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "live-custody")
	ctx := suite.Ctx
	owner := suite.TestAccs[0]
	duration := time.Hour
	p := types.DefaultParams()
	p.LockTiers = []types.LockTier{{Id: 1, Duration: &duration, Multiplier: sdkmath.LegacyOneDec()}}
	suite.App.UstcStakingKeeper.SetParams(ctx, p)
	suite.FundAcc(owner, sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 100)))
	_, err := keeper.NewMsgServerImpl(suite.App.UstcStakingKeeper).Stake(ctx, &types.MsgStake{Owner: owner.String(), Amount: sdk.NewInt64Coin(types.BondDenom, 100), LockTierId: 1})
	require.NoError(t, err)
	require.NoError(t, banktest.FundModuleAccount(ctx, suite.App.BankKeeper, types.RewardPoolName, sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 40))))
	before := staking.ExportGenesis(ctx, suite.App.UstcStakingKeeper)
	pool, err := suite.App.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	manager := module.NewManager(staking.NewAppModule(suite.App.AppCodec(), suite.App.UstcStakingKeeper))
	cfg := module.NewConfigurator(suite.App.AppCodec(), suite.App.MsgServiceRouter(), suite.App.GRPCQueryRouter())
	handler := upgrade.CreateUSTCStakingUpgradeHandler(manager, cfg, nil, suite.App.AppKeepers)
	_, err = handler(ctx, upgradetypes.Plan{Name: upgrade.UpgradeName}, module.VersionMap{types.ModuleName: 2})
	require.NoError(t, err)
	require.Equal(t, before, staking.ExportGenesis(ctx, suite.App.UstcStakingKeeper))
	require.Equal(t, sdkmath.NewInt(100), suite.App.BankKeeper.GetBalance(ctx, auth.NewModuleAddress(types.PrincipalPoolName), types.BondDenom).Amount)
	require.Equal(t, sdkmath.NewInt(40), suite.App.BankKeeper.GetBalance(ctx, auth.NewModuleAddress(types.RewardPoolName), types.BondDenom).Amount)
	after, err := suite.App.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, pool, after)
}

func TestUSTCStakingUpgradeRejectsMalformedExistingVersion(t *testing.T) {
	for _, version := range []uint64{0, 1, 3} {
		t.Run(sdkmath.NewIntFromUint64(version).String(), func(t *testing.T) {
			var suite apphelpers.KeeperTestHelper
			suite.SetT(t)
			suite.Setup(t, "bad-upgrade-state")
			ctx := suite.Ctx
			addr := auth.NewModuleAddress(types.PrincipalPoolName)
			existing := suite.App.AccountKeeper.GetModuleAccount(ctx, types.PrincipalPoolName)
			suite.App.AccountKeeper.SetAccount(ctx, auth.NewBaseAccount(addr, nil, existing.GetAccountNumber(), existing.GetSequence()))
			manager := module.NewManager(staking.NewAppModule(suite.App.AppCodec(), suite.App.UstcStakingKeeper))
			cfg := module.NewConfigurator(suite.App.AppCodec(), suite.App.MsgServiceRouter(), suite.App.GRPCQueryRouter())
			handler := upgrade.CreateUSTCStakingUpgradeHandler(manager, cfg, nil, suite.App.AppKeepers)
			require.NotPanics(t, func() {
				_, err := handler(ctx, upgradetypes.Plan{Name: upgrade.UpgradeName}, module.VersionMap{types.ModuleName: version})
				require.Error(t, err)
			})
		})
	}
}

func TestUSTCStakingUpgradePrefundedPools(t *testing.T) {
	for _, name := range []string{types.PrincipalPoolName, types.RewardPoolName} {
		for _, kind := range []string{"absent", "base", "vesting", "permissions"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				var suite apphelpers.KeeperTestHelper
				suite.SetT(t)
				suite.Setup(t, "prefunded-upgrade")
				vesting.RegisterInterfaces(suite.App.InterfaceRegistry())
				ctx := suite.Ctx
				store := ctx.KVStore(suite.App.GetKey(types.StoreKey))
				store.Delete(types.ParamsKey)
				store.Delete(types.RewardStateKey)
				store.Delete(types.NextPositionIDKey)
				addr := auth.NewModuleAddress(name)
				old := suite.App.AccountKeeper.GetModuleAccount(ctx, name)
				number := old.GetAccountNumber()
				coins := sdk.NewCoins(sdk.NewInt64Coin("uusd", 1), sdk.NewInt64Coin("uluna", 2))
				if kind == "absent" {
					suite.App.AccountKeeper.RemoveAccount(ctx, old)
				} else {
					require.NoError(t, banktest.FundModuleAccount(ctx, suite.App.BankKeeper, name, coins))
					base := auth.NewBaseAccount(addr, nil, number, 9)
					switch kind {
					case "base":
						suite.App.AccountKeeper.SetAccount(ctx, base)
					case "vesting":
						account, err := vesting.NewDelayedVestingAccount(base, coins, ctx.BlockTime().Unix()+1000)
						require.NoError(t, err)
						suite.App.AccountKeeper.SetAccount(ctx, account)
					case "permissions":
						suite.App.AccountKeeper.SetAccount(ctx, auth.NewModuleAccount(base, name, auth.Minter))
					}
				}
				before, _ := suite.App.DistrKeeper.FeePool.Get(ctx)
				supply := suite.App.BankKeeper.GetSupply(ctx, "uusd")
				manager := module.NewManager(staking.NewAppModule(suite.App.AppCodec(), suite.App.UstcStakingKeeper))
				cfg := module.NewConfigurator(suite.App.AppCodec(), suite.App.MsgServiceRouter(), suite.App.GRPCQueryRouter())
				handler := upgrade.CreateUSTCStakingUpgradeHandler(manager, cfg, nil, suite.App.AppKeepers)
				require.NotPanics(t, func() {
					vm, err := handler(ctx, upgradetypes.Plan{Name: upgrade.UpgradeName}, module.VersionMap{})
					require.NoError(t, err)
					require.Equal(t, uint64(2), vm[types.ModuleName])
				})
				account := suite.App.AccountKeeper.GetAccount(ctx, addr).(sdk.ModuleAccountI)
				require.Equal(t, name, account.GetName())
				require.Empty(t, account.GetPermissions())
				require.Nil(t, account.GetPubKey())
				if kind != "absent" {
					require.Equal(t, number, account.GetAccountNumber())
					require.Equal(t, uint64(9), account.GetSequence())
				}
				require.True(t, suite.App.BankKeeper.GetBalance(ctx, addr, types.BondDenom).IsZero())
				if kind != "absent" {
					require.Equal(t, sdkmath.NewInt(2), suite.App.BankKeeper.GetBalance(ctx, addr, "uluna").Amount)
				}
				after, err := suite.App.DistrKeeper.FeePool.Get(ctx)
				require.NoError(t, err)
				expected := before.CommunityPool
				if kind != "absent" {
					expected = expected.Add(sdk.NewDecCoinsFromCoins(sdk.NewInt64Coin(types.BondDenom, 1))...)
				}
				require.Equal(t, expected, after.CommunityPool)
				require.Equal(t, supply, suite.App.BankKeeper.GetSupply(ctx, "uusd"))
				require.True(t, suite.App.UstcStakingKeeper.GetRewardState(ctx).TotalShares.Equal(sdkmath.ZeroInt()))
				require.NoError(t, suite.App.UstcStakingKeeper.ValidateState(ctx))
			})
		}
	}
}
