package dyncomm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v2/x/dyncomm/keeper"
	"github.com/classic-terra/core/v2/x/dyncomm/types"
)

// InitGenesis initializes default parameters
// and the keeper's address to pubkey map
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, data *types.GenesisState) {
	keeper.SetParams(ctx, data.Params)

	for _, commRate := range data.MinCommissionRates {
		keeper.SetDynCommissionRate(ctx, commRate.ValidatorAddress, *commRate.MinCommissionRate)
	}

}

// ExportGenesis writes the current store values
// to a genesis file, which can be imported again
// with InitGenesis
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) (data *types.GenesisState) {
	params := keeper.GetParams(ctx)
	var rates []types.MinCommissionRate

	rates = append(rates)
	keeper.IterateDynCommissionRates(ctx, func(rate types.MinCommissionRate) (stop bool) {
		rates = append(rates, rate)
		return false
	})

	return types.NewGenesisState(params, rates)
}
