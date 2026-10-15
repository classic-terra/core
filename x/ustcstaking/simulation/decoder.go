package simulation

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/kv"
)

func NewDecodeStore(cdc codec.Codec) func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		case bytes.HasPrefix(kvA.Key, types.PositionKeyPrefix):
			var positionA, positionB types.Position
			cdc.MustUnmarshal(kvA.Value, &positionA)
			cdc.MustUnmarshal(kvB.Value, &positionB)
			return fmt.Sprintf("Position %d: %v\nPosition %d: %v", positionA.Id, positionA, positionB.Id, positionB)
		case bytes.HasPrefix(kvA.Key, types.OwnerPositionKeyPrefix):
			return fmt.Sprintf("owner-index %X: %X\nowner-index %X: %X", kvA.Key, kvA.Value, kvB.Key, kvB.Value)
		case bytes.Equal(kvA.Key, types.RewardStateKey):
			var stateA, stateB types.RewardState
			cdc.MustUnmarshal(kvA.Value, &stateA)
			cdc.MustUnmarshal(kvB.Value, &stateB)
			return fmt.Sprintf("RewardState: %v\nRewardState: %v", stateA, stateB)
		case bytes.Equal(kvA.Key, types.ParamsKey):
			var paramsA, paramsB types.Params
			cdc.MustUnmarshal(kvA.Value, &paramsA)
			cdc.MustUnmarshal(kvB.Value, &paramsB)
			return fmt.Sprintf("Params: %v\nParams: %v", paramsA, paramsB)
		case bytes.Equal(kvA.Key, types.NextPositionIDKey):
			return fmt.Sprintf("next-position-id: %d\nnext-position-id: %d", decodeID(kvA.Value), decodeID(kvB.Value))
		case bytes.Equal(kvA.Key, types.ReservedRewardsKey):
			return fmt.Sprintf("reserved-rewards: %s\nreserved-rewards: %s", kvA.Value, kvB.Value)
		default:
			return fmt.Sprintf("unknown ustcstaking key prefix %s: %s", hex.EncodeToString(kvA.Key), hex.EncodeToString(kvA.Value))
		}
	}
}

func decodeID(value []byte) uint64 {
	if len(value) != 8 {
		return 0
	}
	var result uint64
	for _, b := range value {
		result = result<<8 | uint64(b)
	}
	return result
}
