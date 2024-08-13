package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

// TreasuryKeeper for tax charging & recording
type TreasuryKeeper interface {
	RecordEpochTaxProceeds(ctx sdk.Context, delta sdk.Coins)
	GetTaxRate(ctx sdk.Context) (taxRate sdk.Dec)
	GetTaxCap(ctx sdk.Context, denom string) (taxCap math.Int)
	GetBurnSplitRate(ctx sdk.Context) sdk.Dec
	HasBurnTaxExemptionAddress(ctx sdk.Context, addresses ...string) bool
	HasBurnTaxExemptionContract(ctx sdk.Context, address string) bool
	GetMinInitialDepositRatio(ctx sdk.Context) sdk.Dec
	GetOracleSplitRate(ctx sdk.Context) sdk.Dec
}

// BankKeeper defines the contract needed for supply related APIs (noalias)
type BankKeeper interface {
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	IsSendEnabledCoins(ctx sdk.Context, coins ...sdk.Coin) error
	SendCoins(ctx sdk.Context, from, to sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx sdk.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type FeegrantKeeper interface {
	GetAllowance(ctx sdk.Context, granter, grantee sdk.AccAddress) (feegrant.FeeAllowanceI, error)
	UseGrantedFees(ctx sdk.Context, granter, grantee sdk.AccAddress, fee sdk.Coins, msgs []sdk.Msg) error
}

type DistrKeeper interface {
	FundCommunityPool(ctx sdk.Context, amount sdk.Coins, sender sdk.AccAddress) error
	GetFeePool(ctx sdk.Context) distributiontypes.FeePool
	GetCommunityTax(ctx sdk.Context) math.LegacyDec
	SetFeePool(ctx sdk.Context, feePool distributiontypes.FeePool)
}
