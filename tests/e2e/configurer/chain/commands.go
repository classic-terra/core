package chain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/p2p"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	app "github.com/classic-terra/core/v3/app"
	"github.com/classic-terra/core/v3/tests/e2e/initialization"
	"github.com/classic-terra/core/v3/types/assets"
)

func (n *NodeConfig) StoreWasmCode(wasmFile, from string) {
	n.LogActionF("storing wasm code from file %s", wasmFile)
	cmd := []string{"terrad", "tx", "wasm", "store", wasmFile, fmt.Sprintf("--from=%s", from), "--fees=10uluna", "--gas=2000000"}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully stored")
}

func (n *NodeConfig) InstantiateWasmContract(codeID, initMsg, amount, from string, feeDenoms []string, fees ...sdk.Coin) {
	n.LogActionF("instantiating wasm contract %s with %s", codeID, initMsg)
	cmd := []string{"terrad", "tx", "wasm", "instantiate", codeID, initMsg, fmt.Sprintf("--from=%s", from), "--no-admin", "--label=ratelimit"}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	if len(fees) == 0 {
		gasPrices := feeDenomsToGasPrices(feeDenoms)
		cmd = append(cmd, "--gas", "auto", "--gas-adjustment=1.2", fmt.Sprintf("--gas-prices=%s", gasPrices))
	} else {
		feeCoins := sdk.NewCoins(fees...)
		cmd = append(cmd, "--fees", feeCoins.String(), "--gas", "auto", "--gas-adjustment=1.2")
	}
	n.LogActionF(strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)

	require.NoError(n.t, err)

	n.LogActionF("successfully initialized")
}

func (n *NodeConfig) Instantiate2WasmContract(codeID, initMsg, salt, amount, from string, feeDenoms []string) {
	n.LogActionF("instantiating wasm contract %s with %s", codeID, initMsg)
	encodedSalt := make([]byte, hex.EncodedLen(len([]byte(salt))))
	hex.Encode(encodedSalt, []byte(salt))
	cmd := []string{"terrad", "tx", "wasm", "instantiate2", codeID, initMsg, string(encodedSalt), fmt.Sprintf("--from=%s", from), "--no-admin", "--label=ratelimit"}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	gasPrices := feeDenomsToGasPrices(feeDenoms)
	cmd = append(cmd, "--gas", "auto", "--gas-adjustment=1.2", fmt.Sprintf("--gas-prices=%s", gasPrices))

	n.LogActionF(strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully initialized")
}

func (n *NodeConfig) WasmExecute(contract, execMsg, amount, from string, feeDenoms []string) {
	n.LogActionF("executing %s on wasm contract %s from %s", execMsg, contract, from)
	cmd := []string{"terrad", "tx", "wasm", "execute", contract, execMsg, fmt.Sprintf("--from=%s", from)}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	gasPrices := feeDenomsToGasPrices(feeDenoms)
	cmd = append(cmd, "--gas", "auto", "--gas-adjustment=1.2", fmt.Sprintf("--gas-prices=%s", gasPrices))

	n.LogActionF(strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully executed")
}

func (n *NodeConfig) WasmExecuteError(contract, execMsg, amount, from string, feeDenoms []string) {
	n.LogActionF("executing %s on wasm contract %s from %s", execMsg, contract, from)
	cmd := []string{"terrad", "tx", "wasm", "execute", contract, execMsg, fmt.Sprintf("--from=%s", from)}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	gasPrices := feeDenomsToGasPrices(feeDenoms)
	cmd = append(cmd, "--gas", "auto", "--gas-adjustment=1.2", fmt.Sprintf("--gas-prices=%s", gasPrices))

	n.LogActionF(strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmdError(n.t, n.chainID, n.Name, cmd, "", true)
	require.NoError(n.t, err)
	n.LogActionF("executed failed")
}

// QueryParams extracts the params for a given subspace and key. This is done generically via json to avoid having to
// specify the QueryParamResponse type (which may not exist for all params).
func (n *NodeConfig) QueryParams(subspace, key string, result any) {
	cmd := []string{"terrad", "query", "params", "subspace", subspace, key, "--output=json"}

	out, _, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	require.NoError(n.t, err)

	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(n.t, err)
}

func (n *NodeConfig) SubmitParamChangeProposal(proposalJSON, from string) {
	n.LogActionF("submitting param change proposal %s", proposalJSON)
	// ToDo: Is there a better way to do this?
	wd, err := os.Getwd()
	require.NoError(n.t, err)
	localProposalFile := wd + "/scripts/param_change_proposal.json"
	f, err := os.Create(localProposalFile)
	require.NoError(n.t, err)
	_, err = f.WriteString(proposalJSON)
	require.NoError(n.t, err)
	err = f.Close()
	require.NoError(n.t, err)

	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "/terra/param_change_proposal.json", fmt.Sprintf("--from=%s", from)}

	_, _, err = n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)

	err = os.Remove(localProposalFile)
	require.NoError(n.t, err)

	n.LogActionF("successfully submitted param change proposal")
}

func (n *NodeConfig) SubmitAddBurnTaxExemptionAddressProposal(addresses []string, walletName string) int {
	n.LogActionF("submitting add burn tax exemption address proposal %s", addresses)

	cmd := []string{
		"terrad", "tx", "gov", "submit-legacy-proposal",
		"add-burn-tax-exemption-address", strings.Join(addresses, ","),
		"--title=\"burn tax exemption address\"",
		"--description=\"\"burn tax exemption address",
		"--gas", "300000", "--gas-prices", "1uluna",
		fmt.Sprintf("--from=%s", walletName),
	}

	resp, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)

	proposalID, err := extractProposalIDFromResponse(resp.String())
	require.NoError(n.t, err)

	n.LogActionF("successfully submitted add burn tax exemption address proposal")
	return proposalID
}

func (n *NodeConfig) FailIBCTransfer(from, recipient, amount string) {
	n.LogActionF("IBC sending %s from %s to %s", amount, from, recipient)

	cmd := []string{"terrad", "tx", "ibc-transfer", "transfer", "transfer", "channel-0", recipient, amount, fmt.Sprintf("--from=%s", from)}

	_, _, err := n.containerManager.ExecTxCmdWithSuccessString(n.t, n.chainID, n.Name, cmd, "rate limit exceeded")
	require.NoError(n.t, err)

	n.LogActionF("Failed to send IBC transfer (as expected)")
}

func (n *NodeConfig) SendIBCTransfer(from, recipient, amount, memo string) {
	n.LogActionF("IBC sending %s from %s to %s. memo: %s", amount, from, recipient, memo)

	cmd := []string{"terrad", "tx", "ibc-transfer", "transfer", "transfer", "channel-0", recipient, amount, fmt.Sprintf("--from=%s", from), "--memo", memo, "--fees=10uluna"}

	_, _, err := n.containerManager.ExecTxCmdWithSuccessString(n.t, n.chainID, n.Name, cmd, "\"code\":0")
	require.NoError(n.t, err)

	n.LogActionF("successfully submitted sent IBC transfer")
}

func (n *NodeConfig) SubmitTextProposal(text string, initialDeposit sdk.Coin) {
	n.LogActionF("submitting text gov proposal")
	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "--type=text", fmt.Sprintf("--title=\"%s\"", text), "--description=\"test text proposal\"", "--from=val", fmt.Sprintf("--deposit=%s", initialDeposit)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully submitted text gov proposal")
}

func (n *NodeConfig) DepositProposal(proposalNumber int) {
	n.LogActionF("depositing on proposal: %d", proposalNumber)
	deposit := sdk.NewCoin(initialization.TerraDenom, sdk.NewInt(20*assets.MicroUnit)).String()
	cmd := []string{"terrad", "tx", "gov", "deposit", fmt.Sprintf("%d", proposalNumber), deposit, "--from=val", "--gas", "300000", "--fees", "10000000uluna"}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully deposited on proposal %d", proposalNumber)
}

func (n *NodeConfig) VoteYesProposal(from string, proposalNumber int) {
	n.LogActionF("voting yes on proposal: %d", proposalNumber)
	cmd := []string{"terrad", "tx", "gov", "vote", fmt.Sprintf("%d", proposalNumber), "yes", fmt.Sprintf("--from=%s", from), "--gas", "300000", "--fees", "10000000uluna"}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully voted yes on proposal %d", proposalNumber)
}

func (n *NodeConfig) VoteNoProposal(from string, proposalNumber int) {
	n.LogActionF("voting no on proposal: %d", proposalNumber)
	cmd := []string{"terrad", "tx", "gov", "vote", fmt.Sprintf("%d", proposalNumber), "no", fmt.Sprintf("--from=%s", from)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully voted no on proposal: %d", proposalNumber)
}

func AllValsVoteOnProposal(chain *Config, propNumber int) {
	for _, n := range chain.NodeConfigs {
		n.VoteYesProposal(initialization.ValidatorWalletName, propNumber)
	}
}

func extractProposalIDFromResponse(response string) (int, error) {
	// Extract the proposal ID from the response
	startIndex := strings.Index(response, `[{"key":"proposal_id","value":"`) + len(`[{"key":"proposal_id","value":"`)
	endIndex := strings.Index(response[startIndex:], `"`)

	// Extract the proposal ID substring
	proposalIDStr := response[startIndex : startIndex+endIndex]

	// Convert the proposal ID from string to int
	proposalID, err := strconv.Atoi(proposalIDStr)
	if err != nil {
		return 0, err
	}

	return proposalID, nil
}

func (n *NodeConfig) BankSendError(amount string, sendAddress string, receiveAddress string, walletName string, gasLimit string, fees sdk.Coins, failError string) {
	n.LogActionF("bank sending %s from address %s to %s", amount, sendAddress, receiveAddress)
	cmd := []string{"terrad", "tx", "bank", "send", sendAddress, receiveAddress, amount, fmt.Sprintf("--from=%s", walletName)}
	cmd = append(cmd, "--fees", fees.String(), "--gas", gasLimit)
	_, _, err := n.containerManager.ExecTxCmdError(n.t, n.chainID, n.Name, cmd, failError, false)
	require.NoError(n.t, err)
	n.LogActionF("failed sent bank sent %s from address %s to %s", amount, sendAddress, receiveAddress)
}

func (n *NodeConfig) BankSend(amount string, sendAddress string, receiveAddress string, feeDenoms []string, fees ...sdk.Coin) {
	n.BankSendWithWallet(amount, sendAddress, receiveAddress, "val", feeDenoms, fees...)
}

func (n *NodeConfig) BankSendWithWallet(amount string, sendAddress string, receiveAddress string, walletName string, feeDenoms []string, fees ...sdk.Coin) {
	n.LogActionF("bank sending %s from address %s to %s", amount, sendAddress, receiveAddress)
	cmd := []string{"terrad", "tx", "bank", "send", sendAddress, receiveAddress, amount, fmt.Sprintf("--from=%s", walletName)}
	gasPrices := feeDenomsToGasPrices(feeDenoms)
	if len(fees) == 0 {
		cmd = append(cmd, "--gas", "auto", "--gas-adjustment=1.2", fmt.Sprintf("--gas-prices=%s", gasPrices))
	} else {
		feeCoins := sdk.NewCoins(fees...)
		cmd = append(cmd, "--fees", feeCoins.String(), "--gas", "auto", "--gas-adjustment=1.2")
	}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully sent bank sent %s from address %s to %s", amount, sendAddress, receiveAddress)
}

func (n *NodeConfig) BankSendFeeGrantWithWallet(amount string, sendAddress string, receiveAddress string, feeGranter string, walletName string, gasLimit string, fees sdk.Coins) {
	n.LogActionF("bank sending %s from address %s to %s", amount, sendAddress, receiveAddress)
	cmd := []string{"terrad", "tx", "bank", "send", sendAddress, receiveAddress, amount, fmt.Sprintf("--fee-granter=%s", feeGranter), fmt.Sprintf("--from=%s", walletName)}
	cmd = append(cmd, "--fees", fees.String(), "--gas", gasLimit)
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)

	n.LogActionF("successfully sent bank sent %s from address %s to %s", amount, sendAddress, receiveAddress)
}

func (n *NodeConfig) BankMultiSend(amount string, split bool, sendAddress string, feeDenoms []string, receiveAddresses []string) {
	n.LogActionF("bank multisending from %s to %s", sendAddress, strings.Join(receiveAddresses, ","))
	cmd := []string{"terrad", "tx", "bank", "multi-send", sendAddress}
	cmd = append(cmd, receiveAddresses...)
	cmd = append(cmd, amount, "--from=val")

	gasPrices := feeDenomsToGasPrices(feeDenoms)
	cmd = append(cmd, "--gas", "auto", "--gas-adjustment=1.2", fmt.Sprintf("--gas-prices=%s", gasPrices))
	if split {
		cmd = append(cmd, "--split")
	}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully multisent %s to %s", sendAddress, strings.Join(receiveAddresses, ","))
}

func (n *NodeConfig) GrantAddress(granter, gratee string, walletName string, feeDenom string, spendLimit ...string) {
	n.LogActionF("granting for address %s", gratee)
	cmd := []string{"terrad", "tx", "feegrant", "grant", granter, gratee, fmt.Sprintf("--from=%s", walletName), fmt.Sprintf("--spend-limit=%s", strings.Join(spendLimit, ",")), fmt.Sprintf("--fees=10%s", feeDenom)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully granted for address %s", gratee)
}

func (n *NodeConfig) RevokeGrant(granter, gratee string, walletName string, feeDenom string) {
	n.LogActionF("revoking grant for address %s", gratee)
	cmd := []string{"terrad", "tx", "feegrant", "revoke", granter, gratee, fmt.Sprintf("--from=%s", walletName), fmt.Sprintf("--fees=10%s", feeDenom)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully granted for address %s", gratee)
}

func (n *NodeConfig) CreateWallet(walletName string) string {
	n.LogActionF("creating wallet %s", walletName)
	cmd := []string{"terrad", "keys", "add", walletName, "--keyring-backend=test"}
	outBuf, _, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	require.NoError(n.t, err)
	re := regexp.MustCompile("terra1(.{38})")
	walletAddr := fmt.Sprintf("%s\n", re.FindString(outBuf.String()))
	walletAddr = strings.TrimSuffix(walletAddr, "\n")
	n.LogActionF("created wallet %s, waller address - %s", walletName, walletAddr)
	return walletAddr
}

func (n *NodeConfig) GetWallet(walletName string) string {
	n.LogActionF("retrieving wallet %s", walletName)
	cmd := []string{"terrad", "keys", "show", walletName, "--keyring-backend=test"}
	outBuf, _, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	require.NoError(n.t, err)
	re := regexp.MustCompile("terra1(.{38})")
	walletAddr := fmt.Sprintf("%s\n", re.FindString(outBuf.String()))
	walletAddr = strings.TrimSuffix(walletAddr, "\n")
	n.LogActionF("wallet %s found, waller address - %s", walletName, walletAddr)
	return walletAddr
}

type validatorInfo struct {
	Address     bytes.HexBytes
	PubKey      cryptotypes.PubKey
	VotingPower int64
}

// ResultStatus is node's info, same as Tendermint, except that we use our own
// PubKey.
type resultStatus struct {
	NodeInfo      p2p.DefaultNodeInfo
	SyncInfo      coretypes.SyncInfo
	ValidatorInfo validatorInfo
}

func (n *NodeConfig) Status() (resultStatus, error) { //nolint
	cmd := []string{"terrad", "status"}
	_, errBuf, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	if err != nil {
		return resultStatus{}, err
	}

	cfg := app.MakeEncodingConfig()
	legacyAmino := cfg.Amino
	var result resultStatus
	err = legacyAmino.UnmarshalJSON(errBuf.Bytes(), &result)
	fmt.Println("result", result)

	if err != nil {
		return resultStatus{}, err
	}
	return result, nil
}

func feeDenomsToGasPrices(feeDenoms []string) string {
	gasPrices := ""
	for i, feeDenom := range feeDenoms {
		switch feeDenom {
		case initialization.TerraDenom:
			gasPrices += fmt.Sprintf("%s%s", initialization.TerraGasPrice, initialization.TerraDenom)
		case initialization.UsdDenom:
			gasPrices += fmt.Sprintf("%s%s", initialization.UsdGasPrice, initialization.UsdDenom)
		case initialization.EurDenom:
			gasPrices += fmt.Sprintf("%s%s", initialization.EurGasPrice, initialization.EurDenom)
		default:
		}
		if i != len(feeDenoms)-1 {
			gasPrices += ","
		}
	}
	return gasPrices
}
