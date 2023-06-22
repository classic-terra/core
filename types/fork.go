package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// SwapDisableHeight - make min spread to 100% to disable swap
	SwapDisableHeight = 7_607_790
	// TaxPowerUpgradeHeight is when taxes are allowed to go into effect
	// This will still need a parameter change proposal, but can be activated
	// anytime after this height
	TaxPowerUpgradeHeight = 9_346_889
	// IbcEnableHeight - renable IBC only, block height is approximately December 5th, 2022
	IbcEnableHeight = 10_542_500
	// VersionMapEnableHeight - set the version map to enable software upgrades, approximately February 14, 2023
	VersionMapEnableHeight = 11_543_150
)

func IsBeforeTaxPowerUpgradeHeight(ctx sdk.Context) bool {
	currHeight := ctx.BlockHeight()
	return currHeight < TaxPowerUpgradeHeight
}

func IsAfterPowerUpgradeHeight(ctx sdk.Context) bool {
	currHeight := ctx.BlockHeight()
	return ctx.ChainID() == ColumbusChainID && currHeight >= TaxPowerUpgradeHeight
}
