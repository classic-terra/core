package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

type CommunityPoolKeeper interface {
	DistributeFromCommunityPoolToModule(ctx context.Context, amount sdk.Coins, recipientModule string) error
	ReturnRewardDustToCommunityPool(ctx context.Context, amount sdk.Coins) error
}
