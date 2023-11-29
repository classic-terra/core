package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestGenesisValidation(t *testing.T) {

	tests := []struct {
		name     string
		GenState func() *GenesisState
		wantErr  bool
	}{
		{
			name:     "valid state",
			GenState: DefaultGenesisState,
			wantErr:  false,
		},
		{
			name: "invalid BasePool state",
			GenState: func() *GenesisState {
				genState := DefaultGenesisState()
				genState.Params.BasePool = sdk.NewDec(-1)
				return genState
			},
			wantErr: true,
		},
		{
			name: "invalid PoolRecoveryPeriod state",
			GenState: func() *GenesisState {
				genState := DefaultGenesisState()
				genState.Params.PoolRecoveryPeriod = 0
				return genState
			},
			wantErr: true,
		},
		{
			name: "invalid MinStabilitySpread state",
			GenState: func() *GenesisState {
				genState := DefaultGenesisState()
				genState.Params.MinStabilitySpread = sdk.NewDec(-1)
				return genState
			},
			wantErr: true,
		},
		{
			name: "invalid FeeBurnRatio state",
			GenState: func() *GenesisState {
				genState := DefaultGenesisState()
				genState.Params.FeeBurnRatio = sdk.NewDec(-1)
				return genState
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		genState := tt.GenState()
		if err := ValidateGenesis(genState); (err != nil) != tt.wantErr {
			t.Errorf("expected wantErr = %v got %v", tt.wantErr, err != nil)
		}
	}
}
