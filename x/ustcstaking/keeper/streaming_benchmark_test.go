package keeper

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func BenchmarkStreamingAccrual(b *testing.B) {
	for _, population := range []int{1, 100_000} {
		b.Run(fmt.Sprintf("positions_%d", population), func(b *testing.B) {
			ctx, k := newKeeperTest(b)
			params := types.DefaultParams() // zero rate isolates constant-work settlement
			k.SetParams(ctx, params)
			state := newTestRewardState(ctx, math.LegacyZeroDec(), math.OneInt())
			state.UnallocatedRewards = math.OneInt()
			k.SetRewardState(ctx, state)
			k.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewInt64Coin(types.BondDenom, 1)
			store := ctx.KVStore(k.storeKey)
			positionBytes := k.cdc.MustMarshal(&types.Position{Id: 1, Status: types.PositionStatus_POSITION_STATUS_WITHDRAWN})
			for id := 1; id <= population; id++ {
				key := append(append([]byte(nil), types.PositionKeyPrefix...), sdk.Uint64ToBigEndian(uint64(id))...)
				store.Set(key, positionBytes)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				now := ctx.BlockTime().Add(time.Duration(i+1) * time.Second)
				if err := k.AccrueRewards(ctx.WithBlockTime(now)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
