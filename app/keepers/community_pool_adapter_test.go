package keepers

import (
	"context"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"errors"
	ustctypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"
	"testing"
)

type dustAdapterFixture struct {
	key                 *storetypes.KVStoreKey
	bankError, feeError bool
}

func (f dustAdapterFixture) SendCoinsFromModuleToModule(ctx context.Context, from, to string, amount sdk.Coins) error {
	if from != ustctypes.RewardPoolName || to != distrtypes.ModuleName {
		return errors.New("unexpected transfer")
	}
	if f.bankError {
		return errors.New("injected bank failure")
	}
	sdk.UnwrapSDKContext(ctx).KVStore(f.key).Set([]byte("bank"), []byte(amount.AmountOf(ustctypes.BondDenom).String()))
	return nil
}
func (f dustAdapterFixture) Get(context.Context) (distrtypes.FeePool, error) {
	return distrtypes.FeePool{CommunityPool: sdk.NewDecCoinsFromCoins(sdk.NewInt64Coin(ustctypes.BondDenom, 2))}, nil
}
func (f dustAdapterFixture) Set(ctx context.Context, pool distrtypes.FeePool) error {
	sdk.UnwrapSDKContext(ctx).KVStore(f.key).Set([]byte("pool"), []byte(pool.CommunityPool.AmountOf(ustctypes.BondDenom).String()))
	if f.feeError {
		return errors.New("injected FeePool write failure")
	}
	return nil
}
func TestReturnRewardDustAdapterAtomicAccounting(t *testing.T) {
	for _, tc := range []struct {
		name      string
		bank, fee bool
	}{{"success", false, false}, {"bank failure", true, false}, {"FeePool failure", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			key := storetypes.NewKVStoreKey("dust-adapter")
			db := dbm.NewMemDB()
			ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
			ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
			require.NoError(t, ms.LoadLatestVersion())
			ctx := sdk.NewContext(ms, tmproto.Header{}, false, log.NewNopLogger())
			fixture := dustAdapterFixture{key: key, bankError: tc.bank, feeError: tc.fee}
			adapter := communityPoolAdapter{feePool: fixture, bankKeeper: fixture}
			err := adapter.ReturnRewardDustToCommunityPool(ctx, sdk.NewCoins(sdk.NewInt64Coin(ustctypes.BondDenom, 5)))
			if tc.bank || tc.fee {
				require.Error(t, err)
				require.Nil(t, ctx.KVStore(key).Get([]byte("bank")))
				require.Nil(t, ctx.KVStore(key).Get([]byte("pool")))
			} else {
				require.NoError(t, err)
				require.Equal(t, "5", string(ctx.KVStore(key).Get([]byte("bank"))))
				require.Equal(t, math.LegacyNewDec(7).String(), string(ctx.KVStore(key).Get([]byte("pool"))))
			}
		})
	}
}
