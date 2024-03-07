package chain

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/classic-terra/core/v2/tests/e2e/configurer/config"
	"github.com/classic-terra/core/v2/tests/e2e/containers"
	"github.com/classic-terra/core/v2/tests/e2e/initialization"
	treasurytypes "github.com/classic-terra/core/v2/x/treasury/types"
)

type Config struct {
	initialization.ChainMeta

	ValidatorInitConfigs []*initialization.NodeConfig
	// voting period is number of blocks it takes to deposit, 1.2 seconds per validator to vote on the prop, and a buffer.
	VotingPeriod          float32
	ExpeditedVotingPeriod float32
	// upgrade proposal height for chain.
	UpgradePropHeight    int64
	LatestProposalNumber int
	LatestLockNumber     int
	NodeConfigs          []*NodeConfig

	LatestCodeID int

	t                *testing.T
	containerManager *containers.Manager
}

const (
	// defaultNodeIndex to use for querying and executing transactions.
	// It is used when we are indifferent about the node we are working with.
	defaultNodeIndex = 0
	// waitUntilRepeatPauseTime is the time to wait between each check of the node status.
	waitUntilRepeatPauseTime = 2 * time.Second
	// waitUntilrepeatMax is the maximum number of times to repeat the wait until condition.
	waitUntilrepeatMax = 60
)

func New(t *testing.T, containerManager *containers.Manager, id string, initValidatorConfigs []*initialization.NodeConfig) *Config {
	numVal := float32(len(initValidatorConfigs))
	return &Config{
		ChainMeta: initialization.ChainMeta{
			ID: id,
		},
		ValidatorInitConfigs:  initValidatorConfigs,
		VotingPeriod:          config.PropDepositBlocks + numVal*config.PropVoteBlocks + config.PropBufferBlocks,
		ExpeditedVotingPeriod: config.PropDepositBlocks + numVal*config.PropVoteBlocks + config.PropBufferBlocks - 2,
		t:                     t,
		containerManager:      containerManager,
	}
}

// CreateNode returns new initialized NodeConfig.
func (c *Config) CreateNode(initNode *initialization.Node) *NodeConfig {
	nodeConfig := &NodeConfig{
		Node:             *initNode,
		chainID:          c.ID,
		containerManager: c.containerManager,
		t:                c.t,
	}
	c.NodeConfigs = append(c.NodeConfigs, nodeConfig)
	return nodeConfig
}

// RemoveNode removes node and stops it from running.
func (c *Config) RemoveNode(nodeName string) error {
	for i, node := range c.NodeConfigs {
		if node.Name == nodeName {
			c.NodeConfigs = append(c.NodeConfigs[:i], c.NodeConfigs[i+1:]...)
			return node.Stop()
		}
	}
	return fmt.Errorf("node %s not found", nodeName)
}

// WaitUntilHeight waits for all validators to reach the specified height at the minimum.
// returns error, if any.
func (c *Config) WaitUntilHeight(height int64) {
	// Ensure the nodes are making progress.
	doneCondition := func(syncInfo coretypes.SyncInfo) bool {
		curHeight := syncInfo.LatestBlockHeight

		if curHeight < height {
			c.t.Logf("current block height is %d, waiting to reach: %d", curHeight, height)
			return false
		}

		return !syncInfo.CatchingUp
	}

	for _, node := range c.NodeConfigs {
		c.t.Logf("node container: %s, waiting to reach height %d", node.Name, height)
		node.WaitUntil(doneCondition)
	}
}

// WaitForNumHeights waits for all nodes to go through a given number of heights.
func (c *Config) WaitForNumHeights(heightsToWait int64) {
	node, err := c.GetDefaultNode()
	require.NoError(c.t, err)
	currentHeight, err := node.QueryCurrentHeight()
	require.NoError(c.t, err)
	c.WaitUntilHeight(currentHeight + heightsToWait)
}

func (c *Config) SendIBC(dstChain *Config, recipient string, token sdk.Coin, hermesContainerName string) {
	c.t.Logf("IBC sending %s from %s to %s (%s)", token, c.ID, dstChain.ID, recipient)

	dstNode, err := dstChain.GetDefaultNode()
	require.NoError(c.t, err)

	balancesDstPre, err := dstNode.QueryBalances(recipient)
	require.NoError(c.t, err)

	cmd := []string{"hermes", "tx", "raw", "ft-transfer", dstChain.ID, c.ID, "transfer", "channel-0", token.Amount.String(), fmt.Sprintf("--denom=%s", token.Denom), fmt.Sprintf("--receiver=%s", recipient), "--timeout-height-offset=1000"}
	_, _, err = c.containerManager.ExecHermesCmd(c.t, cmd, hermesContainerName, "Success")
	require.NoError(c.t, err)

	require.Eventually(
		c.t,
		func() bool {
			balancesDstPost, err := dstNode.QueryBalances(recipient)
			require.NoError(c.t, err)
			ibcCoin := balancesDstPost.Sub(balancesDstPre...)
			if ibcCoin.Len() == 1 {
				tokenPre := balancesDstPre.AmountOfNoDenomValidation(ibcCoin[0].Denom)
				tokenPost := balancesDstPost.AmountOfNoDenomValidation(ibcCoin[0].Denom)
				resPre := token.Amount
				resPost := tokenPost.Sub(tokenPre)
				return resPost.Uint64() == resPre.Uint64()
			}
			return false
		},
		initialization.FiveMin,
		time.Second,
		"tx not received on destination chain",
	)

	c.t.Log("successfully sent IBC tokens")
}

func (c *Config) GetDefaultNode() (*NodeConfig, error) {
	return c.getNodeAtIndex(defaultNodeIndex)
}

// GetPersistentPeers returns persistent peers from every node
// associated with a chain.
func (c *Config) GetPersistentPeers() []string {
	peers := make([]string, len(c.NodeConfigs))
	for i, node := range c.NodeConfigs {
		peers[i] = node.PeerID
	}
	return peers
}

func (c *Config) getNodeAtIndex(nodeIndex int) (*NodeConfig, error) {
	if nodeIndex > len(c.NodeConfigs) {
		return nil, fmt.Errorf("node index (%d) is greter than the number of nodes available (%d)", nodeIndex, len(c.NodeConfigs))
	}
	return c.NodeConfigs[nodeIndex], nil
}

func (c *Config) AddBurnTaxExemptionAddressProposal(chainANode *NodeConfig, addresses ...string) {
	proposal := treasurytypes.AddBurnTaxExemptionAddressProposal{
		Title:       "Add Burn Tax Exemption Address",
		Description: fmt.Sprintf("Add %s to the burn tax exemption address list", strings.Join(addresses, ",")),
		Addresses:   addresses,
	}
	proposalJSON, err := json.Marshal(proposal)
	require.NoError(c.t, err)

	wd, err := os.Getwd()
	require.NoError(c.t, err)
	localProposalFile := wd + "/scripts/add_burn_tax_exemption_address_proposal.json"
	f, err := os.Create(localProposalFile)
	require.NoError(c.t, err)
	_, err = f.WriteString(string(proposalJSON))
	require.NoError(c.t, err)
	err = f.Close()
	require.NoError(c.t, err)

	propNumber := chainANode.SubmitAddBurnTaxExemptionAddressProposal(addresses, initialization.ValidatorWalletName)

	chainANode.DepositProposal(propNumber)
	AllValsVoteOnProposal(c, propNumber)

	time.Sleep(initialization.TwoMin)
	require.Eventually(c.t, func() bool {
		status, err := chainANode.QueryPropStatus(propNumber)
		if err != nil {
			return false
		}
		return status == "PROPOSAL_STATUS_PASSED"
	}, initialization.OneMin, 10*time.Millisecond)
}
