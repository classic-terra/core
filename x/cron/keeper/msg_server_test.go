package keeper

import (
	"testing"

	"github.com/classic-terra/core/v4/x/cron/types"
	"github.com/stretchr/testify/require"
)

func TestMsgServerAddScheduleAuthority(t *testing.T) {
	input := createTestInput(t)
	server := NewMsgServerImpl(input.Keeper)

	_, err := server.AddSchedule(input.Ctx, &types.MsgAddSchedule{
		Authority: input.Keeper.GetAuthority(),
		Name:      "job",
		Period:    2,
		Msgs:      []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"ping":{}}`}},
	})
	require.NoError(t, err)

	_, err = server.AddSchedule(input.Ctx, &types.MsgAddSchedule{
		Authority: "bad",
		Name:      "job2",
		Period:    2,
		Msgs:      []types.MsgExecuteContract{{Contract: "terra1contract", Msg: `{"ping":{}}`}},
	})
	require.Error(t, err)
}

func TestMsgServerUpdateParams(t *testing.T) {
	input := createTestInput(t)
	server := NewMsgServerImpl(input.Keeper)

	_, err := server.UpdateParams(input.Ctx, &types.MsgUpdateParams{
		Authority: input.Keeper.GetAuthority(),
		Params:    types.NewParams(7),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(7), input.Keeper.GetParams(input.Ctx).Limit)
}
