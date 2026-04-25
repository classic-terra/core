package legacy

import core "github.com/classic-terra/core/v4/types"

const (
	MainnetUpgradeHeightV1 = int64(13215800) // columbus-5 mainnet upgrade height to v4
	MainnetUpgradeHeightV2 = int64(18303000) // columbus-5 mainnet upgrade height to v8
	TestnetUpgradeHeightV1 = int64(14584970) // rebel-2 testnet upgrade height to v4
	TestnetUpgradeHeightV2 = int64(19354000) // rebel-2 testnet upgrade height to v8
	LegacyUpgradeHeightV1  = int64(0)        // This is not included in the local testing as it would need v3 as a basis
	LegacyUpgradeHeightV2  = int64(70)       // Local testing upgrade height to v8 (using upgrade-test-multi.sh script)

	// MainnetStakingV5Height / TestnetStakingV5Height: heights at which the
	// cosmos-sdk staking v4→v5 migration ran on each chain. That migration
	// backfills the DelegationByValIndexKey (0x71) reverse-index from the
	// primary DelegationKey (0x31). Below these heights there are no entries
	// under 0x71, so the SDK's ValidatorDelegations query returns empty
	// unless we route the read through the primary key.
	//
	// Columbus boundary observed empirically on archive LCDs.
	// Rebel-2 boundary corresponds to the v14 (sdk-53 + ibc-v2) upgrade
	// scheduled by proposal #165 (luncdash.com/governance/165).
	MainnetStakingV5Height = int64(28214400)
	TestnetStakingV5Height = int64(28917279)
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

// IsPreStakingV5 reports whether `blockHeight` falls in the window where the
// cosmos-sdk staking v4→v5 reverse-index (DelegationByValIndexKey, 0x71) had
// not yet been backfilled. ValidatorDelegations queries on these heights must
// fall back to a primary-key (DelegationKey, 0x31) iteration; the indexed path
// returns empty.
func IsPreStakingV5(chainID string, blockHeight int64) bool {
	if blockHeight <= 0 {
		return false
	}
	switch chainID {
	case core.ColumbusChainID:
		return blockHeight < MainnetStakingV5Height
	case core.RebelChainID:
		return TestnetStakingV5Height > 0 && blockHeight < TestnetStakingV5Height
	}
	return false
}

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
