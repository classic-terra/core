package keeper_test

import (
	"github.com/classic-terra/core/v3/x/tax2gas/types"
)

func (suite *KeeperTestSuite) TestMsgUpdateParams() {
	// default params
	params := types.DefaultParams()

	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "empty params",
			input: &types.MsgUpdateParams{
				Authority: suite.keeper.GetAuthority(),
				Params:    types.Params{},
			},
			expErr:    true,
			expErrMsg: "must provide at least 1 gas prices",
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: suite.keeper.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			_, err := suite.msgServer.UpdateParams(suite.ctx, tc.input)

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
