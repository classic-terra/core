package types

import (
	context "context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"
)

// AccountKeeper is expected keeper for auth module
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI // only used for simulation
}

// BankKeeper defines expected supply keeper
type BankKeeper interface {
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error

	// only used for simulation
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	IsSendEnabledCoin(ctx context.Context, coin sdk.Coin) bool
}

// OracleKeeper defines expected oracle keeper
type OracleKeeper interface {
	GetLunaExchangeRate(ctx sdk.Context, denom string) (price math.LegacyDec, err error)
	GetTobinTax(ctx sdk.Context, denom string) (tobinTax math.LegacyDec, err error)

	// only used for simulation
	IterateLunaExchangeRates(ctx sdk.Context, handler func(denom string, exchangeRate math.LegacyDec) (stop bool))
	SetLunaExchangeRate(ctx sdk.Context, denom string, exchangeRate math.LegacyDec)
	SetTobinTax(ctx sdk.Context, denom string, tobinTax math.LegacyDec)
}
