package legacy

import core "github.com/classic-terra/core/v3/types"

const (
	MainnetUpgradeHeight  = int64(18303000) // columbus-5 mainnet upgrade height to v8
	TestnetUpgradeHeight  = int64(19354000) // rebel-2 testnet upgrade height to v8
	LegacyUpgradeHeight   = int64(25)       // Local testing upgrade height to v8 (using update-test-multi.sh script)
	LocalnetUpgradeHeight = int64(0)        // Local testing or automated workflows without pre-v8 data
)

// GetUpgradeHeight returns the appropriate upgrade height based on the chain ID
func GetUpgradeHeight(chainID string) int64 {
	switch chainID {
	case core.ColumbusChainID:
		return MainnetUpgradeHeight
	case core.RebelChainID:
		return TestnetUpgradeHeight
	case "localterra-legacy":
		return LegacyUpgradeHeight
	default:
		// For local testing or other networks
		return LocalnetUpgradeHeight
	}
}
