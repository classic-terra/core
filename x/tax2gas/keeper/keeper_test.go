package keeper_test

import (
	"testing"

	"github.com/classic-terra/core/v3/x/tax2gas/keeper"
	"github.com/classic-terra/core/v3/x/tax2gas/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/suite"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	keeper keeper.Keeper

	queryClient types.QueryClient
	msgServer   types.MsgServer

	encCfg moduletestutil.TestEncodingConfig
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	key := sdk.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, sdk.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(tmproto.Header{Time: tmtime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	// gomock initializations

	suite.ctx = ctx
	suite.keeper = keeper.NewKeeper(
		encCfg.Codec,
		key,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	querier := keeper.NewQuerier(suite.keeper)
	types.RegisterQueryServer(queryHelper, querier)
	queryClient := types.NewQueryClient(queryHelper)

	suite.queryClient = queryClient
	suite.msgServer = keeper.NewMsgServerImpl(suite.keeper)
	suite.encCfg = encCfg
}

func (suite *KeeperTestSuite) TestGetAuthority() {
	NewKeeperWithAuthority := func(authority string) keeper.Keeper {
		return keeper.NewKeeper(
			moduletestutil.MakeTestEncodingConfig().Codec,
			sdk.NewKVStoreKey(types.StoreKey),
			authority,
		)
	}

	tests := map[string]string{
		"some random account":    "cosmos139f7kncmglres2nf3h4hc4tade85ekfr8sulz5",
		"gov module account":     authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		"another module account": authtypes.NewModuleAddress(minttypes.ModuleName).String(),
	}

	for name, expected := range tests {
		suite.T().Run(name, func(t *testing.T) {
			kpr := NewKeeperWithAuthority(expected)
			actual := kpr.GetAuthority()
			suite.Require().Equal(expected, actual)
		})
	}
}

func (suite *KeeperTestSuite) TestSetParams() {
	ctx, tax2gasKeeper := suite.ctx, suite.keeper
	require := suite.Require()

	tax2gasKeeper.SetParams(ctx, types.DefaultParams())
	tests := []struct {
		name    string
		params  types.Params
		expFail bool
	}{
		{
			name:    "empty params",
			params:  types.Params{},
			expFail: true,
		},
		{
			name:    "default params",
			params:  types.DefaultParams(),
			expFail: false,
		},
	}

	for _, tc := range tests {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := tax2gasKeeper.SetParams(ctx, tc.params)
			if tc.expFail {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}
