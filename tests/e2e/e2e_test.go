package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"

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

	transferAmount := sdkmath.NewInt(10000000)
	validatorAddr := nodeB.GetWallet(initialization.ValidatorWalletName)
	nodeB.SendIBCTransfer(validatorAddr, contractAddr, fmt.Sprintf("%duluna", transferAmount.Int64()),
		fmt.Sprintf(`{"wasm":{"contract":"%s","msg": {"increment": {}} }}`, contractAddr))

	// check the balance of the contract
	s.Eventually(func() bool {
		balance, err := nodeA.QueryBalances(contractAddr)
		s.Require().NoError(err)
		if len(balance) == 0 {
			return false
		}
		return balance[0].Amount.Equal(transferAmount)
	},
		1*time.Minute,
		10*time.Millisecond)

	// sender wasm addr
	// senderBech32, err := ibchookskeeper.DeriveIntermediateSender("channel-0", validatorAddr, "terra")

	var response map[string]interface{}
	s.Eventually(func() bool {
		response, err = nodeA.QueryWasmSmart(contractAddr, `{"get_total_funds": {}}`)
		if err != nil {
			return false
		}
		totalFunds := response["total_funds"].([]interface{})[0]
		amount, err := strconv.ParseInt(totalFunds.(map[string]interface{})["amount"].(string), 10, 64)
		if err != nil {
			return false
		}
		denom := totalFunds.(map[string]interface{})["denom"].(string)
		// check if denom contains "luna"
		return sdkmath.NewInt(int64(amount)).Equal(transferAmount) && strings.Contains(denom, "ibc")
	},
		15*time.Second,
		10*time.Millisecond,
	)
}

func (s *IntegrationTestSuite) TestPacketForwardMiddleware() {
	if s.skipIBC {
		s.T().Skip("Skipping IBC tests")
	}
	chainA := s.configurer.GetChainConfig(0)
	chainB := s.configurer.GetChainConfig(1)
	chainC := s.configurer.GetChainConfig(2)

	nodeA, err := chainA.GetDefaultNode()
	s.NoError(err)
	nodeB, err := chainB.GetDefaultNode()
	s.NoError(err)
	nodeC, err := chainC.GetDefaultNode()
	s.NoError(err)

	validatorAddr := nodeA.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")

	balan, err := nodeA.QueryBalances(validatorAddr)
	s.NoError(err)
	s.T().Logf("balance validatorAddr: %v", balan)

	receiver := nodeB.CreateWallet("receiver")

	// query old bank balances
	balanceReceiverOld, err := nodeC.QueryBalances(receiver)
	s.NoError(err)
	s.T().Logf("balance olld: %v", balanceReceiverOld)

	nodeA.SendIBCTransfer(validatorAddr, receiver, "100uluna",
		fmt.Sprintf(`{"forward":{"receiver":"%s","port":"transfer","channel":"channel-0"}}`, receiver))

	// sleep 30s
	time.Sleep(30 * time.Second)

	balanceReceiverNew, err := nodeC.QueryBalances(receiver)
	s.NoError(err)
	s.T().Logf("balance new: %v", balanceReceiverNew)
}
