package initialization

import (
	"fmt"
	"os"

	tmrand "github.com/tendermint/tendermint/libs/rand"
	// "github.com/tendermint/tendermint/p2p"
	// "github.com/tendermint/tendermint/privval"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkcrypto "github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	// cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/classic-terra/core/v2/app/params"
)

const (
	KeyringPassphrase = "testpassphrase"
	KeyringAppName    = "testnet"
)

var (
	EncodingConfig params.EncodingConfig
	Cdc            codec.Codec
	TxConfig       client.TxConfig
)

type ChainMeta struct {
	DataDir string
	Id      string
}

// type Node struct {
// 	Moniker       string
// 	Mnemonic      string
// 	PublicAddress string
// 	PublicKey     cryptotypes.PubKey
// 	PrivateKey    cryptotypes.PrivKey
// }

type Chain struct {
	ChainMeta  ChainMeta
	Validators []*Validator
	// Node            []*Node
	GenesisAccounts        []*Account
	GenesisVestingAccounts map[string]sdk.AccAddress
}

func NewChain() (*Chain, error) {
	tmpDir, err := os.MkdirTemp("", "terra-e2e-testnet-")
	if err != nil {
		return nil, err
	}

	return &Chain{
		ChainMeta: ChainMeta{
			Id:      "chain-" + tmrand.Str(6),
			DataDir: tmpDir,
		},
	}, nil
}

func (c *Chain) CreateAndInitValidators(count int) error {
	for i := 0; i < count; i++ {
		node := c.createValidator(i)

		// generate genesis files
		if err := node.init(); err != nil {
			return err
		}

		c.Validators = append(c.Validators, node)

		// create keys
		if err := node.createKey("val"); err != nil {
			return err
		}
		if err := node.createNodeKey(); err != nil {
			return err
		}
		if err := node.createConsensusKey(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Chain) createValidator(index int) *Validator {
	return &Validator{
		Chain:   c,
		Index:   index,
		Moniker: fmt.Sprintf("%s-terra-%d", c.ChainMeta.Id, index),
	}
}

func (c *Chain) configDir() string {
	return fmt.Sprintf("%s/%s", c.ChainMeta.DataDir, c.ChainMeta.Id)
}

func (c *Chain) AddAccountFromMnemonic(counts int) error {
	val0ConfigDir := c.Validators[0].ConfigDir()
	kb, err := keyring.New(KeyringAppName, keyring.BackendTest, val0ConfigDir, nil, Cdc)
	if err != nil {
		return err
	}

	keyringAlgos, _ := kb.SupportedAlgorithms()
	algo, err := keyring.NewSigningAlgoFromString(string(hd.Secp256k1Type), keyringAlgos)
	if err != nil {
		return err
	}

	for i := 0; i < counts; i++ {
		name := fmt.Sprintf("acct-%d", i)
		mnemonic, err := CreateMnemonic()
		if err != nil {
			return err
		}
		info, err := kb.NewAccount(name, mnemonic, "", sdk.FullFundraiserPath, algo)
		if err != nil {
			return err
		}

		privKeyArmor, err := kb.ExportPrivKeyArmor(name, KeyringPassphrase)
		if err != nil {
			return err
		}

		privKey, _, err := sdkcrypto.UnarmorDecryptPrivKey(privKeyArmor, KeyringPassphrase)
		if err != nil {
			return err
		}
		acct := Account{}
		acct.KeyInfo = info
		acct.Mnemonic = mnemonic
		acct.PrivateKey = privKey
		c.GenesisAccounts = append(c.GenesisAccounts, &acct)
	}

	return nil
}
