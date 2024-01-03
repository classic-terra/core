package initialization

import (
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

type ChainMeta struct {
	DataDir string
	Id      string
}

type Validator struct {
	Chain            *Chain
	Index            int
	Moniker          string
	Mnemonic         string
	PublicAddress    string
	PublicKey        cryptotypes.PubKey
	PrivateKey       cryptotypes.PrivKey
	ConsensusKey     privval.FilePVKey
	ConsensusPrivKey cryptotypes.PrivKey
	NodeKey          p2p.NodeKey
}

type Node struct {
	Moniker       string //nolint:unused
	Mnemonic      string
	PublicAddress string
	PublicKey     cryptotypes.PubKey
	PrivateKey    cryptotypes.PrivKey
}

type Chain struct {
	ChainMeta ChainMeta
	Validator []*Validator
	Node	  []*Node
}