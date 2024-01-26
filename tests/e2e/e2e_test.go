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
	s.Len(contracts, 1, "Wrong number of contracts for the counter")
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
		10*time.Second,
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
	s.Require().NoError(err)

	transferAmount1 := sdkmath.NewInt(20000000)
	transferCoin1 := sdk.NewCoin("uluna", transferAmount1)

	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")

	validatorBalance, err := node.QuerySpecificBalance(validatorAddr, "uluna")
	s.Require().NoError(err)

	fmt.Println("validatorBalance ", validatorBalance)

	test1Addr := node.CreateWallet("test1")

	// Test 1: banktypes.MsgSend
	// burn tax with bank send
	node.BankSend(transferCoin1.String(), validatorAddr, test1Addr)

	subAmount := sdk.NewDecFromInt(transferAmount1).Mul(sdk.NewDecWithPrec(102, 2)).TruncateInt()
	taxRate, err := node.QueryTaxRate()
	s.Require().NoError(err)
	s.Require().Equal(subAmount, taxRate.Add(sdk.OneDec()).MulInt(transferAmount1).TruncateInt())

	decremented := validatorBalance.Sub(sdk.NewCoin("uluna", subAmount))
	newValidatorBalance, err := node.QuerySpecificBalance(validatorAddr, "uluna")
	s.Require().NoError(err)

	balanceTest1, err := node.QuerySpecificBalance(test1Addr, "uluna")
	s.Require().NoError(err)

	s.Require().Equal(balanceTest1.Amount, transferAmount1)
	s.Require().Equal(newValidatorBalance, decremented)

	// Test 2: try bank send with grant
	test2Addr := node.CreateWallet("test2")
	transferAmount2 := sdkmath.NewInt(10000000)
	transferCoin2 := sdk.NewCoin("uluna", transferAmount2)

	node.BankSend(transferCoin2.String(), validatorAddr, test2Addr)
	node.GrantAddress(test2Addr, test1Addr, transferCoin2.String(), "test2")

	validatorBalance, err = node.QuerySpecificBalance(validatorAddr, "uluna")
	s.Require().NoError(err)

	node.BankSendFeeGrantWithWallet(transferCoin2.String(), test1Addr, validatorAddr, test2Addr, "test1")

	fmt.Println("validatorBalance ", validatorBalance)

	time.Sleep(10 * time.Second)

	newValidatorBalance, err = node.QuerySpecificBalance(validatorAddr, "uluna")
	s.Require().NoError(err)

	fmt.Println("newValidatorBalance ", newValidatorBalance)

	balanceTest1, err = node.QuerySpecificBalance(test1Addr, "uluna")
	s.Require().NoError(err)

	fmt.Println("balanceTest1 ", balanceTest1)

	balanceTest2, err := node.QuerySpecificBalance(test2Addr, "uluna")
	s.Require().NoError(err)

	fmt.Println("balanceTest2 ", balanceTest2)

	s.Require().Equal(balanceTest1.Amount, transferAmount1.Sub(transferAmount2))
	s.Require().Equal(newValidatorBalance, validatorBalance.Add(transferCoin2))
	s.Require().Equal(balanceTest2.Amount, sdk.NewDecWithPrec(98, 2).MulInt(transferAmount2).TruncateInt())

	// Test 3: try bank send with BurnTaxExemption whitelist address
	whitelistAddr1 := node.CreateWallet("whitelist1")
	node.BankSend(transferCoin1.String(), validatorAddr, whitelistAddr1)
	whitelistAddr2 := node.CreateWallet("whitelist2")
	node.BankSend(transferCoin1.String(), validatorAddr, whitelistAddr2)

	chain.AddBurnTaxExemptionAddressProposal(node, whitelistAddr1, whitelistAddr2)

	whitelistedAddresses, err := node.QueryBurnTaxExemptionList()
	s.Require().NoError(err)
	s.Require().Len(whitelistedAddresses, 2)
	s.Require().Contains(whitelistedAddresses, whitelistAddr1)
	s.Require().Contains(whitelistedAddresses, whitelistAddr2)

	node.BankSendWithWallet(transferCoin2.String(), whitelistAddr1, whitelistAddr2, "whitelist1")

	balancesWhitelistAddr1, err := node.QuerySpecificBalance(whitelistAddr1, "uluna")
	s.Require().NoError(err)
	s.Require().Equal(balancesWhitelistAddr1.Amount, transferAmount2)

	balancesWhitelistAddr2, err := node.QuerySpecificBalance(whitelistAddr2, "uluna")
	s.Require().NoError(err)
	s.Require().Equal(balancesWhitelistAddr2.Amount, transferAmount2.Add(transferAmount1)) // transferAmount1 = 2 * transferAmount2

	// Test 4: banktypes.MsgMultiSend
	validatorBalance, err = node.QuerySpecificBalance(validatorAddr, "uluna")
	s.Require().NoError(err)

	node.BankMultiSend(transferCoin1.String(), false, validatorAddr, test1Addr, test2Addr)

	newValidatorBalance, err = node.QuerySpecificBalance(validatorAddr, "uluna")
	s.Require().NoError(err)

	subAmount = sdk.NewDecWithPrec(202, 2).MulInt(transferAmount1).TruncateInt()
	s.Require().Equal(newValidatorBalance, validatorBalance.Sub(sdk.NewCoin("uluna", subAmount)))
}
