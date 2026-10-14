package ustcstaking

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/x/ustcstaking/keeper"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	secp256k1 "github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

type exportTestBankKeeper struct{}

type exportTestCommunityPoolKeeper struct{}

func (exportTestCommunityPoolKeeper) ReturnRewardDustToCommunityPool(context.Context, sdk.Coins) error {
	return nil
}

func (exportTestCommunityPoolKeeper) DistributeFromCommunityPoolToModule(context.Context, sdk.Coins, string) error {
	return nil
}

func (exportTestBankKeeper) SendCoinsFromAccountToModule(context.Context, sdk.AccAddress, string, sdk.Coins) error {
	return nil
}

func (exportTestBankKeeper) SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error {
	return nil
}

func (exportTestBankKeeper) GetBalance(context.Context, sdk.AccAddress, string) sdk.Coin {
	return sdk.NewCoin(types.BondDenom, math.ZeroInt())
}

func (exportTestBankKeeper) GetAllBalances(context.Context, sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins()
}

func TestWithdrawnPositionExportsWithoutPanic(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stores := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stores.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stores.LoadLatestVersion())

	now := time.Now().UTC()
	ctx := sdk.NewContext(stores, tmproto.Header{Time: now}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(cdc, key, exportTestBankKeeper{}, exportTestCommunityPoolKeeper{})
	owner := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	maturedAt := now.Add(-time.Second)
	duration := time.Hour
	k.SetParams(ctx, types.DefaultParams())
	k.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.ZeroInt()})
	k.SetNextPositionID(ctx, 2)
	k.SetPosition(ctx, types.Position{
		Id:               1,
		Owner:            owner,
		Principal:        sdk.NewCoin(types.BondDenom, math.NewInt(100)),
		LockTierId:       1,
		Shares:           math.ZeroInt(),
		ShareMultiplier:  math.LegacyOneDec(),
		RewardDebt:       math.LegacyZeroDec(),
		LockDuration:     &duration,
		UnbondingEndTime: &maturedAt,
		Status:           types.PositionStatus_POSITION_STATUS_UNBONDING,
		ClaimableRewards: sdk.NewCoin(types.BondDenom, math.NewInt(5)),
	})

	server := keeper.NewMsgServerImpl(k)
	_, err := server.Withdraw(sdk.WrapSDKContext(ctx), &types.MsgWithdraw{Owner: owner, PositionId: 1})
	require.NoError(t, err)

	position, found := k.GetPosition(ctx, 1)
	require.True(t, found)
	require.Equal(t, types.PositionStatus_POSITION_STATUS_WITHDRAWN, position.Status)
	require.True(t, position.Principal.Amount.IsZero())

	var exported *types.GenesisState
	require.NotPanics(t, func() {
		exported = ExportGenesis(ctx, k)
	})
	require.Len(t, exported.Positions, 1)
}
