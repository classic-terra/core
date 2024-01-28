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

	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/p2p"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	app "github.com/classic-terra/core/v2/app"
	"github.com/classic-terra/core/v2/tests/e2e/initialization"
)

func (n *NodeConfig) StoreWasmCode(wasmFile, from string) {
	n.LogActionF("storing wasm code from file %s", wasmFile)
	cmd := []string{"terrad", "tx", "wasm", "store", wasmFile, fmt.Sprintf("--from=%s", from), "--gas=auto", "--gas-prices=0uluna", "--gas-adjustment=1.3"}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully stored")
}

func (n *NodeConfig) InstantiateWasmContract(codeId, initMsg, amount, from string) {
	n.LogActionF("instantiating wasm contract %s with %s", codeId, initMsg)
	cmd := []string{"terrad", "tx", "wasm", "instantiate", codeId, initMsg, fmt.Sprintf("--from=%s", from), "--no-admin", "--label=ratelimit", "--gas=auto", "--gas-prices=0.0uluna", "--gas-adjustment=1.3"}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	n.LogActionF(strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)

	require.NoError(n.t, err)

	n.LogActionF("successfully initialized")
}

func (n *NodeConfig) Instantiate2WasmContract(codeId, initMsg, salt, amount, fee, from string) {
	n.LogActionF("instantiating wasm contract %s with %s", codeId, initMsg)
	encodedSalt := make([]byte, hex.EncodedLen(len([]byte(salt))))
	hex.Encode(encodedSalt, []byte(salt))
	cmd := []string{"terrad", "tx", "wasm", "instantiate2", codeId, initMsg, string(encodedSalt), fmt.Sprintf("--from=%s", from), "--no-admin", "--label=ratelimit", "--gas=auto", "--gas-prices=0.0uluna", "--gas-adjustment=1.3"}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	if fee != "" {
		cmd = append(cmd, fmt.Sprintf("--fees=%s", fee))
	}
	n.LogActionF(strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully initialized")
}

func (n *NodeConfig) WasmExecute(contract, execMsg, amount, fee, from string) {
	n.LogActionF("executing %s on wasm contract %s from %s", execMsg, contract, from)
	cmd := []string{"terrad", "tx", "wasm", "execute", contract, execMsg, fmt.Sprintf("--from=%s", from), "--gas=auto", "--gas-prices=0.0uluna", "--gas-adjustment=1.3"}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	if fee != "" {
		cmd = append(cmd, fmt.Sprintf("--fees=%s", fee))
	}
	n.LogActionF(strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully executed")
}

// QueryParams extracts the params for a given subspace and key. This is done generically via json to avoid having to
// specify the QueryParamResponse type (which may not exist for all params).
func (n *NodeConfig) QueryParams(subspace, key string, result any) {
	cmd := []string{"terrad", "query", "params", "subspace", subspace, key, "--output=json"}

	out, _, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "")
	require.NoError(n.t, err)

	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(n.t, err)
}

func (n *NodeConfig) SubmitParamChangeProposal(proposalJson, from string) {
	n.LogActionF("submitting param change proposal %s", proposalJson)
	// ToDo: Is there a better way to do this?
	wd, err := os.Getwd()
	require.NoError(n.t, err)
	localProposalFile := wd + "/scripts/param_change_proposal.json"
	f, err := os.Create(localProposalFile)
	require.NoError(n.t, err)
	_, err = f.WriteString(proposalJson)
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
		fmt.Sprintf("--from=%s", walletName),
	}

	resp, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)

	proposalId, err := extractProposalIdFromResponse(resp.String())
	require.NoError(n.t, err)

	n.LogActionF("successfully submitted add burn tax exemption address proposal")
	return proposalId
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

	cmd := []string{"terrad", "tx", "ibc-transfer", "transfer", "transfer", "channel-0", recipient, amount, fmt.Sprintf("--from=%s", from), "--memo", memo}

	_, _, err := n.containerManager.ExecTxCmdWithSuccessString(n.t, n.chainID, n.Name, cmd, "code: 0")
	require.NoError(n.t, err)

	n.LogActionF("successfully submitted sent IBC transfer")
}

func (n *NodeConfig) SubmitUpgradeProposal(upgradeVersion string, upgradeHeight int64, initialDeposit sdk.Coin) {
	n.LogActionF("submitting upgrade proposal %s for height %d", upgradeVersion, upgradeHeight)
	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "software-upgrade", upgradeVersion, fmt.Sprintf("--title=\"%s upgrade\"", upgradeVersion), "--description=\"upgrade proposal submission\"", fmt.Sprintf("--upgrade-height=%d", upgradeHeight), "--upgrade-info=\"\"", "--from=val", fmt.Sprintf("--deposit=%s", initialDeposit)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully submitted upgrade proposal")
}

func (n *NodeConfig) SubmitTextProposal(text string, initialDeposit sdk.Coin, isExpedited bool) {
	n.LogActionF("submitting text gov proposal")
	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "--type=text", fmt.Sprintf("--title=\"%s\"", text), "--description=\"test text proposal\"", "--from=val", fmt.Sprintf("--deposit=%s", initialDeposit)}
	if isExpedited {
		cmd = append(cmd, "--is-expedited=true")
	}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully submitted text gov proposal")
}

func (n *NodeConfig) DepositProposal(proposalNumber int, isExpedited bool) {
	n.LogActionF("depositing on proposal: %d", proposalNumber)
	deposit := sdk.NewCoin("luna", sdk.NewInt(20)).String()
	// if isExpedited {
	// 	deposit = sdk.NewCoin("luna", sdk.NewInt(config.MinExpeditedDepositValue)).String()
	// }
	cmd := []string{"terrad", "tx", "gov", "deposit", fmt.Sprintf("%d", proposalNumber), deposit, "--from=val"}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully deposited on proposal %d", proposalNumber)
}

func (n *NodeConfig) VoteYesProposal(from string, proposalNumber int) {
	n.LogActionF("voting yes on proposal: %d", proposalNumber)
	cmd := []string{"terrad", "tx", "gov", "vote", fmt.Sprintf("%d", proposalNumber), "yes", fmt.Sprintf("--from=%s", from)}
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

func extractProposalIdFromResponse(response string) (int, error) {
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

func (n *NodeConfig) BankSend(amount string, sendAddress string, receiveAddress string) {
	n.BankSendWithWallet(amount, sendAddress, receiveAddress, "val")
}

func (n *NodeConfig) BankSendWithWallet(amount string, sendAddress string, receiveAddress string, walletName string) {
	n.LogActionF("bank sending %s from address %s to %s", amount, sendAddress, receiveAddress)
	cmd := []string{"terrad", "tx", "bank", "send", sendAddress, receiveAddress, amount, fmt.Sprintf("--from=%s", walletName)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully sent bank sent %s from address %s to %s", amount, sendAddress, receiveAddress)
}

func (n *NodeConfig) BankSendFeeGrantWithWallet(amount string, sendAddress string, receiveAddress string, feeGranter string, walletName string) {
	n.LogActionF("bank sending %s from address %s to %s", amount, sendAddress, receiveAddress)
	cmd := []string{"terrad", "tx", "bank", "send", sendAddress, receiveAddress, amount, fmt.Sprintf("--fee-granter=%s", feeGranter), fmt.Sprintf("--from=%s", walletName)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)

	n.LogActionF("successfully sent bank sent %s from address %s to %s", amount, sendAddress, receiveAddress)
}

func (n *NodeConfig) BankMultiSend(amount string, split bool, sendAddress string, receiveAddresses ...string) {
	n.LogActionF("bank multisending from %s to %s", sendAddress, strings.Join(receiveAddresses, ","))
	cmd := []string{"terrad", "tx", "bank", "multi-send", sendAddress}
	cmd = append(cmd, receiveAddresses...)
	cmd = append(cmd, amount, "--from=val")
	if split {
		cmd = append(cmd, "--split")
	}

	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully multisent %s to %s", sendAddress, strings.Join(receiveAddresses, ","))
}

func (n *NodeConfig) GrantAddress(granter, gratee string, spendLimit string, walletName string) {
	n.LogActionF("granting for address %s", gratee)
	cmd := []string{"terrad", "tx", "feegrant", "grant", granter, gratee, fmt.Sprintf("--from=%s", walletName), fmt.Sprintf("--spend-limit=%s", spendLimit)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully granted for address %s", gratee)
}

func (n *NodeConfig) CreateWallet(walletName string) string {
	n.LogActionF("creating wallet %s", walletName)
	cmd := []string{"terrad", "keys", "add", walletName, "--keyring-backend=test"}
	outBuf, _, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "")
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
	outBuf, _, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "")
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

func (n *NodeConfig) Status() (resultStatus, error) {
	cmd := []string{"terrad", "status"}
	_, errBuf, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "")
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
