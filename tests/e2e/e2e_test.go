package e2e

import (
	"fmt"
	"strconv"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v3/tests/e2e/initialization"
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
	chainA.LatestCodeID = int(nodeA.QueryLatestWasmCodeID())
	nodeA.InstantiateWasmContract(
		strconv.Itoa(chainA.LatestCodeID),
		`{"count": "0"}`, "",
		initialization.ValidatorWalletName, "200000", sdk.NewCoins(sdk.NewCoin(initialization.TerraDenom, sdk.NewInt(10))))

	contracts, err := nodeA.QueryContractsFromID(chainA.LatestCodeID)
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
		initialization.OneMin,
		10*time.Millisecond)

	// sender wasm addr
	// senderBech32, err := ibchookskeeper.DeriveIntermediateSender("channel-0", validatorAddr, "terra")
	var response interface{}
	response, err = nodeA.QueryWasmSmart(contractAddr, `{"get_total_funds": {}}`)
	s.Require().NoError(err)

	s.Eventually(func() bool {
		response, err = nodeA.QueryWasmSmart(contractAddr, `{"get_total_funds": {}}`)
		if err != nil {
			return false
		}

		totalFunds := response.([]interface{})[0]
		amount, err := strconv.ParseInt(totalFunds.(map[string]interface{})["amount"].(string), 10, 64)
		if err != nil {
			return false
		}
		denom := totalFunds.(map[string]interface{})["denom"].(string)

		response, err = nodeA.QueryWasmSmart(contractAddr, `{"get_count": {}}`)
		if err != nil {
			return false
		}
		count, err := strconv.ParseInt(response.(string), 10, 64)
		if err != nil {
			return false
		}
		// check if denom is uluna token ibc
		return sdk.NewInt(amount).Equal(transferAmount) && denom == initialization.TerraIBCDenom && count == 1
	},
		10*time.Second,
		10*time.Millisecond,
	)
}

func (s *IntegrationTestSuite) TestAddBurnTaxExemptionAddress() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	whitelistAddr1 := node.CreateWallet("whitelist1")
	whitelistAddr2 := node.CreateWallet("whitelist2")

	chain.AddBurnTaxExemptionAddressProposal(node, whitelistAddr1, whitelistAddr2)

	whitelistedAddresses, err := node.QueryBurnTaxExemptionList()
	s.Require().NoError(err)
	s.Require().Len(whitelistedAddresses, 2)
	s.Require().Contains(whitelistedAddresses, whitelistAddr1)
	s.Require().Contains(whitelistedAddresses, whitelistAddr2)
}

// Each tx gas will cost 2 uluna (1 is for ante handler, 1 is for post handler)
func (s *IntegrationTestSuite) TestFeeTax() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	transferAmount1 := sdkmath.NewInt(20000000)
	transferCoin1 := sdk.NewCoin(initialization.TerraDenom, transferAmount1)

	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")

	validatorBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	test1Addr := node.CreateWallet("test1")
	test2Addr := node.CreateWallet("test2")

	// Test 1: banktypes.MsgSend
	// burn tax with bank send
	subAmount := transferAmount1.Add(initialization.TaxRate.MulInt(transferAmount1).TruncateInt())

	gasLimit := transferCoin1.Amount.MulRaw(initialization.E10).String()
	node.BankSend(transferCoin1.String(), validatorAddr, test1Addr, gasLimit, sdk.NewCoins(transferCoin1))

	decremented := validatorBalance.Sub(sdk.NewCoin(initialization.TerraDenom, subAmount.AddRaw(2)))
	newValidatorBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	balanceTest1, err := node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	s.Require().Equal(balanceTest1.Amount, transferAmount1)
	s.Require().Equal(newValidatorBalance, decremented)

	// Test 2: banktypes.MsgMultiSend
	validatorBalance, err = node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	totalTransferAmount := transferAmount1.Mul(sdk.NewInt(2))
	gasLimit = transferCoin1.Amount.MulRaw(initialization.E10).String()
	node.BankMultiSend(transferCoin1.String(), false, validatorAddr, gasLimit, sdk.NewCoins(transferCoin1), test1Addr, test2Addr)

	newValidatorBalance, err = node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	subAmount = totalTransferAmount.Add(initialization.TaxRate.MulInt(totalTransferAmount).TruncateInt())
	s.Require().Equal(newValidatorBalance, validatorBalance.Sub(sdk.NewCoin(initialization.TerraDenom, subAmount.AddRaw(2))))
}

// Each tx gas will cost 2 uluna (1 is for ante handler, 1 is for post handler)
func (s *IntegrationTestSuite) TestFeeTaxWasm() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	testAddr := node.CreateWallet("test")
	transferAmount := sdkmath.NewInt(100000000)
	transferCoin := sdk.NewCoin(initialization.TerraDenom, transferAmount)

	gasLimit := transferCoin.Amount.MulRaw(initialization.E10).String()
	node.BankSend(fmt.Sprintf("%suluna", transferAmount.Mul(sdk.NewInt(4))), initialization.ValidatorWalletName, testAddr, gasLimit, sdk.NewCoins(transferCoin))
	node.StoreWasmCode("counter.wasm", initialization.ValidatorWalletName)
	chain.LatestCodeID = int(node.QueryLatestWasmCodeID())
	// instantiate contract and transfer 100000000uluna
	node.InstantiateWasmContract(
		strconv.Itoa(chain.LatestCodeID),
		`{"count": "0"}`, transferCoin.String(),
		"test", gasLimit, sdk.NewCoins(transferCoin))

	contracts, err := node.QueryContractsFromID(chain.LatestCodeID)
	s.Require().NoError(err)
	s.Require().Len(contracts, 1, "Wrong number of contracts for the counter")

	balance1, err := node.QuerySpecificBalance(testAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	// 400000000 - 100000000 - 100000000 * TaxRate - 2 (gas) = 300000000 - 10000000 * TaxRate - 2 (gas)
	taxAmount := initialization.TaxRate.MulInt(transferAmount).TruncateInt()
	s.Require().Equal(balance1.Amount, transferAmount.Mul(sdk.NewInt(3)).Sub(taxAmount).SubRaw(2))

	node.Instantiate2WasmContract(
		strconv.Itoa(chain.LatestCodeID),
		`{"count": "0"}`, "salt",
		transferCoin.String(),
		"test", gasLimit, sdk.NewCoins(transferCoin))

	contracts, err = node.QueryContractsFromID(chain.LatestCodeID)
	s.Require().NoError(err)
	s.Require().Len(contracts, 2, "Wrong number of contracts for the counter")

	balance2, err := node.QuerySpecificBalance(testAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	// balance1 - 100000000 - 100000000 * TaxRate - 2 (gas)
	taxAmount = initialization.TaxRate.MulInt(transferAmount).TruncateInt()
	s.Require().Equal(balance2.Amount, balance1.Amount.Sub(transferAmount).Sub(taxAmount).SubRaw(2))

	contractAddr := contracts[0]
	node.WasmExecute(contractAddr, `{"donate": {}}`, transferCoin.String(), "test", gasLimit, sdk.NewCoins(transferCoin))

	balance3, err := node.QuerySpecificBalance(testAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	// balance2 - 100000000 - 100000000 * TaxRate - 2 (gas)
	taxAmount = initialization.TaxRate.MulInt(transferAmount).TruncateInt()
	s.Require().Equal(balance3.Amount, balance2.Amount.Sub(transferAmount).Sub(taxAmount).SubRaw(2))
}

func (s *IntegrationTestSuite) TestFeeTaxGrant() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	transferAmount := sdkmath.NewInt(100000000)
	transferCoin := sdk.NewCoin(initialization.TerraDenom, transferAmount)

	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")

	test1Addr := node.CreateWallet("test1")
	test2Addr := node.CreateWallet("test2")

	// Test 1: try bank send with grant
	gasLimit := transferCoin.Amount.MulRaw(initialization.E10).String()
	node.BankSend(transferCoin.String(), validatorAddr, test1Addr, gasLimit, sdk.NewCoins(transferCoin))
	node.BankSend(transferCoin.String(), validatorAddr, test1Addr, gasLimit, sdk.NewCoins(transferCoin))
	node.BankSend(transferCoin.String(), validatorAddr, test2Addr, gasLimit, sdk.NewCoins(transferCoin))
	node.GrantAddress(test2Addr, test1Addr, transferCoin.String(), "test2")

	validatorBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	node.BankSendFeeGrantWithWallet(transferCoin.String(), test1Addr, validatorAddr, test2Addr, "test1", gasLimit, sdk.NewCoins(transferCoin))

	newValidatorBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	balanceTest1, err := node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	balanceTest2, err := node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	s.Require().Equal(balanceTest1, transferCoin)
	s.Require().Equal(newValidatorBalance, validatorBalance.Add(transferCoin))
	// addr2 lost 2uluna to pay for grant msg's gas,  100000000 * TaxRate + 2uluna to pay for bank send msg's tx fees,
	s.Require().Equal(balanceTest2.Amount, transferAmount.Sub(initialization.TaxRate.MulInt(transferAmount).TruncateInt()).SubRaw(4))

	// Test 2: try bank send with grant but pay by multiple fees denom
}
