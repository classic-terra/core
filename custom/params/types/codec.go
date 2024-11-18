package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/params/types/proposal"
)

// Wrapper type for backward compatibility
type LegacyParameterChangeProposal struct {
	proposal.ParameterChangeProposal
}

func (l LegacyParameterChangeProposal) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.ParameterChangeProposal)
}

func (l *LegacyParameterChangeProposal) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.ParameterChangeProposal)
}

// RegisterLegacyAminoCodec registers all necessary param module types with a given LegacyAmino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	proposal.RegisterLegacyAminoCodec(cdc)
	cdc.RegisterConcrete(&LegacyParameterChangeProposal{}, "params/ParameterChangeProposal", nil)
}

func init() {
	// govtypes.RegisterProposalTypeCodec(&LegacyParameterChangeProposal{}, "params/ParameterChangeProposal")
}
