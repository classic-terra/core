package types

// NewGenesisState creates a new GenesisState object
func NewGenesisState(params Params, rates []MinCommissionRate) *GenesisState {
	return &GenesisState{
		Params:             params,
		MinCommissionRates: rates,
	}
}

// DefaultGenesisState gets raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	emptySet := []MinCommissionRate{}
	return &GenesisState{
		Params:             DefaultParams(),
		MinCommissionRates: emptySet,
	}
}
