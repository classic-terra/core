package keeper_test

import (
	"context"

	"github.com/classic-terra/core/v3/x/tax2gas/types"
)

func (suite *KeeperTestSuite) TestQueryParams() {
	res, err := suite.queryClient.Params(context.Background(), &types.QueryParamsRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(suite.keeper.GetParams(suite.ctx), res.GetParams())
}
