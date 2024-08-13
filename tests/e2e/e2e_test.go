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
		initialization.ValidatorWalletName, []string{}, sdk.NewCoin(initialization.TerraDenom, sdk.NewInt(2)))

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

	node.BankSend(transferCoin1.String(), validatorAddr, test1Addr, []string{initialization.TerraDenom})

	// Due to the fee estimate when using
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
	node.BankMultiSend(transferCoin1.String(), false, validatorAddr, []string{initialization.TerraDenom}, []string{test1Addr, test2Addr})

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

	testAddr := node.CreateWallet("test-wasm")
	transferAmount := sdkmath.NewInt(100000000)
	transferCoin := sdk.NewCoin(initialization.TerraDenom, transferAmount)

	node.BankSend(fmt.Sprintf("%suluna", transferAmount.Mul(sdk.NewInt(4))), initialization.ValidatorWalletName, testAddr, []string{initialization.TerraDenom})
	node.StoreWasmCode("counter.wasm", initialization.ValidatorWalletName)
	chain.LatestCodeID = int(node.QueryLatestWasmCodeID())
	// instantiate contract and transfer 100000000uluna
	node.InstantiateWasmContract(
		strconv.Itoa(chain.LatestCodeID),
		`{"count": "0"}`, transferCoin.String(),
		"test-wasm", []string{initialization.TerraDenom})

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
		"test-wasm", []string{initialization.TerraDenom})

	contracts, err = node.QueryContractsFromID(chain.LatestCodeID)
	s.Require().NoError(err)
	s.Require().Len(contracts, 2, "Wrong number of contracts for the counter")

	balance2, err := node.QuerySpecificBalance(testAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	// balance1 - 100000000 - 100000000 * TaxRate - 2 (gas)
	taxAmount = initialization.TaxRate.MulInt(transferAmount).TruncateInt()
	s.Require().Equal(balance2.Amount, balance1.Amount.Sub(transferAmount).Sub(taxAmount).SubRaw(2))

	contractAddr := contracts[0]
	node.WasmExecute(contractAddr, `{"donate": {}}`, transferCoin.String(), "test-wasm", []string{initialization.TerraDenom})

	balance3, err := node.QuerySpecificBalance(testAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	// balance2 - 100000000 - 100000000 * TaxRate - 2 (gas)
	taxAmount = initialization.TaxRate.MulInt(transferAmount).TruncateInt()
	s.Require().Equal(balance3.Amount, balance2.Amount.Sub(transferAmount).Sub(taxAmount).SubRaw(2))
}

// Each tx gas will cost 2 token (1 is for ante handler, 1 is for post handler)
func (s *IntegrationTestSuite) TestFeeTaxGrant() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	transferAmount1 := sdkmath.NewInt(100000000)
	transferCoin1 := sdk.NewCoin(initialization.TerraDenom, transferAmount1)

	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")

	test1Addr := node.CreateWallet("test1-grant")
	test2Addr := node.CreateWallet("test2-grant")

	// Test 1: try bank send with grant
	node.BankSend(transferCoin1.String(), validatorAddr, test1Addr, []string{initialization.TerraDenom})
	node.BankSend(transferCoin1.String(), validatorAddr, test1Addr, []string{initialization.TerraDenom})
	node.BankSend(transferCoin1.String(), validatorAddr, test2Addr, []string{initialization.TerraDenom})
	node.GrantAddress(test2Addr, test1Addr, "test2-grant", initialization.TerraDenom, transferCoin1.String())

	validatorBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	node.BankSendFeeGrantWithWallet(transferCoin1.String(), test1Addr, validatorAddr, test2Addr, "test1-grant", []string{initialization.TerraDenom})

	newValidatorBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	balanceTest1, err := node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	balanceTest2, err := node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	s.Require().Equal(balanceTest1, transferCoin1)
	s.Require().Equal(newValidatorBalance, validatorBalance.Add(transferCoin1))
	// addr2 lost 2uluna to pay for grant msg's gas,  100000000 * TaxRate + 2uluna to pay for bank send msg's tx fees,
	s.Require().Equal(balanceTest2.Amount, transferAmount1.Sub(initialization.TaxRate.MulInt(transferAmount1).TruncateInt()).SubRaw(12))

	// Test 3: try bank send with no grant
	transferAmount2 := sdkmath.NewInt(200000000)
	transferUsdCoin2 := sdk.NewCoin(initialization.UsdDenom, transferAmount2)
	transferTerraCoin2 := sdk.NewCoin(initialization.TerraDenom, transferAmount2)

	node.BankSend(transferTerraCoin2.String(), validatorAddr, test1Addr, []string{initialization.TerraDenom})
	node.BankSend(transferUsdCoin2.String(), validatorAddr, test1Addr, []string{initialization.UsdDenom})
	node.BankSend(transferUsdCoin2.String(), validatorAddr, test2Addr, []string{initialization.UsdDenom})

	// Revoke previous grant and grant new ones
	node.RevokeGrant(test2Addr, test1Addr, "test2-grant", initialization.UsdDenom)
	feeAmountTerraDenom := sdkmath.NewInt(10)
	feeCoinTerraDenom := sdk.NewCoin(initialization.TerraDenom, feeAmountTerraDenom)
	node.GrantAddress(test2Addr, test1Addr, "test2-grant", initialization.UsdDenom, sdk.NewCoins(transferUsdCoin2, feeCoinTerraDenom).String())

	validatorTerraBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	balanceTest2TerraBalance, err := node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	node.BankSendFeeGrantWithWallet(transferTerraCoin2.String(), test1Addr, validatorAddr, test2Addr, "test1-grant", []string{}, transferUsdCoin2, feeCoinTerraDenom)

	newValidatorTerraBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	balanceTest1TerraBalance, err := node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)
	balanceTest1UsdBalance, err := node.QuerySpecificBalance(test1Addr, initialization.UsdDenom)
	s.Require().NoError(err)
	newBalanceTest2TerraBalance, err := node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)
	balanceTest2UsdBalance, err := node.QuerySpecificBalance(test2Addr, initialization.UsdDenom)
	s.Require().NoError(err)
	// The fee grant msg only support to pay by one denom, so only uusd balance will change
	s.Require().Equal(newValidatorTerraBalance, validatorTerraBalance.Add(transferTerraCoin2))
	s.Require().Equal(balanceTest1TerraBalance, balanceTest1)
	s.Require().Equal(balanceTest1UsdBalance, transferUsdCoin2)
	// 1uluna will be used for ante handler
	s.Require().Equal(newBalanceTest2TerraBalance.Amount, balanceTest2TerraBalance.Amount.SubRaw(1))
	s.Require().Equal(
		balanceTest2UsdBalance.Amount,
		transferAmount2.Sub(
			initialization.TaxRate.MulInt(transferAmount2). // tax amount in the form of terra denom
									Mul(initialization.UsdGasPrice.Quo(initialization.TerraGasPrice)). // convert terra denom to usd denom base on gas price
									TruncateInt(),
		).SubRaw(21), // addr2 lost 10uusd to pay for revoke msg's gas, 10uusd to pay for grant msg's gas, 1uusd to pay for band send msg's gas
	)
}

func (s *IntegrationTestSuite) TestFeeTaxNotSupport() {
	if s.skipIBC {
		s.T().Skip("Skipping IBC tests")
	}
	chainA := s.configurer.GetChainConfig(0)
	chainB := s.configurer.GetChainConfig(1)

	nodeA, err := chainA.GetDefaultNode()
	s.NoError(err)
	nodeB, err := chainB.GetDefaultNode()
	s.NoError(err)

	transferAmount1 := sdkmath.NewInt(30000000)
	transferCoin1 := sdk.NewCoin(initialization.TerraDenom, transferAmount1)

	validatorAddrChainA := nodeA.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddrChainA, "")
	validatorAddrChainB := nodeB.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddrChainB, "")

	testAddrChainA := nodeA.CreateWallet("test1-feetax-not-support")
	test1AddrChainB := nodeB.CreateWallet("test1-feetax-not-support")
	test2AddrChainB := nodeB.CreateWallet("test2-feetax-not-support")

	// Test 1: try bank send with ibc denom
	nodeA.BankSend(transferCoin1.String(), validatorAddrChainA, testAddrChainA, []string{initialization.TerraDenom})
	nodeB.BankSend(transferCoin1.String(), validatorAddrChainB, test1AddrChainB, []string{initialization.TerraDenom})

	transferAmount2 := sdkmath.NewInt(20000000)
	transferCoin2 := sdk.NewCoin(initialization.TerraDenom, transferAmount2)
	nodeA.SendIBCTransfer("test1-feetax-not-support", test1AddrChainB, transferCoin2.String(), "")

	// check the balance of the contract
	s.Eventually(
		func() bool {
			balance, err := nodeB.QueryBalances(test1AddrChainB)
			s.Require().NoError(err)
			if len(balance) == 0 {
				return false
			}
			return balance[0].Amount.Equal(transferAmount2)
		},
		initialization.OneMin,
		10*time.Millisecond,
	)
	terraIBCBalance, err := nodeB.QuerySpecificBalance(test1AddrChainB, initialization.TerraIBCDenom)
	s.Require().NoError(err)
	s.Require().Equal(terraIBCBalance.Amount, transferAmount2)

	terraBalance, err := nodeB.QuerySpecificBalance(test1AddrChainB, initialization.TerraDenom)
	s.Require().NoError(err)
	s.Require().Equal(terraBalance.Amount, transferAmount1)

	transferAmount3 := sdkmath.NewInt(10000000)
	transferCoin3 := sdk.NewCoin(initialization.TerraIBCDenom, transferAmount3)

	nodeB.BankSend(transferCoin3.String(), test1AddrChainB, test2AddrChainB, []string{}, sdk.NewCoin(initialization.TerraDenom, sdkmath.NewInt(2)))

	newTerraIBCBalance, err := nodeB.QuerySpecificBalance(test1AddrChainB, initialization.TerraIBCDenom)
	s.Require().NoError(err)
	s.Require().Equal(newTerraIBCBalance.Amount, terraIBCBalance.Amount.Sub(transferAmount3))
	newTerraIBCBalance, err = nodeB.QuerySpecificBalance(test2AddrChainB, initialization.TerraIBCDenom)
	s.Require().NoError(err)
	s.Require().Equal(newTerraIBCBalance.Amount, transferAmount3)

	newTerraBalance, err := nodeB.QuerySpecificBalance(test1AddrChainB, initialization.TerraDenom)
	s.Require().NoError(err)
	// Tx will only cost 10uluna on chain B as gas
	s.Require().Equal(newTerraBalance.Amount, terraBalance.Amount.Sub(sdkmath.NewInt(2)))
}

func (s *IntegrationTestSuite) TestFeeTaxMultipleDenoms() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	transferAmount := sdkmath.NewInt(100000000)
	transferCoin1 := sdk.NewCoin(initialization.TerraDenom, transferAmount)
	transferCoin2 := sdk.NewCoin(initialization.UsdDenom, transferAmount)

	test1Addr := node.CreateWallet("test1-multiple-fees")
	test2Addr := node.CreateWallet("test2-multiple-fees")

	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")

	node.BankSend(transferCoin1.String(), validatorAddr, test1Addr, []string{initialization.TerraDenom})
	node.BankSend(transferCoin1.String(), validatorAddr, test1Addr, []string{initialization.TerraDenom})

	node.BankSend(transferCoin2.String(), validatorAddr, test1Addr, []string{initialization.TerraDenom})

	taxByTerraDenom := initialization.TaxRate.MulInt(transferAmount.QuoRaw(2)).TruncateInt()
	feeByTerraDenom := sdk.NewCoin(initialization.TerraDenom, taxByTerraDenom.AddRaw(1)) // 1 uluna to pay for ante handler gas
	taxByUsdDenom := initialization.TaxRate.MulInt(transferAmount.QuoRaw(2)).
		// convert terra denom to usd denom base on gas price
		Mul(initialization.UsdGasPrice.Quo(initialization.TerraGasPrice)).
		TruncateInt()
	feeByUsdDenom := sdk.NewCoin(initialization.UsdDenom, taxByUsdDenom.AddRaw(1)) // 1 uusd to pay for post handler gas

	node.BankSend(transferCoin1.String(), test1Addr, test2Addr, []string{}, feeByTerraDenom, feeByUsdDenom)

	test1AddrTerraBalance, err := node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)
	test1AddrUsdBalance, err := node.QuerySpecificBalance(test1Addr, initialization.UsdDenom)
	s.Require().NoError(err)
	test2AddrTerraBalance, err := node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	// Final denom will be paid by both uluna and uusd
	s.Require().Equal(test2AddrTerraBalance, transferCoin1)
	s.Require().Equal(test1AddrTerraBalance, transferCoin1.Sub(feeByTerraDenom))
	s.Require().Equal(test1AddrUsdBalance, transferCoin2.Sub(feeByUsdDenom))
}

func (s *IntegrationTestSuite) TestFeeTaxForwardWasm() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	transferAmount1 := sdkmath.NewInt(700000000)
	transferCoin1 := sdk.NewCoin(initialization.TerraDenom, transferAmount1)

	test1Addr := node.CreateWallet("test1-forward-wasm")
	test2Addr := node.CreateWallet("test2-forward-wasm")

	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)

	node.BankSend(transferCoin1.String(), validatorAddr, test1Addr, []string{initialization.TerraDenom})

	// Test 1: User ----(execute contract with funds)---> Contract ---(execute bank send msg)---> Another User
	node.StoreWasmCode("forwarder.wasm", initialization.ValidatorWalletName)

	chain.LatestCodeID = int(node.QueryLatestWasmCodeID())
	node.InstantiateWasmContract(
		strconv.Itoa(chain.LatestCodeID),
		`{}`, "",
		initialization.ValidatorWalletName, []string{}, sdk.NewCoin(initialization.TerraDenom, sdk.NewInt(2)))

	contracts, err := node.QueryContractsFromID(chain.LatestCodeID)
	s.NoError(err)
	s.Len(contracts, 1, "Wrong number of contracts for the counter")
	contract1Addr := contracts[0]

	transferAmount2 := sdkmath.NewInt(100000000)
	transferCoin2 := sdk.NewCoin(initialization.TerraDenom, transferAmount2)
	node.WasmExecute(
		contract1Addr,
		fmt.Sprintf(`{"forward": {"recipient": "%s"}}`, test2Addr),
		transferCoin2.String(),
		"test1-forward-wasm",
		[]string{initialization.TerraDenom},
	)

	test1AddrBalance, err := node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)
	test2AddrBalance, err := node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	s.Require().Equal(test2AddrBalance, transferCoin2)
	s.Require().Equal(test1AddrBalance.Amount, transferAmount1.Sub(transferAmount2).
		// User 1 will paid 2 times on taxes due to the contract execute bank send msg
		// 2uluna will be used for gas
		Sub(initialization.TaxRate.MulInt(transferAmount2.MulRaw(2)).TruncateInt()).SubRaw(2))

	// Test 2: Contract trigger another contract's execute msg
	node.InstantiateWasmContract(
		strconv.Itoa(chain.LatestCodeID),
		`{}`, "",
		initialization.ValidatorWalletName, []string{}, sdk.NewCoin(initialization.TerraDenom, sdk.NewInt(2)))

	contracts, err = node.QueryContractsFromID(chain.LatestCodeID)
	s.NoError(err)
	s.Len(contracts, 2, "Wrong number of contracts for the counter")
	contract2Addr := contracts[1]

	node.WasmExecute(
		contract1Addr,
		fmt.Sprintf(`{"forward_to_contract": {"contract": "%s", "recipient": "%s"}}`, contract2Addr, test2Addr),
		transferCoin2.String(),
		"test1-forward-wasm",
		[]string{initialization.TerraDenom},
	)

	newTest1AddrBalance, err := node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)
	newTest2AddrBalance, err := node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	s.Require().Equal(newTest2AddrBalance, test2AddrBalance.Add(transferCoin2))
	s.Require().Equal(newTest1AddrBalance.Amount, test1AddrBalance.Amount.Sub(transferAmount2).
		// User 1 will paid 3 times on taxes: execute contract1 msg, contract 1 execute contract 2 msg, contract 2 execute bank msg
		// 2uluna will be used for gas
		Sub(initialization.TaxRate.MulInt(transferAmount2.MulRaw(3)).TruncateInt()).SubRaw(2))

	// Test 3: Error when forward tx
	test1AddrBalance = newTest1AddrBalance
	test2AddrBalance = newTest2AddrBalance

	node.WasmExecuteError(
		contract1Addr,
		fmt.Sprintf(`{"forward_to_cause_error": {"contract": "%s"}}`, contract2Addr),
		transferCoin2.String(),
		"test1-forward-wasm",
		[]string{initialization.TerraDenom},
	)

	newTest1AddrBalance, err = node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)
	newTest2AddrBalance, err = node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	s.Require().Equal(newTest2AddrBalance, test2AddrBalance)
	// Transfer amount will we return
	s.Require().Equal(newTest1AddrBalance.Amount, test1AddrBalance.Amount)
}

func (s *IntegrationTestSuite) TestFeeTaxNotAcceptDenom() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	transferAmount1 := sdkmath.NewInt(500000000)
	transferCoin1TerraDenom := sdk.NewCoin(initialization.TerraDenom, transferAmount1)
	transferCoin1NonValueDenom := sdk.NewCoin(initialization.NonValueDenom, transferAmount1)

	test1Addr := node.CreateWallet("test1-not-accept-denom")
	test2Addr := node.CreateWallet("test2-not-accept-denom")

	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)

	node.BankSend(transferCoin1TerraDenom.String(), validatorAddr, test1Addr, []string{initialization.TerraDenom})

	node.BankSend(transferCoin1NonValueDenom.String(), validatorAddr, test1Addr, []string{}, sdk.NewCoin(initialization.TerraDenom, sdkmath.NewInt(10)))

	// Test 1: Try to pay tx fee with non-value denom
	transferAmount2 := sdkmath.NewInt(100000000)
	transferCoin2 := sdk.NewCoin(initialization.TerraDenom, transferAmount2)

	gasLimit := transferAmount2.MulRaw(initialization.E10).String()
	fees := sdk.NewCoins(sdk.NewCoin(initialization.NonValueDenom, transferAmount2))
	err = fmt.Errorf("can't find coin that matches")
	// Tx will cause error cause it doesn't have the correct fees to pay for tx
	node.BankSendError(transferCoin2.String(), test1Addr, test2Addr, "test1-not-accept-denom", gasLimit, fees, err.Error())

	// Test 2: Try to trick the chain by paying with both uluna and non-value denom

	feeTerra := initialization.TaxRate.MulInt(transferAmount2).TruncateInt().AddRaw(2)
	feeTerraCoin := sdk.NewCoin(initialization.TerraDenom, feeTerra)
	fees = sdk.NewCoins(sdk.NewCoin(initialization.NonValueDenom, transferAmount2), feeTerraCoin)

	// At this time, the tx will ignore non-value denom and only deduct the uluna
	node.BankSendWithWallet(transferCoin2.String(), test1Addr, test2Addr, "test1-not-accept-denom", []string{}, fees...)

	balanceTest1Terra, err := node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)
	balanceTest1NonValueDenom, err := node.QuerySpecificBalance(test1Addr, initialization.NonValueDenom)
	s.Require().NoError(err)

	s.Require().Equal(balanceTest1Terra.Amount, transferAmount1.Sub(transferAmount2).Sub(feeTerra))
	s.Require().Equal(balanceTest1NonValueDenom.Amount, transferAmount1)
}
