package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/upgrade/types"

	govtypes "github.com/classic-terra/core/v3/custom/gov/types"
)

// Wrapper type for backward compatibility
type LegacyPlan struct {
	types.Plan
}

// Wrapper type for backward compatibility
type LegacySoftwareUpgradeProposal struct {
	types.SoftwareUpgradeProposal
}

// Wrapper type for backward compatibility
type LegacyCancelSoftwareUpgradeProposal struct {
	types.CancelSoftwareUpgradeProposal
}

func (l LegacyPlan) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.Plan)
}

func (l *LegacyPlan) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.Plan)
}

func (l LegacySoftwareUpgradeProposal) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.SoftwareUpgradeProposal)
}

func (l *LegacySoftwareUpgradeProposal) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.SoftwareUpgradeProposal)
}

func (l LegacyCancelSoftwareUpgradeProposal) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.CancelSoftwareUpgradeProposal)
}

func (l *LegacyCancelSoftwareUpgradeProposal) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.CancelSoftwareUpgradeProposal)
}

// RegisterLegacyAminoCodec registers concrete types on the LegacyAmino codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)

	cdc.RegisterConcrete(LegacyPlan{}, "upgrade/Plan", nil)
	cdc.RegisterConcrete(&LegacySoftwareUpgradeProposal{}, "upgrade/SoftwareUpgradeProposal", nil)
	cdc.RegisterConcrete(&LegacyCancelSoftwareUpgradeProposal{}, "upgrade/CancelSoftwareUpgradeProposal", nil)
}

func init() {
	govtypes.RegisterProposalTypeCodec(&LegacySoftwareUpgradeProposal{}, "upgrade/SoftwareUpgradeProposal")
	govtypes.RegisterProposalTypeCodec(&LegacyCancelSoftwareUpgradeProposal{}, "upgrade/CancelSoftwareUpgradeProposal")
}
