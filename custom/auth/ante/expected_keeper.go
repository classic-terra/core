package ante

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// TreasuryKeeper for tax charging & recording
type TreasuryKeeper interface {
	RecordEpochTaxProceeds(ctx sdk.Context, delta sdk.Coins)
	GetTaxRate(ctx sdk.Context) (taxRate sdk.Dec)
	GetTaxCap(ctx sdk.Context, denom string) (taxCap math.Int)
	GetBurnSplitRate(ctx sdk.Context) sdk.Dec
	// HasBurnTaxExemptionAddress(ctx sdk.Context, addresses ...string) bool
	// HasBurnTaxExemptionContract(ctx sdk.Context, address string) bool
	GetMinInitialDepositRatio(ctx sdk.Context) sdk.Dec
	GetOracleSplitRate(ctx sdk.Context) sdk.Dec
}

// OracleKeeper for feeder validation
type OracleKeeper interface {
	ValidateFeeder(ctx sdk.Context, feederAddr sdk.AccAddress, validatorAddr sdk.ValAddress) error
}

// BankKeeper defines the contract needed for supply related APIs (noalias)
type BankKeeper interface {
	IsSendEnabledCoins(ctx sdk.Context, coins ...sdk.Coin) error
	SendCoins(ctx sdk.Context, from, to sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx sdk.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
}

type DistrKeeper interface {
	FundCommunityPool(ctx sdk.Context, amount sdk.Coins, sender sdk.AccAddress) error
	GetFeePool(ctx sdk.Context) distributiontypes.FeePool
	GetCommunityTax(ctx sdk.Context) math.LegacyDec
	SetFeePool(ctx sdk.Context, feePool distributiontypes.FeePool)
}

type GovKeeper interface {
	GetDepositParams(ctx sdk.Context) govv1.DepositParams
}

type TaxKeeper interface {
	GetBurnTaxRate(ctx sdk.Context) sdk.Dec
}
