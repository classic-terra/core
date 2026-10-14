package keepers

import (
	"context"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

type communityPoolBankKeeper interface {
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amount sdk.Coins) error
}

func (a communityPoolAdapter) ReturnRewardDustToCommunityPool(ctx context.Context, amount sdk.Coins) error {
	if len(amount) != 1 || amount[0].Denom != ustcstakingtypes.BondDenom || !amount.IsAllPositive() {
		return ustcstakingtypes.ErrInvalidDenom
	}
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	ctx = cached
	pool, err := a.feePool.Get(ctx)
	if err != nil {
		return err
	}
	if err = a.bankKeeper.SendCoinsFromModuleToModule(ctx, ustcstakingtypes.RewardPoolName, types.ModuleName, amount); err != nil {
		return err
	}
	pool.CommunityPool = pool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(amount...)...)
	if err := a.feePool.Set(ctx, pool); err != nil {
		return err
	}
	write()
	return nil
}

type distributionFeePool interface {
	Get(ctx context.Context) (types.FeePool, error)
	Set(ctx context.Context, feePool types.FeePool) error
}

type communityPoolAdapter struct {
	feePool    distributionFeePool
	bankKeeper communityPoolBankKeeper
}

func (a communityPoolAdapter) DistributeFromCommunityPoolToModule(ctx context.Context, amount sdk.Coins, recipientModule string) error {
	feePool, err := a.feePool.Get(ctx)
	if err != nil {
		return err
	}
	newPool, negative := feePool.CommunityPool.SafeSub(sdk.NewDecCoinsFromCoins(amount...))
	if negative {
		return types.ErrBadDistribution
	}
	// Distribution's account-oriented helper cannot target a module account.
	// Keep the reward pool blocked from ordinary sends and move funds module-to-module.
	if err := a.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, recipientModule, amount); err != nil {
		return err
	}
	feePool.CommunityPool = newPool
	return a.feePool.Set(ctx, feePool)
}
