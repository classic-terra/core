package helpers

import (
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/stretchr/testify/require"
)

func (s *AppTestSuite) TestABCI_Proposal_HappyPath() {
	pool := mempool.NewSenderNonceMempool()
	var txs [][]byte
	// Create 5 oracle and 5 non-oracle txs in mixed order
	for i := uint64(0); i < 10; i++ {
		txs = append(txs, s.createNumberedTx(i, i%2 == 0))
	}

	txDecoder := s.App.GetTxConfig().TxDecoder()

	reqCheckTx := abci.RequestCheckTx{
		Tx:   txs[0],
		Type: abci.CheckTxType_New,
	}
	s.App.CheckTx(reqCheckTx)

	for i := uint64(0); i < 10; i++ {
		decodedTx, err := txDecoder(txs[i])
		require.NoError(s.T(), err)
		err = pool.Insert(s.Ctx, decodedTx)
		require.NoError(s.T(), err)
	}
	reqPrepareProposal := abci.RequestPrepareProposal{
		MaxTxBytes: 1000,
		Height:     1,
		Txs:        txs[0:5],
	}
	resPrepareProposal := s.App.PrepareProposal(reqPrepareProposal)
	require.Equal(s.T(), 5, len(resPrepareProposal.Txs))

	// 3 oracleTx of first 5 tx should be placed in front
	s.verifyOraclePriority(resPrepareProposal.Txs[0:5], 3)

	reqProcessProposal := abci.RequestProcessProposal{
		Txs:    resPrepareProposal.Txs,
		Height: reqPrepareProposal.Height,
	}
	resProcessProposal := s.App.ProcessProposal(reqProcessProposal)
	require.Equal(s.T(), abci.ResponseProcessProposal_ACCEPT, resProcessProposal.Status)
}
