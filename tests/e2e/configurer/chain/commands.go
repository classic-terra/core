package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	app "github.com/classic-terra/core/v3/app"
	"github.com/classic-terra/core/v3/tests/e2e/initialization"
	"github.com/classic-terra/core/v3/types/assets"
	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/p2p"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
)

// extractTxHashFromJSON tries to parse a Cosmos SDK tx response JSON and return the txhash field.
// Returns an empty string if not found or parsing fails.
func extractTxHashFromJSON(payload []byte) string {
	var resp struct {
		TxHash string `json:"txhash"`
	}
	if err := json.Unmarshal(payload, &resp); err == nil && resp.TxHash != "" {
		return resp.TxHash
	}
	return ""
}

// GetModuleAccountAddress returns the account address for a given module name (e.g., "market").
// It queries `terrad query auth module-accounts --output=json` and scans for the matching ModuleAccount name.
func (n *NodeConfig) GetModuleAccountAddress(moduleName string) string {
	cmd := []string{"terrad", "query", "auth", "module-accounts", "--output=json"}
	outBuf, errBuf, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	require.NoErrorf(n.t, err, "failed to query module accounts: stdout=%q stderr=%q", strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()))
	var resp struct {
		Accounts []struct {
			Name        string `json:"name"`
			BaseAccount struct {
				Address string `json:"address"`
			} `json:"base_account"`
		} `json:"accounts"`
	}
	require.NoErrorf(n.t, json.Unmarshal(outBuf.Bytes(), &resp), "failed to decode module accounts json: %q", strings.TrimSpace(outBuf.String()))
	for _, acc := range resp.Accounts {
		if acc.Name == moduleName {
			if acc.BaseAccount.Address != "" {
				return acc.BaseAccount.Address
			}
		}
	}
	require.Failf(n.t, "module account not found", "module %s not found in module-accounts", moduleName)
	return ""
}

func (n *NodeConfig) StoreWasmCode(wasmFile, from string) {
	n.LogActionF("storing wasm code from file %s", wasmFile)
	cmd := []string{"terrad", "tx", "wasm", "store", wasmFile, fmt.Sprintf("--from=%s", from)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully stored")
}

// DelegateOracleFeedConsent sets the feeder address for this validator to the provided account address.
func (n *NodeConfig) DelegateOracleFeedConsent(feeder string) {
	if !n.IsValidator {
		n.LogActionF("skipping feeder delegation: node is not a validator")
		return
	}
	if n.OperatorAddress == "" {
		_ = n.extractOperatorAddressIfValidator()
	}
	require.NotEmpty(n.t, n.OperatorAddress, "validator operator address must be known before delegating feeder consent")
	n.LogActionF("delegating oracle feed consent: validator=%s feeder=%s", n.OperatorAddress, feeder)
	// terrad tx oracle set-feeder [feeder]
	cmd := []string{"terrad", "tx", "oracle", "set-feeder", feeder, fmt.Sprintf("--from=%s", initialization.ValidatorWalletName)}
	outBuf, errBuf, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err, "feeder delegation tx failed: stderr=%s stdout=%s", strings.TrimSpace(errBuf.String()), strings.TrimSpace(outBuf.String()))
	// Verify via query with retries until feeder mapping matches expected
	var lastOut, lastErr string
	for i := 0; i < 20; i++ {
		q := []string{"terrad", "query", "oracle", "feeder", n.OperatorAddress, "--output=json"}
		qOut, qErr, qE := n.containerManager.ExecCmd(n.t, n.Name, q, "", false)
		lastOut, lastErr = strings.TrimSpace(qOut.String()), strings.TrimSpace(qErr.String())
		if qE == nil && lastOut != "" {
			var resp struct {
				FeederAddr string `json:"feeder_addr"`
			}
			if err := json.Unmarshal(qOut.Bytes(), &resp); err == nil {
				n.LogActionF("feeder query: operator=%s feeder=%s (expected=%s)", n.OperatorAddress, resp.FeederAddr, feeder)
				if resp.FeederAddr == feeder {
					break
				}
			} else {
				n.LogActionF("failed to decode feeder query json; stdout=%q stderr=%q", lastOut, lastErr)
			}
		} else if qE != nil {
			n.LogActionF("feeder query failed; stdout=%q stderr=%q err=%v", lastOut, lastErr, qE)
		}
		time.Sleep(200 * time.Millisecond)
	}
	require.Containsf(n.t, lastOut, feeder, "feeder mapping for %s did not match expected feeder after delegation; stdout=%s stderr=%s", n.OperatorAddress, lastOut, lastErr)
}

func (n *NodeConfig) InstantiateWasmContract(codeID, initMsg, amount, from string) {
	n.LogActionF("instantiating wasm contract %s with %s", codeID, initMsg)
	cmd := []string{"terrad", "tx", "wasm", "instantiate", codeID, initMsg, fmt.Sprintf("--from=%s", from), "--no-admin", "--label=ratelimit"}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	n.LogActionF("%s", strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully initialized")
}

func (n *NodeConfig) Instantiate2WasmContract(codeID, initMsg, salt, amount, fee, gas, from string) {
	n.LogActionF("instantiating wasm contract %s with %s", codeID, initMsg)
	encodedSalt := make([]byte, hex.EncodedLen(len([]byte(salt))))
	hex.Encode(encodedSalt, []byte(salt))
	cmd := []string{"terrad", "tx", "wasm", "instantiate2", codeID, initMsg, string(encodedSalt), fmt.Sprintf("--from=%s", from), "--no-admin", "--label=ratelimit"}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	if fee != "" {
		cmd = append(cmd, fmt.Sprintf("--fees=%s", fee))
	}
	if gas != "" {
		cmd = append(cmd, fmt.Sprintf("--gas=%s", gas))
	}
	n.LogActionF("%s", strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully initialized")
}

func (n *NodeConfig) WasmExecute(contract, execMsg, amount, fee, from string) {
	n.LogActionF("executing %s on wasm contract %s from %s", execMsg, contract, from)
	cmd := []string{"terrad", "tx", "wasm", "execute", contract, execMsg, fmt.Sprintf("--from=%s", from)}
	if amount != "" {
		cmd = append(cmd, fmt.Sprintf("--amount=%s", amount))
	}
	if fee != "" {
		cmd = append(cmd, fmt.Sprintf("--fees=%s", fee))
	}
	n.LogActionF("%s", strings.Join(cmd, " "))
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully executed")
}

// QueryOracleVotePeriod queries on-chain oracle params and returns the vote period as an int64.
func (n *NodeConfig) QueryOracleVotePeriod() int64 {
	cmd := []string{"terrad", "query", "oracle", "params", "--output=json"}
	out, _, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	require.NoError(n.t, err)
	var resp struct {
		Params struct {
			VotePeriod string `json:"vote_period"`
		} `json:"params"`
	}
	require.NoError(n.t, json.Unmarshal(out.Bytes(), &resp))
	vp, err := strconv.ParseInt(resp.Params.VotePeriod, 10, 64)
	require.NoError(n.t, err)
	return vp
}

// QueryOracleExchangeRates queries all current oracle exchange rates and returns a map denom->amount (as string decimal).
// It uses `terrad query oracle exchange-rates --output=json` and parses the DecCoins response.
func (n *NodeConfig) QueryOracleExchangeRates() map[string]string {
	cmd := []string{"terrad", "query", "oracle", "exchange-rates", "--output=json"}
	outBuf, errBuf, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	if err != nil {
		n.LogActionF("failed to query exchange rates: %v; stdout=%q stderr=%q", err, strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()))
		return map[string]string{}
	}
	var resp struct {
		ExchangeRates []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"exchange_rates"`
	}
	if jerr := json.Unmarshal(outBuf.Bytes(), &resp); jerr != nil {
		n.LogActionF("failed to decode exchange rates json: %v; payload=%q", jerr, strings.TrimSpace(outBuf.String()))
		return map[string]string{}
	}
	m := make(map[string]string, len(resp.ExchangeRates))
	for _, dc := range resp.ExchangeRates {
		m[dc.Denom] = dc.Amount
	}
	return m
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

func (n *NodeConfig) SubmitAddBurnTaxExemptionAddressProposalV1(addresses []string, walletName string) int {
	n.LogActionF("submitting add burn tax exemption address proposal (v1 JSON) %s", addresses)
	deposit := sdk.NewCoin(initialization.TerraDenom, sdkmath.NewInt(20*assets.MicroUnit)).String()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	proposal := map[string]any{
		"messages": []any{
			map[string]any{
				"@type": "/cosmos.gov.v1.MsgExecLegacyContent",
				"content": map[string]any{
					"@type":       "/terra.treasury.v1beta1.AddBurnTaxExemptionAddressProposal",
					"title":       "burn tax exemption address",
					"description": "burn tax exemption address",
					"addresses":   addresses,
				},
				"authority": authority,
			},
		},
		"metadata": "",
		"deposit":  deposit,
		"title":    "burn tax exemption address",
		"summary":  "burn tax exemption address",
	}
	bz, err := json.Marshal(proposal)
	require.NoError(n.t, err)
	wd, err := os.Getwd()
	require.NoError(n.t, err)
	localProposalFile := wd + "/scripts/taxexemption_proposal.json"
	f, err := os.Create(localProposalFile)
	require.NoError(n.t, err)
	_, err = f.Write(bz)
	require.NoError(n.t, err)
	require.NoError(n.t, f.Close())

	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "/terra/taxexemption_proposal.json", fmt.Sprintf("--from=%s", walletName)}
	resp, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	proposalID, err := extractProposalIDFromResponse(resp.String())
	require.NoError(n.t, err)
	_ = os.Remove(localProposalFile)
	n.LogActionF("successfully submitted add burn tax exemption address proposal (v1 JSON)")
	return proposalID
}

func (n *NodeConfig) SubmitAddTaxExemptionZoneProposal(zone string, addresses []string, exemptIncoming bool, exemptOutgoing bool, exemptCrossZone bool, walletName string) int {
	n.LogActionF("submitting add tax exemption zone proposal: zone=%s addresses=%s incoming=%t outgoing=%t cross=%t", zone, strings.Join(addresses, ","), exemptIncoming, exemptOutgoing, exemptCrossZone)
	deposit := sdk.NewCoin(initialization.TerraDenom, sdkmath.NewInt(20*assets.MicroUnit)).String()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	proposal := map[string]any{
		"messages": []any{
			map[string]any{
				"@type":      "/terra.taxexemption.v1.MsgAddTaxExemptionZone",
				"zone":       zone,
				"outgoing":   exemptOutgoing,
				"incoming":   exemptIncoming,
				"cross_zone": exemptCrossZone,
				"addresses":  addresses,
				"authority":  authority,
			},
		},
		"metadata": "",
		"deposit":  deposit,
		"title":    "add tax exemption zone",
		"summary":  "add tax exemption zone",
	}
	bz, err := json.Marshal(proposal)
	require.NoError(n.t, err)
	wd, err := os.Getwd()
	require.NoError(n.t, err)
	localProposalFile := wd + "/scripts/taxexemption_proposal.json"
	f, err := os.Create(localProposalFile)
	require.NoError(n.t, err)
	_, err = f.Write(bz)
	require.NoError(n.t, err)
	require.NoError(n.t, f.Close())

	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "/terra/taxexemption_proposal.json", fmt.Sprintf("--from=%s", walletName)}
	resp, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	proposalID, err := extractProposalIDFromResponse(resp.String())
	require.NoError(n.t, err)
	_ = os.Remove(localProposalFile)
	n.LogActionF("successfully submitted add tax exemption zone proposal")
	return proposalID
}

func (n *NodeConfig) SubmitModifyTaxExemptionZoneProposal(zone string, exemptIncoming bool, exemptOutgoing bool, exemptCrossZone bool, walletName string) int {
	n.LogActionF("submitting modify tax exemption zone proposal: zone=%s incoming=%t outgoing=%t cross=%t", zone, exemptIncoming, exemptOutgoing, exemptCrossZone)
	deposit := sdk.NewCoin(initialization.TerraDenom, sdkmath.NewInt(20*assets.MicroUnit)).String()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	proposal := map[string]any{
		"messages": []any{
			map[string]any{
				"@type":      "/terra.taxexemption.v1.MsgModifyTaxExemptionZone",
				"zone":       zone,
				"outgoing":   exemptOutgoing,
				"incoming":   exemptIncoming,
				"cross_zone": exemptCrossZone,
				"authority":  authority,
			},
		},
		"metadata": "",
		"deposit":  deposit,
		"title":    "modify tax exemption zone",
		"summary":  "modify tax exemption zone",
	}
	bz, err := json.Marshal(proposal)
	require.NoError(n.t, err)
	wd, err := os.Getwd()
	require.NoError(n.t, err)
	localProposalFile := wd + "/scripts/taxexemption_proposal.json"
	f, err := os.Create(localProposalFile)
	require.NoError(n.t, err)
	_, err = f.Write(bz)
	require.NoError(n.t, err)
	require.NoError(n.t, f.Close())

	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "/terra/taxexemption_proposal.json", fmt.Sprintf("--from=%s", walletName)}
	resp, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	proposalID, err := extractProposalIDFromResponse(resp.String())
	require.NoError(n.t, err)
	_ = os.Remove(localProposalFile)
	n.LogActionF("successfully submitted modify tax exemption zone proposal")
	return proposalID
}

func (n *NodeConfig) SubmitRemoveTaxExemptionZoneProposal(zone string, walletName string) int {
	n.LogActionF("submitting remove tax exemption zone proposal: zone=%s", zone)
	deposit := sdk.NewCoin(initialization.TerraDenom, sdkmath.NewInt(20*assets.MicroUnit)).String()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	proposal := map[string]any{
		"messages": []any{
			map[string]any{
				"@type":     "/terra.taxexemption.v1.MsgRemoveTaxExemptionZone",
				"zone":      zone,
				"authority": authority,
			},
		},
		"metadata": "",
		"deposit":  deposit,
		"title":    "remove tax exemption zone",
		"summary":  "remove tax exemption zone",
	}
	bz, err := json.Marshal(proposal)
	require.NoError(n.t, err)
	wd, err := os.Getwd()
	require.NoError(n.t, err)
	localProposalFile := wd + "/scripts/taxexemption_proposal.json"
	f, err := os.Create(localProposalFile)
	require.NoError(n.t, err)
	_, err = f.Write(bz)
	require.NoError(n.t, err)
	require.NoError(n.t, f.Close())

	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "/terra/taxexemption_proposal.json", fmt.Sprintf("--from=%s", walletName)}
	resp, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	proposalID, err := extractProposalIDFromResponse(resp.String())
	require.NoError(n.t, err)
	_ = os.Remove(localProposalFile)
	n.LogActionF("successfully submitted remove tax exemption zone proposal")
	return proposalID
}

func (n *NodeConfig) SubmitAddTaxExemptionAddressProposal(zone string, addresses []string, walletName string) int {
	n.LogActionF("submitting add tax exemption address proposal: zone=%s addresses=%s", zone, strings.Join(addresses, ","))
	deposit := sdk.NewCoin(initialization.TerraDenom, sdkmath.NewInt(20*assets.MicroUnit)).String()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	proposal := map[string]any{
		"messages": []any{
			map[string]any{
				"@type":     "/terra.taxexemption.v1.MsgAddTaxExemptionAddress",
				"zone":      zone,
				"addresses": addresses,
				"authority": authority,
			},
		},
		"metadata": "",
		"deposit":  deposit,
		"title":    "add tax exemption address",
		"summary":  "add tax exemption address",
	}
	bz, err := json.Marshal(proposal)
	require.NoError(n.t, err)
	wd, err := os.Getwd()
	require.NoError(n.t, err)
	localProposalFile := wd + "/scripts/taxexemption_proposal.json"
	f, err := os.Create(localProposalFile)
	require.NoError(n.t, err)
	_, err = f.Write(bz)
	require.NoError(n.t, err)
	require.NoError(n.t, f.Close())

	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "/terra/taxexemption_proposal.json", fmt.Sprintf("--from=%s", walletName)}
	resp, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	proposalID, err := extractProposalIDFromResponse(resp.String())
	require.NoError(n.t, err)
	_ = os.Remove(localProposalFile)
	n.LogActionF("successfully submitted add tax exemption address proposal")
	return proposalID
}

func (n *NodeConfig) SubmitRemoveTaxExemptionAddressProposal(zone string, addresses []string, walletName string) int {
	n.LogActionF("submitting remove tax exemption address proposal: zone=%s addresses=%s", zone, strings.Join(addresses, ","))
	deposit := sdk.NewCoin(initialization.TerraDenom, sdkmath.NewInt(20*assets.MicroUnit)).String()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	proposal := map[string]any{
		"messages": []any{
			map[string]any{
				"@type":     "/terra.taxexemption.v1.MsgRemoveTaxExemptionAddress",
				"zone":      zone,
				"addresses": addresses,
				"authority": authority,
			},
		},
		"metadata": "",
		"deposit":  deposit,
		"title":    "remove tax exemption address",
		"summary":  "remove tax exemption address",
	}
	bz, err := json.Marshal(proposal)
	require.NoError(n.t, err)
	wd, err := os.Getwd()
	require.NoError(n.t, err)
	localProposalFile := wd + "/scripts/taxexemption_proposal.json"
	f, err := os.Create(localProposalFile)
	require.NoError(n.t, err)
	_, err = f.Write(bz)
	require.NoError(n.t, err)
	require.NoError(n.t, f.Close())

	cmd := []string{"terrad", "tx", "gov", "submit-proposal", "/terra/taxexemption_proposal.json", fmt.Sprintf("--from=%s", walletName)}
	resp, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	proposalID, err := extractProposalIDFromResponse(resp.String())
	require.NoError(n.t, err)
	_ = os.Remove(localProposalFile)
	n.LogActionF("successfully submitted remove tax exemption address proposal")
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

	cmd := []string{"terrad", "tx", "ibc-transfer", "transfer", "transfer", "channel-0", recipient, amount, fmt.Sprintf("--from=%s", from), "--memo", memo}

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
	deposit := sdk.NewCoin(initialization.TerraDenom, sdkmath.NewInt(20*assets.MicroUnit)).String()
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
	n.LogActionF("bank multisend %s to %s", sendAddress, strings.Join(receiveAddresses, ","))
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

func (n *NodeConfig) MarketSwap(offerCoin string, askDenom string, walletName string) {
	n.LogActionF("market swap %s -> %s from %s", offerCoin, askDenom, walletName)
	cmd := []string{"terrad", "tx", "market", "swap", offerCoin, askDenom, fmt.Sprintf("--from=%s", walletName)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully swapped %s to %s", offerCoin, askDenom)
}

func (n *NodeConfig) GrantAddress(granter, gratee string, spendLimit string, walletName string) {
	n.LogActionF("granting for address %s", gratee)
	cmd := []string{"terrad", "tx", "feegrant", "grant", granter, gratee, fmt.Sprintf("--from=%s", walletName), fmt.Sprintf("--spend-limit=%s", spendLimit)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully granted for address %s", gratee)
}

func (n *NodeConfig) GrantBankSend(gratee string, spendLimit string, walletName string) {
	n.LogActionF("granting for address %s", gratee)
	cmd := []string{"terrad", "tx", "authz", "grant", gratee, "send", fmt.Sprintf("--from=%s", walletName), fmt.Sprintf("--spend-limit=%s", spendLimit)}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully granted bank send for address %s", gratee)
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

func (n *NodeConfig) Status() (resultStatus, error) {
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

func (n *NodeConfig) SubmitOracleAggregatePrevote(salt string, amount string) {
	// Only validators should submit oracle prevotes.
	if !n.IsValidator {
		n.LogActionF("skipping oracle aggregate prevote: node is not a validator")
		return
	}
	if n.OperatorAddress == "" {
		// Best-effort resolve now to avoid CLI inference assigning to the wrong validator
		_ = n.extractOperatorAddressIfValidator()
	}
	require.NotEmpty(n.t, n.OperatorAddress, "validator operator address must be known before submitting oracle prevote")
	// Compute expected hash only for logging; CLI expects [salt, exchange-rates, validator] and computes the hash internally
	preimage := fmt.Sprintf("%s:%s:%s", salt, amount, n.OperatorAddress)
	sum := sha256.Sum256([]byte(preimage))
	hash := hex.EncodeToString(sum[:])
	n.LogActionF("submitting oracle aggregate prevote for %s (salt=%s rates=%s hash=%s)", n.OperatorAddress, salt, amount, hash)
	// IMPORTANT: positional args must come BEFORE flags for cobra to parse them; pass validator before --from
	cmd := []string{"terrad", "tx", "oracle", "aggregate-prevote", salt, amount, n.OperatorAddress, fmt.Sprintf("--from=%s", initialization.ValidatorWalletName)}
	outBuf, errBuf, err := n.containerManager.ExecTxCmd(n.t, n.chainID, n.Name, cmd)
	require.NoError(n.t, err)
	// Try to log txhash for correlation
	if txh := extractTxHashFromJSON(outBuf.Bytes()); txh != "" {
		n.LogActionF("prevote txhash=%s", txh)
	} else if txh := extractTxHashFromJSON(errBuf.Bytes()); txh != "" {
		n.LogActionF("prevote txhash=%s (stderr)", txh)
	}
	// After success, confirm prevote exists and log submit block for traceability
	if hash2, sb, ok := n.GetOracleAggregatePrevote(); ok {
		n.LogActionF("submitted prevote ok: hash=%s submit_block=%d", hash2, sb)
	} else {
		n.LogActionF("prevote not found immediately after submit; stdout=%q stderr=%q", strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()))
	}
	n.LogActionF("successfully submitted oracle aggregate prevote")
}

// should be submitted after prevote, and using the same salt
func (n *NodeConfig) SubmitOracleAggregateVote(salt string, amount string) {
	// Only validators should submit oracle votes.
	if !n.IsValidator {
		n.LogActionF("skipping oracle aggregate vote: node is not a validator")
		return
	}
	if n.OperatorAddress == "" {
		// Best-effort resolve now to avoid reveal with mismatched/unknown validator
		_ = n.extractOperatorAddressIfValidator()
	}
	require.NotEmpty(n.t, n.OperatorAddress, "validator operator address must be known before submitting oracle vote")
	n.LogActionF("submitting oracle aggregate vote for %s", n.OperatorAddress)
	// IMPORTANT: positional args must come BEFORE flags for cobra to parse them; pass validator before --from
	base := []string{"terrad", "tx", "oracle", "aggregate-vote", salt, amount, n.OperatorAddress,
		fmt.Sprintf("--from=%s", initialization.ValidatorWalletName), fmt.Sprintf("--chain-id=%s", n.chainID), "--yes", "--keyring-backend=test", "--log_format=json",
		"--gas=4000000", "--fees=0uluna"}
	// Use ExecCmd directly with empty success string so we can parse and retry ourselves without require.Eventually gating.
	for attempt := 1; attempt <= 6; attempt++ {
		outBuf, errBuf, _ := n.containerManager.ExecCmd(n.t, n.Name, base, "", false)
		out := strings.TrimSpace(outBuf.String())
		errS := strings.TrimSpace(errBuf.String())
		// Try to decode code field; fall back to substring search
		var resp struct {
			Code   int    `json:"code"`
			RawLog string `json:"raw_log"`
		}
		_ = json.Unmarshal(outBuf.Bytes(), &resp)
		if resp.Code == 0 || strings.Contains(out, "\"code\":0") {
			if txh := extractTxHashFromJSON(outBuf.Bytes()); txh != "" {
				n.LogActionF("vote tx accepted; txhash=%s", txh)
			} else if txh := extractTxHashFromJSON(errBuf.Bytes()); txh != "" {
				n.LogActionF("vote tx accepted; txhash=%s (stderr)", txh)
			} else {
				n.LogActionF("vote tx accepted; stdout=%q stderr=%q", out, errS)
			}
			n.LogActionF("successfully submitted oracle aggregate vote")
			return
		}
		if strings.Contains(out, "no aggregate prevote") || strings.Contains(resp.RawLog, "no aggregate prevote") {
			n.LogActionF("vote attempt %d failed with 'no aggregate prevote'; retrying shortly... stdout=%q stderr=%q", attempt, out, errS)
			time.Sleep(1 * time.Second)
			continue
		}
		// Non-retryable failure: surface details and stop
		require.Failf(n.t, "aggregate vote failed", "validator=%s stdout=%s stderr=%s", n.OperatorAddress, out, errS)
	}
}

// HasOracleAggregatePrevote returns true if this validator has an aggregate prevote recorded on-chain.
func (n *NodeConfig) HasOracleAggregatePrevote() bool {
	if !n.IsValidator {
		return false
	}
	if n.OperatorAddress == "" {
		n.LogActionF("cannot query aggregate prevote: operator address unknown")
		return false
	}
	_, _, ok := n.GetOracleAggregatePrevote()
	return ok
}

// CountOracleAggregatePrevotes returns the number of outstanding aggregate prevotes across all validators.
func (n *NodeConfig) CountOracleAggregatePrevotes() int {
	cmd := []string{"terrad", "query", "oracle", "aggregate-prevotes", "--output=json"}
	outBuf, _, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	if err != nil {
		n.LogActionF("failed to query aggregate prevotes: %v", err)
		return 0
	}
	var resp struct {
		AggregatePrevotes []struct {
			Voter string `json:"voter"`
		} `json:"aggregate_prevotes"`
	}
	if err := json.Unmarshal(outBuf.Bytes(), &resp); err != nil {
		n.LogActionF("failed to decode aggregate prevotes json: %v", err)
		return 0
	}
	return len(resp.AggregatePrevotes)
}

// QueryOracleAggregatePrevoteFor returns (hash, submitBlock, ok) for the given voter address, querying via this node's container.
func (n *NodeConfig) QueryOracleAggregatePrevoteFor(voter string) (string, uint64, bool) {
	if voter == "" {
		return "", 0, false
	}
	cmd := []string{"terrad", "query", "oracle", "aggregate-prevotes", voter, "--output=json"}
	outBuf, errBuf, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	if err != nil {
		n.LogActionF("aggregate prevote query failed for %s (err=%v)", voter, err)
		return "", 0, false
	}
	if strings.TrimSpace(outBuf.String()) == "" {
		if strings.TrimSpace(errBuf.String()) != "" {
			n.LogActionF("aggregate prevote stderr for %s: %s", voter, errBuf.String())
		}
		return "", 0, false
	}
	var resp struct {
		AggregatePrevote struct {
			Hash        string `json:"hash"`
			Voter       string `json:"voter"`
			SubmitBlock string `json:"submit_block"`
		} `json:"aggregate_prevote"`
	}
	if err := json.Unmarshal(outBuf.Bytes(), &resp); err != nil {
		n.LogActionF("failed to decode aggregate prevote json for %s: %v", voter, err)
		return "", 0, false
	}
	if resp.AggregatePrevote.Voter != voter || resp.AggregatePrevote.Hash == "" {
		return "", 0, false
	}
	sb, perr := strconv.ParseUint(resp.AggregatePrevote.SubmitBlock, 10, 64)
	if perr != nil {
		n.LogActionF("failed to parse submit_block %q for %s: %v", resp.AggregatePrevote.SubmitBlock, voter, perr)
		return resp.AggregatePrevote.Hash, 0, true
	}
	return resp.AggregatePrevote.Hash, sb, true
}

// GetOracleAggregatePrevote returns (hash, submitBlock, ok) for this validator's aggregate prevote
func (n *NodeConfig) GetOracleAggregatePrevote() (string, uint64, bool) {
	if !n.IsValidator || n.OperatorAddress == "" {
		return "", 0, false
	}
	return n.QueryOracleAggregatePrevoteFor(n.OperatorAddress)
}

// GetOracleAggregatePrevoteVia queries this validator's aggregate prevote using the provided reference node's container.
// This helps avoid cases where the validator's own container is lagging or in state-sync and cannot serve the query reliably.
func (n *NodeConfig) GetOracleAggregatePrevoteVia(via *NodeConfig) (string, uint64, bool) {
	if via == nil || !n.IsValidator || n.OperatorAddress == "" {
		return "", 0, false
	}
	return via.QueryOracleAggregatePrevoteFor(n.OperatorAddress)
}

// IsCatchingUp returns true if the node reports it is still catching up (state-sync/fast-sync) via `terrad status`.
// It parses the JSON output to read `sync_info.catching_up` and falls back to true on any error to be conservative.
func (n *NodeConfig) IsCatchingUp() bool {
	cmd := []string{"terrad", "status"}
	outBuf, errBuf, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "", false)
	if err != nil {
		n.LogActionF("status query failed: %v", err)
		return true
	}
	// terrad status usually writes JSON to stderr; try stderr first, then stdout as fallback
	payload := errBuf.Bytes()
	if len(payload) == 0 {
		payload = outBuf.Bytes()
	}
	var resp struct {
		SyncInfo struct {
			CatchingUp bool `json:"catching_up"`
		} `json:"sync_info"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		n.LogActionF("failed to decode status json: %v; stdout=%q stderr=%q", err, strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()))
		return true
	}
	return resp.SyncInfo.CatchingUp
}
