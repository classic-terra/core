package v5

import (
	"github.com/classic-terra/core/v3/custom/gov/types/v2custom"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	return migrateParams(ctx, storeKey, cdc)
}

func migrateParams(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(storeKey)

	bz := store.Get(govtypes.ParamsKey)
	var params govv1.Params
	err := cdc.Unmarshal(bz, &params)
	if err != nil {
		return err
	}

	defaultParams := v2custom.DefaultParams()
	newParams := v2custom.NewParams(
		params.MinDeposit,
		*params.MaxDepositPeriod,
		*params.VotingPeriod,
		params.Quorum,
		params.Threshold,
		params.VetoThreshold,
		params.MinInitialDepositRatio,
		params.BurnProposalDepositPrevote,
		params.BurnVoteQuorum,
		params.BurnVoteVeto,
		defaultParams.MinUsdDeposit,
	)

	bz, err = cdc.Marshal(&newParams)
	if err != nil {
		return err
	}

	store.Set(govtypes.ParamsKey, bz)

	return nil
}
