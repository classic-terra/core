package keeper

import (
	"bytes"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOwnerPrefixSeparatesVariableLengthAddresses(t *testing.T) {
	ctx, k := newKeeperTest(t)
	a := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	b := sdk.AccAddress(append(append([]byte{}, a...), bytes.Repeat([]byte{2}, 12)...))
	require.NoError(t, k.SetPosition(ctx, types.Position{Id: 1, Owner: a.String()}))
	require.NoError(t, k.SetPosition(ctx, types.Position{Id: 2, Owner: b.String()}))
	for _, owner := range []sdk.AccAddress{a, b} {
		res, err := NewQueryServerImpl(k).PositionsByOwner(ctx, &types.QueryPositionsByOwnerRequest{Owner: owner.String()})
		require.NoError(t, err)
		require.Len(t, res.Positions, 1)
		require.Equal(t, owner.String(), res.Positions[0].Owner)
		key := types.OwnerPositionKey(owner, 3)
		decoded, id, err := types.OwnerFromPositionKey(key)
		require.NoError(t, err)
		require.Equal(t, owner, decoded)
		require.Equal(t, uint64(3), id)
	}
}
