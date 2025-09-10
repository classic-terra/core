//nolint:revive
package v13

import (
	"github.com/classic-terra/core/v3/app/upgrades"
)

const UpgradeName = "v13"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV13UpgradeHandler,
}

// For testing
type LegacyPrefix struct {
	KeySequenceCodeID                              []byte
	KeySequenceInstanceID                          []byte
	CodeKeyPrefix                                  []byte
	ContractKeyPrefix                              []byte
	ContractStorePrefix                            []byte
	ContractCodeHistoryElementPrefix               []byte
	PinnedCodeIndexPrefix                          []byte
	TXCounterPrefix                                []byte
	ContractsByCreatorPrefix                       []byte
	ContractByCodeIDAndCreatedSecondaryIndexPrefix []byte
	ParamsKey                                      []byte
	AbsoluteTxPositionLen                          int
}

// Global ready-to-use instance
var LegacyPrefixes = LegacyPrefix{
	KeySequenceCodeID:                              []byte{0x01},
	KeySequenceInstanceID:                          []byte{0x02},
	CodeKeyPrefix:                                  []byte{0x03},
	ContractKeyPrefix:                              []byte{0x04},
	ContractStorePrefix:                            []byte{0x05},
	ContractCodeHistoryElementPrefix:               []byte{0x06},
	PinnedCodeIndexPrefix:                          []byte{0x07},
	TXCounterPrefix:                                []byte{0x08},
	ContractsByCreatorPrefix:                       []byte{0x09},
	ContractByCodeIDAndCreatedSecondaryIndexPrefix: []byte{0x10},
	ParamsKey:                                      []byte{0x11},
	AbsoluteTxPositionLen:                          16,
}
