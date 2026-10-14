package keeper

import (
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStoredAuthorityCannotDelegateGovernance(t *testing.T) {
	ctx, k := newKeeperTest(t)
	wallet := testAddress()
	p := types.DefaultParams()
	p.Authority = wallet
	k.SetParams(ctx, p)
	server := NewMsgServerImpl(k)
	_, err := server.FundRewards(ctx, &types.MsgFundRewards{Authority: wallet, Amount: sdk.NewInt64Coin(types.BondDenom, 1)})
	require.ErrorIs(t, err, types.ErrUnauthorized)
	_, err = server.UpdateParams(ctx, &types.MsgUpdateParams{Authority: wallet, Params: types.DefaultParams()})
	require.ErrorIs(t, err, types.ErrUnauthorized)
	_, err = server.UpdateParams(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: types.DefaultParams()})
	require.NoError(t, err)
	require.Equal(t, k.GetAuthority(), k.GetParams(ctx).Authority)
	require.Zero(t, k.communityPoolKeeper.(*recordingCommunityPoolKeeper).calls)
}
