package keeper_test



import (
    "context"
    "testing"
     "fmt"
     sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/classic-terra/core/v3/custom/gov/types/v2lunc1" // Adjust the import path as necessary
    "github.com/stretchr/testify/require"
)

// MockQueryServer is a mock implementation of the v2lunc1.QueryServer interface
type MockQueryServer struct {
    // You can store any state needed for the mock here
    ProposalResponse *v2lunc1.QueryMinimalDepositProposalResponse
    Err              error
}

// ProposalMinimalLUNCByUusd implements the v2lunc1.QueryServer interface
func (m *MockQueryServer) ProposalMinimalLUNCByUusd(ctx context.Context, req *v2lunc1.QueryProposalRequest) (*v2lunc1.QueryMinimalDepositProposalResponse, error) {
    if m.Err != nil {
        return nil, m.Err
    }
    return m.ProposalResponse, nil
}

// Test your keeper logic with the mock
func TestKeeperWithMockQueryServer(t *testing.T) {
mockQueryServer := &MockQueryServer{
	ProposalResponse: &v2lunc1.QueryMinimalDepositProposalResponse{
		//MinimalDeposit: sdk.Coin{sdk.NewCoin("uusd", sdk.NewInt(1000))},
		MinimalDeposit: sdk.NewCoin("uusd", sdk.NewInt(1000)),
	},
}

    // Create your context and other setup logic as needed

    t.Run("successful query", func(t *testing.T) {
        req := &v2lunc1.QueryProposalRequest{ProposalId: 1}
        res, err := mockQueryServer.ProposalMinimalLUNCByUusd(context.Background(), req)
        require.NoError(t, err)
        require.NotNil(t, res)
        require.Equal(t, sdk.NewInt(1000), res.MinimalDeposit.Amount)
    })

    t.Run("error when querying proposal", func(t *testing.T) {
        mockQueryServer.Err = fmt.Errorf("proposal not found")
        req := &v2lunc1.QueryProposalRequest{ProposalId: 1}
        _, err := mockQueryServer.ProposalMinimalLUNCByUusd(context.Background(), req)
        require.Error(t, err)
        require.Equal(t, "proposal not found", err.Error())
    })
}
