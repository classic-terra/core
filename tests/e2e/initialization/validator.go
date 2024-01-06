package initialization

import (
	"cosmossdk.io/math"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	tmcfg "github.com/tendermint/tendermint/config"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"

	sdkcrypto "github.com/cosmos/cosmos-sdk/crypto"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	terraapp "github.com/classic-terra/core/v2/app"
)

type Validator struct {
	Chain            *Chain
	Index            int
	Moniker          string
	Mnemonic         string
	KeyInfo          *keyring.Record
	PublicAddress    string
	PublicKey        cryptotypes.PubKey
	PrivateKey       cryptotypes.PrivKey
	ConsensusKey     privval.FilePVKey
	ConsensusPrivKey cryptotypes.PrivKey
	NodeKey          p2p.NodeKey
}

type Account struct {
	Moniker    string
	Mnemonic   string
	KeyInfo    *keyring.Record
	PrivateKey cryptotypes.PrivKey
}

func (v *Validator) InstanceName() string {
	return fmt.Sprintf("%s%d", v.Moniker, v.Index)
}

func (v *Validator) ConfigDir() string {
	return fmt.Sprintf("%s/%s", v.Chain.configDir(), v.InstanceName())
}

func (v *Validator) createConfig() error {
	p := path.Join(v.ConfigDir(), "config")
	return os.MkdirAll(p, 0o755)
}

func (v *Validator) init() error {
	if err := v.createConfig(); err != nil {
		return err
	}

	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config

	config.SetRoot(v.ConfigDir())
	config.Moniker = v.Moniker

	genDoc, err := getGenDoc(v.ConfigDir())
	if err != nil {
		return err
	}

	appState, err := json.MarshalIndent(terraapp.ModuleBasics.DefaultGenesis(Cdc), "", " ")
	if err != nil {
		return fmt.Errorf("failed to JSON encode app genesis state: %w", err)
	}

	genDoc.ChainID = v.Chain.ChainMeta.Id
	genDoc.Validators = nil
	genDoc.AppState = appState

	if err = genutil.ExportGenesisFile(genDoc, config.GenesisFile()); err != nil {
		return fmt.Errorf("failed to export app genesis state: %w", err)
	}

	tmcfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)
	return nil
}

func (v *Validator) createKey(name string) error {
	mnemonic, err := CreateMnemonic()
	if err != nil {
		return err
	}

	return v.createKeyFromMnemonic(name, mnemonic)
}

func (v *Validator) createKeyFromMnemonic(name, mnemonic string) error {
	dir := v.ConfigDir()
	kb, err := keyring.New(KeyringAppName, keyring.BackendTest, dir, nil, Cdc)
	if err != nil {
		return err
	}

	keyringAlgos, _ := kb.SupportedAlgorithms()
	algo, err := keyring.NewSigningAlgoFromString(string(hd.Secp256k1Type), keyringAlgos)
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

	v.KeyInfo = info
	v.Mnemonic = mnemonic
	v.PrivateKey = privKey

	return nil
}

func (v *Validator) createNodeKey() error {
	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config

	config.SetRoot(v.ConfigDir())
	config.Moniker = v.Moniker

	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return err
	}

	v.NodeKey = *nodeKey
	return nil
}

func (v *Validator) createConsensusKey() error {
	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config

	config.SetRoot(v.ConfigDir())
	config.Moniker = v.Moniker

	pvKeyFile := config.PrivValidatorKeyFile()
	if err := tmos.EnsureDir(filepath.Dir(pvKeyFile), 0o777); err != nil {
		return err
	}

	pvStateFile := config.PrivValidatorStateFile()
	if err := tmos.EnsureDir(filepath.Dir(pvStateFile), 0o777); err != nil {
		return err
	}

	filePV := privval.LoadOrGenFilePV(pvKeyFile, pvStateFile)
	v.ConsensusKey = filePV.Key

	return nil
}

func (v *Validator) BuildCreateValidatorMsg(amount sdk.Coin) (sdk.Msg, error) {
	description := stakingtypes.NewDescription(v.Moniker, "", "", "", "")
	commissionRates := stakingtypes.CommissionRates{
		Rate:          sdk.MustNewDecFromStr("0.1"),
		MaxRate:       sdk.MustNewDecFromStr("0.2"),
		MaxChangeRate: sdk.MustNewDecFromStr("0.01"),
	}

	valPubKey, err := cryptocodec.FromTmPubKeyInterface(v.ConsensusKey.PubKey)
	if err != nil {
		return nil, err
	}
	addr, err := v.KeyInfo.GetAddress()
	if err != nil {
		return nil, err
	}
	return stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(addr),
		valPubKey,
		amount,
		description,
		commissionRates,
		math.OneInt(),
	)
}

func (v *Validator) SignMsg(msgs ...sdk.Msg) (*sdktx.Tx, error) {
	txBuilder := EncodingConfig.TxConfig.NewTxBuilder()

	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	txBuilder.SetMemo(fmt.Sprintf("%s@%s:26656", v.NodeKey.ID(), v.InstanceName()))
	txBuilder.SetFeeAmount(sdk.NewCoins())
	txBuilder.SetGasLimit(200000)

	signerData := authsigning.SignerData{
		ChainID:       v.Chain.ChainMeta.Id,
		AccountNumber: 0,
		Sequence:      0,
	}

	// For SIGN_MODE_DIRECT, calling SetSignatures calls setSignerInfos on
	// TxBuilder under the hood, and SignerInfos is needed to generate the sign
	// bytes. This is the reason for setting SetSignatures here, with a nil
	// signature.
	//
	// Note: This line is not needed for SIGN_MODE_LEGACY_AMINO, but putting it
	// also doesn't affect its generated sign bytes, so for code's simplicity
	// sake, we put it here.
	pubkey, err := v.KeyInfo.GetPubKey()
	if err != nil {
		return nil, err
	}
	sig := txsigning.SignatureV2{
		PubKey: pubkey,
		Data: &txsigning.SingleSignatureData{
			SignMode:  txsigning.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: 0,
	}

	if err := txBuilder.SetSignatures(sig); err != nil {
		return nil, err
	}

	bytesToSign, err := EncodingConfig.TxConfig.SignModeHandler().GetSignBytes(
		txsigning.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder.GetTx(),
	)
	if err != nil {
		return nil, err
	}

	sigBytes, err := v.PrivateKey.Sign(bytesToSign)
	if err != nil {
		return nil, err
	}

	sig = txsigning.SignatureV2{
		PubKey: pubkey,
		Data: &txsigning.SingleSignatureData{
			SignMode:  txsigning.SignMode_SIGN_MODE_DIRECT,
			Signature: sigBytes,
		},
		Sequence: 0,
	}
	if err := txBuilder.SetSignatures(sig); err != nil {
		return nil, err
	}

	signedTx := txBuilder.GetTx()
	bz, err := EncodingConfig.TxConfig.TxEncoder()(signedTx)
	if err != nil {
		return nil, err
	}

	return DecodeTx(bz)
}
