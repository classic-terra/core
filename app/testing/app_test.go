package helpers

import (
	"fmt"
	anteauth "github.com/classic-terra/core/v3/custom/auth/ante"
	"testing"

	oracletypes "github.com/classic-terra/core/v3/x/oracle/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/suite"
)

type AppTestSuite struct {
	KeeperTestHelper
}

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}

func (s *AppTestSuite) SetupTest() {
	s.Setup(s.T(), SimAppChainID)
}

func (s *AppTestSuite) TestPreBlocker() {
	// Create test transactions
	oracleTx := s.createTestTx([]sdk.Msg{
		&oracletypes.MsgAggregateExchangeRatePrevote{}, // Oracle message
	})

	bankTx := s.createTestTx([]sdk.Msg{
		&banktypes.MsgSend{}, // Non-oracle message
	})

	// Test cases
	testCases := []struct {
		name     string
		txs      [][]byte
		expected [][]byte
	}{
		{
			name:     "oracle tx should be first",
			txs:      [][]byte{bankTx, oracleTx},
			expected: [][]byte{oracleTx, bankTx},
		},
		{
			name:     "already ordered txs should remain ordered",
			txs:      [][]byte{oracleTx, bankTx},
			expected: [][]byte{oracleTx, bankTx},
		},
		{
			name:     "duplicate txs should be removed",
			txs:      [][]byte{bankTx, oracleTx, bankTx},
			expected: [][]byte{oracleTx, bankTx},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			req := abci.RequestPrepareProposal{
				Txs: tc.txs,
			}

			response := s.App.PreBlocker(s.Ctx, req)
			s.Require().Equal(len(tc.expected), len(response.Txs), "number of transactions should match")

			for i := range tc.expected {
				s.Require().Equal(tc.expected[i], response.Txs[i], "transaction at index %d should match", i)
			}
		})
	}
}

func (s *AppTestSuite) TestOraclePriorityProcessing() {
	testCases := []struct {
		name      string
		setup     func() [][]byte
		numTxs    int
		numOracle int
	}{
		{
			name: "small batch of mixed transactions",
			setup: func() [][]byte {
				var txs [][]byte
				// Create 5 oracle and 5 non-oracle txs in mixed order
				for i := uint64(0); i < 10; i++ {
					txs = append(txs, s.createNumberedTx(i, i%2 == 0))
				}
				return txs
			},
			numTxs:    10,
			numOracle: 5,
		},
		{
			name: "large batch of mixed transactions",
			setup: func() [][]byte {
				var txs [][]byte
				// Create 50 oracle and 50 non-oracle txs in mixed order
				for i := uint64(0); i < 100; i++ {
					txs = append(txs, s.createNumberedTx(i, i%2 == 0))
				}
				return txs
			},
			numTxs:    100,
			numOracle: 50,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			txs := tc.setup()

			// Process transactions through PreBlocker
			req := abci.RequestPrepareProposal{
				Txs: txs,
			}

			response := s.App.PreBlocker(s.Ctx, req)

			// Verify the number of transactions
			s.Require().Equal(tc.numTxs, len(response.Txs),
				"expected %d transactions, got %d", tc.numTxs, len(response.Txs))
			// Verify oracle transactions are prioritized
			s.verifyOraclePriority(response.Txs[:tc.numOracle])
		})
	}
}

// Helper function to create test transactions with sequential numbers
func (s *AppTestSuite) createNumberedTx(i uint64, isOracle bool) []byte {
	var msg sdk.Msg
	if isOracle {
		if i%3 == 0 {
			msg = &oracletypes.MsgAggregateExchangeRatePrevote{
				Hash:      fmt.Sprintf("hash_%d", i),
				Feeder:    fmt.Sprintf("feeder_%d", i),
				Validator: fmt.Sprintf("validator_%d", i),
			}
		} else {
			msg = &oracletypes.MsgAggregateExchangeRateVote{
				ExchangeRates: fmt.Sprintf("1000ukrw,1000uluna,1000usdr_%d", i),
				Feeder:        fmt.Sprintf("feeder_%d", i),
				Validator:     fmt.Sprintf("validator_%d", i),
			}
		}
	} else {
		msg = &banktypes.MsgSend{
			FromAddress: fmt.Sprintf("from_%d", i),
			ToAddress:   fmt.Sprintf("to_%d", i),
			Amount:      sdk.NewCoins(sdk.NewCoin("uluna", sdk.NewInt(int64(i)))),
		}
	}
	return s.createTestTx([]sdk.Msg{msg})
}

// Helper function to verify oracle transactions come first
func (s *AppTestSuite) verifyOraclePriority(txs [][]byte) {
	oracleFound := false
	for _, tx := range txs {
		// Decode tx to check if it's an oracle tx
		decodedTx, err := s.App.GetTxConfig().TxDecoder()(tx)
		s.Require().NoError(err)
		msgs := decodedTx.GetMsgs()
		isOracleTx := anteauth.IsOracleTx(msgs)
		if isOracleTx {
			oracleFound = true
		} else if oracleFound {
			// If we find a non-oracle tx after an oracle tx, fail the test
			s.Require().Fail("Non-oracle transaction found after oracle transaction")
		}
	}
}

// Helper method for the test suite
func (s *AppTestSuite) createTestTx(msgs []sdk.Msg) []byte {
	builder := s.App.GetTxConfig().NewTxBuilder()
	builder.SetMsgs(msgs...)

	// Add a dummy signature
	sigV2 := signing.SignatureV2{
		PubKey: nil,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: []byte("dummy_signature"),
		},
		Sequence: 0,
	}

	err := builder.SetSignatures(sigV2)
	s.Require().NoError(err)

	txBytes, err := s.App.GetTxConfig().TxEncoder()(builder.GetTx())
	s.Require().NoError(err)

	return txBytes
}
