package types

import (
	govtypes "github.com/classic-terra/core/v3/custom/gov/types/v2custom"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/params/types/proposal"
)

// RegisterLegacyAminoCodec registers all necessary param module types with a given LegacyAmino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&proposal.ParameterChangeProposal{}, "params/ParameterChangeProposal", nil)
}

func init() {
	govtypes.RegisterProposalTypeCodec(&proposal.ParameterChangeProposal{}, "params/ParameterChangeProposal")
}
