package legacy

import core "github.com/classic-terra/core/v3/types"

const (
	MainnetUpgradeHeight  = int64(18303000) // columbus-5 mainnet upgrade height to v8
	TestnetUpgradeHeight  = int64(19354000) // rebel-2 testnet upgrade height to v8
	LocalnetUpgradeHeight = int64(0)       // Local testing upgrade height to v8 (using update-test-multi.sh script)
)

// GetUpgradeHeight returns the appropriate upgrade height based on the chain ID
func GetUpgradeHeight(chainID string) int64 {
	switch chainID {
	case core.ColumbusChainID:
		return MainnetUpgradeHeight
	case core.RebelChainID:
		return TestnetUpgradeHeight
	default:
		// For local testing or other networks
		return LocalnetUpgradeHeight
	}
}
