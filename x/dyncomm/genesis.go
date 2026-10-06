package dyncomm

import (
	"github.com/classic-terra/core/v4/x/dyncomm/keeper"
	"github.com/classic-terra/core/v4/x/dyncomm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// InitGenesis initializes default parameters
// and the keeper's address to pubkey map
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, data *types.GenesisState) {
	keeper.SetParams(ctx, data.Params)

	for _, rate := range data.ValidatorCommissionRates {
		if rate.MinCommissionRate != nil {
			keeper.SetDynCommissionRate(ctx, rate.ValidatorAddress, *rate.MinCommissionRate)
		}
		if rate.TargetCommissionRate != nil {
			keeper.SetTargetCommissionRate(ctx, rate.ValidatorAddress, *rate.TargetCommissionRate)
		}
	}

	// iterate validators and set target rates for validators
	// that have no rate stored yet
	keeper.StakingKeeper.IterateValidators(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		val := validator.(stakingtypes.Validator)
		if bz := ctx.KVStore(keeper.StoreKey()).Get(types.GetMinCommissionRatesKey(val.OperatorAddress)); bz == nil {
			keeper.SetTargetCommissionRate(ctx, val.OperatorAddress, val.Commission.Rate)
		}
		return false
	})

	err := keeper.UpdateAllBondedValidatorRates(ctx)
	if err != nil {
		panic("could not initialize genesis")
	}
}

// ExportGenesis writes the current store values
// to a genesis file, which can be imported again
// with InitGenesis
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) (data *types.GenesisState) {
	params := keeper.GetParams(ctx)
	var rates []types.ValidatorCommissionRate

	// rates = append(rates)
	keeper.IterateDynCommissionRates(ctx, func(rate types.ValidatorCommissionRate) (stop bool) {
		rates = append(rates, rate)
		return false
	})

	return types.NewGenesisState(params, rates)
}
