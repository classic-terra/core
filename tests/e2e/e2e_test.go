package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

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

	transferAmount := sdk.NewInt(10000000)
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
		return sdk.NewInt(int64(amount)).Equal(transferAmount) && strings.Contains(denom, "ibc")
	},
		15*time.Second,
		10*time.Millisecond,
	)
}

func (s *IntegrationTestSuite) TestPacketForwardMiddleware() {
	if s.skipIBC {
		s.T().Skip("Skipping Packet Forward Middleware tests")
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

	transferAmount := sdk.NewInt(10000000)

	validatorAddr := nodeA.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")

	receiver := nodeB.GetWallet(initialization.ValidatorWalletName)

	// query old bank balances
	balanceReceiverOld, err := nodeC.QueryBalances(receiver)
	s.NoError(err)
	found, _ := balanceReceiverOld.Find("luna")
	s.False(found)

	nodeA.SendIBCTransfer(validatorAddr, receiver, fmt.Sprintf("%duluna", transferAmount.Int64()),
		fmt.Sprintf(`{"forward":{"receiver":"%s","port":"transfer","channel":"channel-2"}}`, receiver))

	// wait for ibc cycle
	time.Sleep(30 * time.Second)

	s.Eventually(func() bool {
		balanceReceiver, err := nodeC.QueryBalances(receiver)
		s.Require().NoError(err)
		if len(balanceReceiver) == 0 {
			return false
		}
		return balanceReceiver[0].Amount.Equal(transferAmount)
	},
		15*time.Second,
		10*time.Millisecond,
	)
}

func (s *IntegrationTestSuite) TestFeeTax() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.NoError(err)

	transferAmount := sdkmath.NewInt(10000000)

	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")

	validatorBalance, err := node.QuerySpecificBalance(validatorAddr, "uluna")
	s.NoError(err)

	fmt.Println("validatorBalance ", validatorBalance)

	testAddr := node.CreateWallet("test1")

	// Test burn tax with bank send
	node.BankSend(fmt.Sprintf("%duluna", transferAmount.Int64()), validatorAddr, testAddr)

	// wait 10s
	time.Sleep(10 * time.Second)

	newValidatorBalance, err := node.QuerySpecificBalance(validatorAddr, "uluna")
	s.Require().NoError(err)

	fmt.Println("newValidatorBalance", newValidatorBalance)
	subAmount := sdk.NewDecFromInt(transferAmount).Mul(sdk.NewDecWithPrec(102, 2)).TruncateInt()
	fmt.Println("subAmount", subAmount)

	taxRate, err := node.QueryTaxRate()
	s.Require().NoError(err)
	fmt.Println("taxRate", taxRate)

	s.Eventually(func() bool {
		decremented := validatorBalance.Sub(sdk.NewCoin("uluna", subAmount))
		newValidatorBalance, err := node.QuerySpecificBalance(validatorAddr, "uluna")
		s.Require().NoError(err)

		balanceTest1, err := node.QuerySpecificBalance(testAddr, "uluna")
		s.Require().NoError(err)

		return balanceTest1.Amount.Equal(transferAmount) && newValidatorBalance.IsEqual(decremented)
	},
		15*time.Second,
		10*time.Millisecond,
	)

}
