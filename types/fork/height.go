package fork

// SwapDisableForkHeight - make min spread to 100% to disable swap
const (
	// SwapDisableForkHeight - make min spread to 100% to disable swap, block height is approximately June 1st, 2022.  A highly regrettable decision.
	SwapDisableForkHeight = 7607790
	// SwapEnableForkHeight - renable IBC only, block height is approximately December 5th, 2022
	IBCEnableForkHeight = 10542500
	// VersionMapEnableHeight - set the version map to enable software upgrades, approximately February 14, 2023
	VersionMapEnableHeight = 11543150
	// JusticeAndTruthHeight - Re-enable swaps, mock Sam Bankman-Fried, pursue decentralized money.
	JusticeAndTruthHeight = 13469420
)
