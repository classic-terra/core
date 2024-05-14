package helpers

import (
	"context"
	"strings"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/stretchr/testify/require"
)

func GetIBCHooksUserAddress(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, channel, uaddr string) string {
	chainNode := chain.FullNodes[0]
	// terrad q ibchooks wasm-sender channel-0 "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au" --node http://localhost:26657
	cmd := []string{"ibchooks", "wasm-sender", channel, uaddr}

	// This query does not return a type, just prints the string.
	stdout, _, err := chainNode.ExecQuery(ctx, cmd...)
	require.NoError(t, err)

	address := strings.Replace(string(stdout), "\n", "", -1)
	return address
}

func GetIBCHookTotalFunds(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, contract string, uaddr string) GetTotalFundsResponse {
	var res GetTotalFundsResponse
	err := QueryContract(ctx, chain, contract, QueryMsg{GetTotalFunds: &GetTotalFundsQuery{Addr: uaddr}}, &res)
	require.NoError(t, err)
	return res
}

func GetIBCHookCount(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, contract string, uaddr string) GetCountResponse {
	var res GetCountResponse
	err := QueryContract(ctx, chain, contract, QueryMsg{GetCount: &GetCountQuery{Addr: uaddr}}, &res)
	require.NoError(t, err)
	return res
}
