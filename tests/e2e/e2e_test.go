package e2e

import (
	"fmt"
	"strconv"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v3/tests/e2e/initialization"
	coreassets "github.com/classic-terra/core/v3/types/assets"
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
		initialization.ValidatorWalletName)

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
	s.Require().NotEqual(test1Addr, "")

	// Test 1: banktypes.MsgSend
	// burn tax with bank send
	node.BankSend(transferCoin1.String(), validatorAddr, test1Addr)

	subAmount := transferAmount1.Add(initialization.BurnTaxRate.MulInt(transferAmount1).TruncateInt())

	decremented := validatorBalance.Sub(sdk.NewCoin(initialization.TerraDenom, subAmount))
	newValidatorBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	balanceTest1, err := node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	s.Require().Equal(balanceTest1.Amount, transferAmount1)
	s.Require().Equal(newValidatorBalance, decremented)

	// Test 2: try bank send with grant
	test2Addr := node.CreateWallet("test2")
	s.Require().NotEqual(test2Addr, "")
	transferAmount2 := sdkmath.NewInt(10000000)
	transferCoin2 := sdk.NewCoin(initialization.TerraDenom, transferAmount2)

	node.BankSend(transferCoin2.String(), validatorAddr, test2Addr)
	node.GrantAddress(test2Addr, test1Addr, transferCoin2.String(), "test2")

	validatorBalance, err = node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	node.BankSendFeeGrantWithWallet(transferCoin2.String(), test1Addr, validatorAddr, test2Addr, "test1")

	newValidatorBalance, err = node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	balanceTest1, err = node.QuerySpecificBalance(test1Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	balanceTest2, err := node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	s.Require().Equal(balanceTest1.Amount, transferAmount1.Sub(transferAmount2))
	s.Require().Equal(newValidatorBalance, validatorBalance.Add(transferCoin2))
	s.Require().Equal(balanceTest2.Amount, transferAmount2.Sub(initialization.BurnTaxRate.MulInt(transferAmount2).TruncateInt()))

	// Test 3: banktypes.MsgMultiSend
	validatorBalance, err = node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	node.BankMultiSend(transferCoin1.String(), false, validatorAddr, test1Addr, test2Addr)

	newValidatorBalance, err = node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	totalTransferAmount := transferAmount1.Mul(sdk.NewInt(2))
	subAmount = totalTransferAmount.Add(initialization.BurnTaxRate.MulInt(totalTransferAmount).TruncateInt())
	s.Require().Equal(newValidatorBalance, validatorBalance.Sub(sdk.NewCoin(initialization.TerraDenom, subAmount)))
}

func (s *IntegrationTestSuite) TestAuthz() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	transferAmount1 := sdkmath.NewInt(20000000)
	transferCoin1 := sdk.NewCoin(initialization.TerraDenom, transferAmount1)
	test1WalletName := "authz1"
	test2WalletName := "authz2"
	test1Addr := node.CreateWallet(test1WalletName)
	test2Addr := node.CreateWallet(test2WalletName)
	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")
	validatorBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	node.GrantBankSend(test1Addr, transferCoin1.String(), "val")
	node.BankSendWithWallet(transferCoin1.String(), validatorAddr, test2Addr, test1WalletName)

	newValidatorBalance, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)

	balanceTest2, err := node.QuerySpecificBalance(test2Addr, initialization.TerraDenom)
	s.Require().NoError(err)

	s.Require().Equal(transferAmount1, balanceTest2.Amount)
	s.Require().Equal(validatorBalance.Amount.Sub(transferAmount1).Sub(initialization.BurnTaxRate.MulInt(transferAmount1).TruncateInt()), newValidatorBalance.Amount)
}

func (s *IntegrationTestSuite) TestFeeTaxWasm() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)

	testAddr := node.CreateWallet("test")
	transferAmount := sdkmath.NewInt(100000000)
	transferCoin := sdk.NewCoin(initialization.TerraDenom, transferAmount)
	node.BankSend(fmt.Sprintf("%suluna", transferAmount.Mul(sdk.NewInt(4))), initialization.ValidatorWalletName, testAddr)
	node.StoreWasmCode("counter.wasm", initialization.ValidatorWalletName)
	chain.LatestCodeID = int(node.QueryLatestWasmCodeID())
	// instantiate contract and transfer 100000000uluna
	node.InstantiateWasmContract(
		strconv.Itoa(chain.LatestCodeID),
		`{"count": "0"}`, transferCoin.String(),
		"test")

	contracts, err := node.QueryContractsFromID(chain.LatestCodeID)
	s.Require().NoError(err)
	s.Require().Len(contracts, 1, "Wrong number of contracts for the counter")

	balance1, err := node.QuerySpecificBalance(testAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	// 400000000 - 100000000 - 100000000 * TaxRate = 300000000 - 10000000 * TaxRate
	// taxAmount := initialization.BurnTaxRate.MulInt(transferAmount).TruncateInt()
	// s.Require().Equal(balance1.Amount, transferAmount.Mul(sdk.NewInt(3)).Sub(taxAmount))
	// no longer taxed
	s.Require().Equal(balance1.Amount, transferAmount.Mul(sdk.NewInt(3)))

	stabilityFee := sdk.NewDecWithPrec(2, 2).MulInt(transferAmount)

	node.Instantiate2WasmContract(
		strconv.Itoa(chain.LatestCodeID),
		`{"count": "0"}`, "salt",
		transferCoin.String(),
		fmt.Sprintf("%duluna", stabilityFee), "300000", "test")

	contracts, err = node.QueryContractsFromID(chain.LatestCodeID)
	s.Require().NoError(err)
	s.Require().Len(contracts, 2, "Wrong number of contracts for the counter")

	balance2, err := node.QuerySpecificBalance(testAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	// balance1 - 100000000 - 100000000 * TaxRate
	// taxAmount = initialization.BurnTaxRate.MulInt(transferAmount).TruncateInt()
	// s.Require().Equal(balance2.Amount, balance1.Amount.Sub(transferAmount).Sub(taxAmount))
	// no longer taxed
	s.Require().Equal(balance2.Amount, balance1.Amount.Sub(transferAmount))

	contractAddr := contracts[0]
	node.WasmExecute(contractAddr, `{"donate": {}}`, transferCoin.String(), fmt.Sprintf("%duluna", stabilityFee), "test")

	balance3, err := node.QuerySpecificBalance(testAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	// balance2 - 100000000 - 100000000 * TaxRate
	// taxAmount = initialization.BurnTaxRate.MulInt(transferAmount).TruncateInt()
	// s.Require().Equal(balance3.Amount, balance2.Amount.Sub(transferAmount).Sub(taxAmount))
	// no longer taxed
	s.Require().Equal(balance3.Amount, balance2.Amount.Sub(transferAmount))
}

func (s *IntegrationTestSuite) TestMarketSwap() {
	chain := s.configurer.GetChainConfig(0)
	node, err := chain.GetDefaultNode()
	s.Require().NoError(err)
	// Ensure the app has produced at least one block before making gRPC queries
	chain.WaitForNumHeights(1)
	node.LogActionF("STEP 0: TestMarketSwap start — chain produced at least one block")

	// Ensure validator address and initial balances
	validatorAddr := node.GetWallet(initialization.ValidatorWalletName)
	s.Require().NotEqual(validatorAddr, "")

	// Ensure each validator delegates feeder consent to its signer account before oracle txs
	for _, v := range chain.NodeConfigs {
		if !v.IsValidator {
			continue
		}
		feeder := v.GetWallet(initialization.ValidatorWalletName)
		s.Require().NotEmpty(feeder)
		v.DelegateOracleFeedConsent(feeder)
	}
	node.LogActionF("STEP 2: delegated feeder consent for all validators; waiting 1 block for inclusion")
	chain.WaitForNumHeights(1)

	// Minimal oracle flow across all validators:
	// P: prevote; wait boundary; P+1: vote(prev P) then +1 block and prevote; then assert exchange rates and swap.
	votePeriod := node.QueryOracleVotePeriod()
	node.LogActionF("STEP 3: oracle votePeriod=%d", votePeriod)
	rates := "1000.0ukrw,1.0uusd,1.0usdr,1.0UST"
	saltP := "0101"
	// Anchor to next period start P
	curH0, err := node.QueryCurrentHeight()
	s.Require().NoError(err)
	startP := ((curH0 / votePeriod) + 1) * votePeriod
	node.LogActionF("STEP 4: anchoring to startP=%d (curH0=%d)", startP, curH0)
	chain.WaitUntilHeight(startP)
	// Prevote in P for all validators
	node.LogActionF("STEP 5: submitting prevote for all validators in P")
	for _, v := range chain.NodeConfigs {
		if v.IsValidator {
			v.SubmitOracleAggregatePrevote(saltP, rates)
		}
	}
	chain.WaitForNumHeights(1) // ensure inclusion before boundary handling
	// Wait to boundary P+1 with a small safe offset
	boundaryP1 := startP + votePeriod
	safeOffset := int64(1)
	node.LogActionF("STEP 6: waiting until boundary P+1 (%d) + offset %d", boundaryP1, safeOffset)
	chain.WaitUntilHeight(boundaryP1 + safeOffset)
	// Vote reveal for P, then wait 1 block and prevote for continuity
	saltP1 := "0202"
	node.LogActionF("STEP 7: submitting vote(reveal P) for all validators")
	for _, v := range chain.NodeConfigs {
		if v.IsValidator {
			v.SubmitOracleAggregateVote(saltP, rates)
		}
	}
	// Record the period in which we revealed votes, then wait until the next boundary so rates are tallied
	curAfterVote, err := node.QueryCurrentHeight()
	s.Require().NoError(err)
	revealedPeriod := curAfterVote / votePeriod

	chain.WaitForNumHeights(1)
	for _, v := range chain.NodeConfigs {
		if v.IsValidator {
			v.SubmitOracleAggregatePrevote(saltP1, rates)
		}
	}

	// wait for next period, then vote again
	boundaryP2 := boundaryP1 + votePeriod
	saltP2 := "0303"
	chain.WaitUntilHeight(boundaryP2)
	for _, v := range chain.NodeConfigs {
		if v.IsValidator {
			v.SubmitOracleAggregateVote(saltP1, rates)
		}
	}
	chain.WaitForNumHeights(1)
	for _, v := range chain.NodeConfigs {
		if v.IsValidator {
			v.SubmitOracleAggregatePrevote(saltP2, rates)
		}
	}

	// Verify exchange rates reflect our submitted rates
	node.LogActionF("STEP 9: verifying exchange rates updated")
	got := node.QueryOracleExchangeRates()
	expected := map[string]string{"ukrw": "1000.0", "uusd": "1.0", "usdr": "1.0"}
	for denom, exp := range expected {
		val, ok := got[denom]
		s.Require().Truef(ok, "missing exchange rate for %s", denom)
		expDec, err := sdk.NewDecFromStr(exp)
		s.Require().NoError(err)
		gotDec, err := sdk.NewDecFromStr(val)
		s.Require().NoError(err)
		s.Require().Truef(expDec.Equal(gotDec), "exchange rate mismatch for %s: expected %s got %s", denom, expDec.String(), gotDec.String())
	}

	// Seed market module account with liquidity to avoid insufficient pool errors
	offer := "1000000uluna"
	marketModule := node.GetModuleAccountAddress("market")
	marketAccumulator := node.GetModuleAccountAddress("market_accumulator")
	// capture pre-seeding balances in case module accounts are pre-funded via genesis
	preMarketBal, err := node.QuerySpecificBalance(marketModule, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	preAccBal, err := node.QuerySpecificBalance(marketAccumulator, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	node.LogActionF("STEP 10: seeding market=%s and accumulator=%s with 10000000uusd each (pre: market=%s acc=%s)", marketModule, marketAccumulator, preMarketBal.Amount.String(), preAccBal.Amount.String())
	node.BankSend("10000000uusd", validatorAddr, marketModule)
	node.BankSend("10000000uusd", validatorAddr, marketAccumulator)
	chain.WaitForNumHeights(1)

	// query balance of market and accumulator
	marketBalance, err := node.QuerySpecificBalance(marketModule, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	accumulatorBalance, err := node.QuerySpecificBalance(marketAccumulator, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	node.LogActionF("STEP 10: module balances after seeding: market uusd=%s (pre=%s) accumulator uusd=%s (pre=%s)", marketBalance.Amount.String(), preMarketBal.Amount.String(), accumulatorBalance.Amount.String(), preAccBal.Amount.String())

	// After seeding, if a new period started, reveal previous prevote and submit a new prevote, then wait to boundary
	hBeforeSwap, err := node.QueryCurrentHeight()
	s.Require().NoError(err)
	curPeriod := hBeforeSwap / votePeriod
	if curPeriod > revealedPeriod {
		node.LogActionF("STEP 10a: new period detected after seeding (cur=%d > revealed=%d): submitting vote+prevote again", curPeriod, revealedPeriod)
		// Reveal the prevote from last period (saltP1), then prevote a new one
		for _, v := range chain.NodeConfigs {
			if v.IsValidator {
				v.SubmitOracleAggregateVote(saltP1, rates)
			}
		}
		chain.WaitForNumHeights(1)
		saltP2 := "0303"
		for _, v := range chain.NodeConfigs {
			if v.IsValidator {
				v.SubmitOracleAggregatePrevote(saltP2, rates)
			}
		}
		revealedPeriod = curPeriod
	}

	// query balance of market and accumulator
	marketBalance, err = node.QuerySpecificBalance(marketModule, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	accumulatorBalance, err = node.QuerySpecificBalance(marketAccumulator, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	node.LogActionF("STEP 10b: module balances after seeding: market uusd=%s (pre=%s) accumulator uusd=%s (pre=%s)", marketBalance.Amount.String(), preMarketBal.Amount.String(), accumulatorBalance.Amount.String(), preAccBal.Amount.String())

	preLuna, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	preUSD, err := node.QuerySpecificBalance(validatorAddr, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	node.LogActionF("STEP 10b: captured initial balances uluna=%s uusd=%s", preLuna.Amount.String(), preUSD.Amount.String())

	// Ensure there is sufficient liquidity. Module accounts are pre-funded at genesis; require at least 10,000,000 uusd each.
	minLiquidity := sdk.NewInt(10000000)
	s.Require().True(marketBalance.Amount.GTE(minLiquidity), "market balance should be >= %s uusd", minLiquidity.String())
	s.Require().True(accumulatorBalance.Amount.GTE(minLiquidity), "accumulator balance should be >= %s uusd", minLiquidity.String())

	node.LogActionF("STEP 10b: performing market swap offer=%s -> %s", offer, coreassets.MicroUSDDenom)
	node.MarketSwap(offer, coreassets.MicroUSDDenom, initialization.ValidatorWalletName)

	// Verify balances changed appropriately
	postLuna, err := node.QuerySpecificBalance(validatorAddr, initialization.TerraDenom)
	s.Require().NoError(err)
	postUSD, err := node.QuerySpecificBalance(validatorAddr, coreassets.MicroUSDDenom)
	s.Require().NoError(err)

	node.LogActionF("STEP 11: post-swap balances uluna=%s uusd=%s (pre uluna=%s uusd=%s)", postLuna.Amount.String(), postUSD.Amount.String(), preLuna.Amount.String(), preUSD.Amount.String())
	s.Require().True(postUSD.Amount.GT(preUSD.Amount), "uusd balance should increase after swap: before=%s after=%s", preUSD.Amount, postUSD.Amount)
	s.Require().True(postLuna.Amount.LT(preLuna.Amount), "uluna balance should decrease after swap: before=%s after=%s", preLuna.Amount, postLuna.Amount)

	// ----- Epoch boundary checks (epoch length = 50 blocks) -----
	epochLen := int64(50)
	curHForEpoch, err := node.QueryCurrentHeight()
	s.Require().NoError(err)
	// Use the NEXT epoch boundary explicitly
	nextEpochStart := ((curHForEpoch / epochLen) + 1) * epochLen
	h46 := nextEpochStart - 4
	node.LogActionF("STEP 11a: waiting for epoch height %d (nextEpochStart=%d - 4) to capture balances before burn->refill", h46, nextEpochStart)
	chain.WaitUntilHeight(h46)

	marketAt46, err := node.QuerySpecificBalance(marketModule, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	accAt46, err := node.QuerySpecificBalance(marketAccumulator, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	node.LogActionF("STEP 11a: at h=%d market=%s accumulator=%s", h46, marketAt46.Amount.String(), accAt46.Amount.String())

	h51 := nextEpochStart + 1 // cross the epoch boundary (burn/refill should have executed)
	node.LogActionF("STEP 11b: waiting for epoch height %d (nextEpochStart=%d + 1) to verify accumulator drained to market", h51, nextEpochStart)
	chain.WaitUntilHeight(h51)
	// Allow one block after boundary for end-block processing to be reflected
	chain.WaitForNumHeights(1)

	marketAt51, err := node.QuerySpecificBalance(marketModule, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	accAt51, err := node.QuerySpecificBalance(marketAccumulator, coreassets.MicroUSDDenom)
	s.Require().NoError(err)
	node.LogActionF("STEP 11b: at h=%d market=%s accumulator=%s (prev h=%d acc=%s)", h51, marketAt51.Amount.String(), accAt51.Amount.String(), h46, accAt46.Amount.String())

	// Expect accumulator to be empty and market to equal previous accumulator balance
	s.Require().True(accAt51.Amount.IsZero(), "accumulator should be empty after epoch rollover")
	s.Require().True(marketAt51.Amount.Equal(accAt46.Amount), "market balance should equal previous accumulator balance: got market=%s want=%s", marketAt51.Amount.String(), accAt46.Amount.String())

}
