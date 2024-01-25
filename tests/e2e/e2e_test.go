package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v2/tests/e2e/initialization"
	treasurytypes "github.com/classic-terra/core/v2/x/treasury/types"
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
	if s.skipIBC {
		s.T().Skip("Skipping Packet Forward Middleware tests")
	}
	chainA := s.configurer.GetChainConfig(0)
	nodeA, err := chainA.GetDefaultNode()
	s.NoError(err)
	amountSend := sdk.NewInt(1000_000)

	a := nodeA.CreateWallet("a")

	validatorAddr := nodeA.GetWallet(initialization.ValidatorWalletName)

	balanceOld, err := nodeA.QueryBalances(validatorAddr)
	s.Require().NoError(err)
	s.T().Logf("check baaaa val old:%v", balanceOld)

	found, junoAmoutOld := balanceOld.Find("juno")
	s.True(found)

	nodeA.BankSend(fmt.Sprintf("%djuno", amountSend.Int64()), validatorAddr, a)
	// wait 7s
	time.Sleep(7 * time.Second)

	balanceReceiver, err := nodeA.QueryBalances(a)
	s.Require().NoError(err)
	s.T().Logf("check baaaa:%v", balanceReceiver)

	balanceNew, err := nodeA.QueryBalances(validatorAddr)
	s.Require().NoError(err)
	s.T().Logf("check baaaa val new:%v", balanceNew)

	found, junoAmoutNew := balanceNew.Find("juno")
	s.True(found)
	feeUsed := junoAmoutOld.Amount.Sub(junoAmoutNew.Amount).Sub(amountSend)
	s.T().Logf("fees used:%v", feeUsed)

	s.Eventually(func() bool {
		feeUsed := junoAmoutOld.Amount.Sub(junoAmoutNew.Amount).Sub(amountSend)
		s.T().Logf("fees used:%v", feeUsed)
		return feeUsed.Equal(treasurytypes.DefaultTaxRate.Mul(sdk.NewDecFromInt(amountSend)).RoundInt())
	},
		15*time.Second,
		10*time.Millisecond,
	)
}
