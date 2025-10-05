package types

import (
	"context"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper expected account keeper
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

// BankKeeper expected bank keeper
type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// MarketKeeper expected market keeper
type MarketKeeper interface {
	ComputeInternalSwap(ctx sdk.Context, offerCoin sdk.DecCoin, askDenom string) (sdk.DecCoin, error)
}

// StakingKeeper expected keeper for staking module
type StakingKeeper interface {
	TotalBondedTokens(context.Context) (math.Int, error) // total bonded tokens within the validator set
}

// OracleKeeper defines expected oracle keeper
type OracleKeeper interface {
	Whitelist(ctx sdk.Context) (res oracletypes.DenomList)

	// only used for test purpose
	SetLunaExchangeRate(ctx sdk.Context, denom string, exchangeRate sdkmath.LegacyDec)
	SetWhitelist(ctx sdk.Context, whitelist oracletypes.DenomList)
}
