package types

import (
	"cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"

	govtypes "github.com/classic-terra/core/v3/custom/gov/types"
)

// RegisterLegacyAminoCodec registers concrete types on the LegacyAmino codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(types.Plan{}, "upgrade/Plan", nil)
	cdc.RegisterConcrete(&types.SoftwareUpgradeProposal{}, "upgrade/SoftwareUpgradeProposal", nil)
	cdc.RegisterConcrete(&types.CancelSoftwareUpgradeProposal{}, "upgrade/CancelSoftwareUpgradeProposal", nil)
}

func init() {
	govtypes.RegisterProposalTypeCodec(&types.SoftwareUpgradeProposal{}, "upgrade/SoftwareUpgradeProposal")
	govtypes.RegisterProposalTypeCodec(&types.CancelSoftwareUpgradeProposal{}, "upgrade/CancelSoftwareUpgradeProposal")
}
