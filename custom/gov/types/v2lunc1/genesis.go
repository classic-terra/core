package v2lunc1

import (
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// NewGenesisState creates a new genesis state for the governance module
func NewGenesisState(startingProposalID uint64, params Params) *GenesisState {
	return &GenesisState{
		StartingProposalId: startingProposalID,
		Params:             &params,
	}
}

// DefaultGenesisState defines the default governance genesis state
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(
		v1.DefaultStartingProposalID,
		DefaultParams(),
	)
}
