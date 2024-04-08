package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
)

func SetupContract(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, keyname string, fileName string, message string) (codeId, contract string) {
	codeId, err := StoreContract(ctx, chain, keyname, fileName)
	require.NoError(t, err)

	contractAddr, err := InstantiateContract(ctx, chain, keyname, codeId, message, true)
	require.NoError(t, err)

	return codeId, contractAddr
}

func ExecuteMsgWithAmount(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, contractAddr, amount, message string) {
	chainNode := chain.FullNodes[0]

	cmd := []string{"terrad", "tx", "wasm", "execute", contractAddr, message,
		"--amount", amount,
	}
	_, err := chainNode.ExecTx(ctx, user.KeyName(), cmd...)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, chain)
	require.NoError(t, err)
}

func ExecuteMsgWithFee(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, contractAddr, amount, feeCoin, message string) {
	chainNode := chain.FullNodes[0]

	cmd := []string{"terrad", "tx", "wasm", "execute", contractAddr, message,
		"--fees", feeCoin,
	}

	if amount != "" {
		cmd = append(cmd, "--amount", amount)
	}

	_, err := chainNode.ExecTx(ctx, user.KeyName(), cmd...)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, chain)
	require.NoError(t, err)
}

// StoreContract takes a file path to smart contract and stores it on-chain. Returns the contracts code id.
func StoreContract(ctx context.Context, chain *cosmos.CosmosChain, keyname string, fileName string) (string, error) {
	_, file := filepath.Split(fileName)
	chainNode := chain.FullNodes[0]
	err := chainNode.CopyFile(ctx, fileName, file)
	if err != nil {
		return "", fmt.Errorf("writing contract file to docker volume: %w", err)
	}

	_, err = chainNode.ExecTx(ctx, keyname, "wasm", "store", path.Join(chainNode.HomeDir(), file), "--gas", "2000000")
	if err != nil {
		return "", fmt.Errorf("store contract: %w", err)
	}

	err = testutil.WaitForBlocks(ctx, 10, chain)
	if err != nil {
		return "", fmt.Errorf("wait for blocks: %w", err)
	}

	stdout, _, err := chainNode.ExecQuery(ctx, "wasm", "list-code", "--reverse")
	if err != nil {
		return "", err
	}

	res := CodeInfosResponse{}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		return "", err
	}

	return res.CodeInfos[0].CodeID, nil
}

// InstantiateContract takes a code id for a smart contract and initialization message and returns the instantiated contract address.
func InstantiateContract(ctx context.Context, chain *cosmos.CosmosChain, keyName string, codeID string, initMessage string, needsNoAdminFlag bool, extraExecTxArgs ...string) (string, error) {
	chainNode := chain.FullNodes[0]

	command := []string{"wasm", "instantiate", codeID, initMessage, "--label", "wasm-contract"}
	command = append(command, extraExecTxArgs...)
	if needsNoAdminFlag {
		command = append(command, "--no-admin")
	}
	_, err := chainNode.ExecTx(ctx, keyName, command...)
	if err != nil {
		return "", err
	}

	stdout, _, err := chainNode.ExecQuery(ctx, "wasm", "list-contract-by-code", codeID)
	if err != nil {
		return "", err
	}

	contactsRes := QueryContractResponse{}
	if err := json.Unmarshal([]byte(stdout), &contactsRes); err != nil {
		return "", err
	}

	contractAddress := contactsRes.Contracts[len(contactsRes.Contracts)-1]
	return contractAddress, nil
}

// QueryContract performs a smart query, taking in a query struct and returning a error with the response struct populated.
func QueryContract(ctx context.Context, chain *cosmos.CosmosChain, contractAddress string, queryMsg any, response any) error {
	chainNode := chain.FullNodes[0]

	query, err := json.Marshal(queryMsg)
	if err != nil {
		return err
	}
	stdout, _, err := chainNode.ExecQuery(ctx, "wasm", "contract-state", "smart", contractAddress, string(query))
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(stdout), response)
	return err
}
