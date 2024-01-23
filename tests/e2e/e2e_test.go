package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	ibchookskeeper "github.com/terra-money/core/v2/x/ibc-hooks/keeper"

	"github.com/classic-terra/core/v2/tests/e2e/initialization"
)

func (s *IntegrationTestSuite) TestIBCWasmHooks() {
	if s.skipIBC {
		s.T().Skip("Skipping IBC tests")
	}
	chainA := s.configurer.GetChainConfig(0)
	chainB := s.configurer.GetChainConfig(1)

	nodeA, err := chainA.GetDefaultNode()
	s.NoError(err)
	nodeB, err := chainB.GetDefaultNode()
	s.NoError(err)

	nodeA.StoreWasmCode("counter.wasm", initialization.ValidatorWalletName)
	chainA.LatestCodeId = int(nodeA.QueryLatestWasmCodeID())
	nodeA.InstantiateWasmContract(
		strconv.Itoa(chainA.LatestCodeId),
		`{"count": "0"}`,
		initialization.ValidatorWalletName)

	contracts, err := nodeA.QueryContractsFromId(chainA.LatestCodeId)
	s.NoError(err)
	s.Require().Len(contracts, 1, "Wrong number of contracts for the counter")
	contractAddr := contracts[0]

	transferAmount := int64(10)
	validatorAddr := nodeB.GetWallet(initialization.ValidatorWalletName)
	nodeB.SendIBCTransfer(validatorAddr, contractAddr, fmt.Sprintf("%dluna", transferAmount),
		fmt.Sprintf(`{"wasm":{"contract":"%s","msg": {"increment": {}} }}`, contractAddr))

	// check the balance of the contract
	s.Eventually(func() bool {
		balance, err := nodeA.QueryBalances(contractAddr)
		s.Require().NoError(err)
		if len(balance) == 0 {
			return false
		}
		return balance[0].Amount.Int64() == transferAmount
	},
		1*time.Minute,
		10*time.Millisecond,
	)

	// sender wasm addr
	senderBech32, err := ibchookskeeper.DeriveIntermediateSender("channel-0", validatorAddr, "terra")

	var response map[string]interface{}
	s.Eventually(func() bool {
		response, err = nodeA.QueryWasmSmart(contractAddr, fmt.Sprintf(`{"get_total_funds": {"addr": "%s"}}`, senderBech32))
		totalFunds := response["total_funds"].([]interface{})[0]
		amount := totalFunds.(map[string]interface{})["amount"].(string)
		denom := totalFunds.(map[string]interface{})["denom"].(string)
		// check if denom contains "luna"
		return err == nil && amount == strconv.FormatInt(transferAmount, 10) && strings.Contains(denom, "ibc")
	},
		15*time.Second,
		10*time.Millisecond,
	)
}
