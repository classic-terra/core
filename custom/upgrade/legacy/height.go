package legacy

import core "github.com/classic-terra/core/v3/types"

const (
	MainnetUpgradeHeightV1 = int64(13215800) // columbus-5 mainnet upgrade height to v4
	MainnetUpgradeHeightV2 = int64(18303000) // columbus-5 mainnet upgrade height to v8
	TestnetUpgradeHeightV1 = int64(14584970) // rebel-2 testnet upgrade height to v4
	TestnetUpgradeHeightV2 = int64(19354000) // rebel-2 testnet upgrade height to v8
	LegacyUpgradeHeightV1  = int64(0)        // This is not included in the local testing as it would need v3 as a basis
	LegacyUpgradeHeightV2  = int64(25)       // Local testing upgrade height to v8 (using upgrade-test-multi.sh script)
)

// LegacyHandlingVersion represents different versions of legacy handling
// V1: Original legacy handling
// V2: New legacy handling with different behavior
// None: No legacy handling needed
type LegacyHandlingVersion int

const (
	LegacyHandlingNone LegacyHandlingVersion = iota
	LegacyHandlingV1
	LegacyHandlingV2
)

// GetLegacyHandling returns the appropriate legacy handling version based on the chain ID and block height
func GetLegacyHandling(chainID string, blockHeight int64) LegacyHandlingVersion {
	if blockHeight == 0 {
		return LegacyHandlingNone
	}

	switch chainID {
	case core.ColumbusChainID:
		if blockHeight < MainnetUpgradeHeightV1 {
			return LegacyHandlingV1
		} else if blockHeight < MainnetUpgradeHeightV2 {
			return LegacyHandlingV2
		}
	case core.RebelChainID:
		if blockHeight < TestnetUpgradeHeightV1 {
			return LegacyHandlingV1
		} else if blockHeight < TestnetUpgradeHeightV2 {
			return LegacyHandlingV2
		}
	case "localterra-legacy":
		if blockHeight < LegacyUpgradeHeightV1 {
			return LegacyHandlingV1
		} else if blockHeight < LegacyUpgradeHeightV2 {
			return LegacyHandlingV2
		}
	default:
		// For local testing or other networks do not use legacy handling
		return LegacyHandlingNone
	}

	return LegacyHandlingNone
}
