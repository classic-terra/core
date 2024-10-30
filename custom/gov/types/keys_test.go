package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var addr = sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

func TestProposalKeys(t *testing.T) {
	// key proposal
	key := govtypes.ProposalKey(1)
	proposalID := govtypes.SplitProposalKey(key)
	require.Equal(t, proposalID, uint64(1))

	// key active proposal queue
	now := time.Now()
	key = govtypes.ActiveProposalQueueKey(3, now)
	proposalID, expTime := govtypes.SplitActiveProposalQueueKey(key)
	require.Equal(t, proposalID, uint64(3))
	require.True(t, now.Equal(expTime))

	// key inactive proposal queue
	key = govtypes.InactiveProposalQueueKey(3, now)
	proposalID, expTime = govtypes.SplitInactiveProposalQueueKey(key)
	require.Equal(t, proposalID, uint64(3))
	require.True(t, now.Equal(expTime))

	// invalid key
	require.Panics(t, func() { govtypes.SplitProposalKey([]byte("test")) })
	require.Panics(t, func() { govtypes.SplitInactiveProposalQueueKey([]byte("test")) })
}

func TestDepositKeys(t *testing.T) {
	key := govtypes.DepositsKey(2)
	proposalID := govtypes.SplitProposalKey(key)
	require.Equal(t, proposalID, uint64(2))

	key = govtypes.DepositKey(2, addr)
	proposalID, depositorAddr := govtypes.SplitKeyDeposit(key)
	require.Equal(t, proposalID, uint64(2))
	require.Equal(t, addr, depositorAddr)
}

func TestVoteKeys(t *testing.T) {
	key := govtypes.VotesKey(2)
	proposalID := govtypes.SplitProposalKey(key)
	require.Equal(t, proposalID, uint64(2))

	key = govtypes.VoteKey(2, addr)
	proposalID, voterAddr := govtypes.SplitKeyDeposit(key)
	require.Equal(t, proposalID, uint64(2))
	require.Equal(t, addr, voterAddr)
}

func TestTotalDepositKeys(t *testing.T) {
	key := TotalDepositKey(2)
	proposalID := govtypes.SplitProposalKey(key)
	require.Equal(t, proposalID, uint64(2))
}
