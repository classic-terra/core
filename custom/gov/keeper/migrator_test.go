package keeper_test

import (
	"testing"

	// "github.com/cosmos/cosmos-sdk/x/gov/exported"
	sdk "github.com/cosmos/cosmos-sdk/types"
	// "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/classic-terra/core/v3/custom/gov/keeper"

	"github.com/stretchr/testify/require"
)

// MockParamSubspace is a mock implementation of ParamSubspace for testing.
type MockParamSubspace struct{}

// Satisfy the ParamSubspace interface by adding dummy methods.
func (m MockParamSubspace) Get(_ sdk.Context, _ []byte, _ interface{}) {}

// TestNewMigrator tests the NewMigrator constructor
func TestNewMigrator(t *testing.T) {
	// Create a keeper
	mockKeeper := &keeper.Keeper{}
	// mockKeeper := &keeper.Keeper{}
	t.Logf("mockKeeper: %v", mockKeeper)

	// Use the mock ParamSubspace
	mockSubspace := MockParamSubspace{}
	t.Logf("mockSubspace: %v", mockSubspace)

	migrator := keeper.NewMigrator(mockKeeper, mockSubspace)
	require.NotNil(t, migrator)
}

