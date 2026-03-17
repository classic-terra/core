package ante

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// TreasuryKeeper for tax charging & recording
type TreasuryKeeper interface {
	RecordEpochTaxProceeds(ctx sdk.Context, delta sdk.Coins)
	GetTaxRate(ctx sdk.Context) (taxRate sdkmath.LegacyDec)
	GetTaxCap(ctx sdk.Context, denom string) (taxCap sdkmath.Int)
	GetBurnSplitRate(ctx sdk.Context) sdkmath.LegacyDec
	GetMinInitialDepositRatio(ctx sdk.Context) sdkmath.LegacyDec
	GetOracleSplitRate(ctx sdk.Context) sdkmath.LegacyDec
}

// OracleKeeper for feeder validation
type OracleKeeper interface {
	ValidateFeeder(ctx sdk.Context, feederAddr sdk.AccAddress, validatorAddr sdk.ValAddress) error
}

// BankKeeper defines the contract needed for supply related APIs (noalias)
type BankKeeper interface {
	IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error
	SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

type DistrKeeper interface {
	FundCommunityPool(ctx context.Context, amount sdk.Coins, sender sdk.AccAddress) error
}

type GovKeeper interface {
	GetDepositParams(ctx sdk.Context) govv1.DepositParams
}

type TaxKeeper interface {
	GetBurnTaxRate(ctx sdk.Context) sdkmath.LegacyDec
}
