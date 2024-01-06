package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"

	tmconfig "github.com/tendermint/tendermint/config"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/rand"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/server"
	srvconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	e2einit "github.com/classic-terra/core/v2/tests/e2e/initialization"
	e2eutil "github.com/classic-terra/core/v2/tests/e2e/util"
)

const (
	terradBinary                          = "terrad"
	txCommand                             = "tx"
	queryCommand                          = "query"
	keysCommand                           = "keys"
	terraHomePath                         = "/home/nonroot/.terra"
	photonDenom                           = "photon"
	ulunaDenom                            = "uluna"
	stakeDenom                            = "stake"
	initBalanceStr                        = "110000000000stake,100000000000000000photon,100000000000000000uluna"
	minGasPrice                           = "0.00001"
	initialGlobalFeeAmt                   = "0.00001"
	lowGlobalFeesAmt                      = "0.000001"
	highGlobalFeeAmt                      = "0.0001"
	maxTotalBypassMinFeeMsgGasUsage       = "1"
	gas                                   = 200000
	govProposalBlockBuffer                = 35
	relayerAccountIndexHermes0            = 0
	relayerAccountIndexHermes1            = 1
	numberOfEvidences                     = 10
	slashingShares                  int64 = 10000

	proposalGlobalFeeFilename           = "proposal_globalfee.json"
	proposalBypassMsgFilename           = "proposal_bypass_msg.json"
	proposalMaxTotalBypassFilename      = "proposal_max_total_bypass.json"
	proposalCommunitySpendFilename      = "proposal_community_spend.json"
	proposalAddConsumerChainFilename    = "proposal_add_consumer.json"
	proposalRemoveConsumerChainFilename = "proposal_remove_consumer.json"
	proposalLSMParamUpdateFilename      = "proposal_lsm_param_update.json"

	hermesBinary              = "hermes"
	hermesConfigWithGasPrices = "/root/.hermes/config.toml"
	hermesConfigNoGasPrices   = "/root/.hermes/config-zero.toml"
	transferChannel           = "channel-0"
)

var (
	terraConfigPath       = filepath.Join(terraHomePath, "config")
	stakingAmount         = sdk.NewInt(100000000000)
	stakingAmountCoin     = sdk.NewCoin(ulunaDenom, stakingAmount)
	tokenAmount           = sdk.NewCoin(ulunaDenom, sdk.NewInt(3300000000)) // 3,300uluna
	standardFees          = sdk.NewCoin(ulunaDenom, sdk.NewInt(330000))     // 0.33uluna
	depositAmount         = sdk.NewCoin(ulunaDenom, sdk.NewInt(330000000))  // 3,300uluna
	distModuleAddress     = authtypes.NewModuleAddress(distrtypes.ModuleName).String()
	proposalCounter       = 0
	HermesResource0Purged = false
)

type IntegrationTestSuite struct {
	suite.Suite

	tmpDirs []string
	chainA  *e2einit.Chain
	chainB  *e2einit.Chain

	dkrPool         *dockertest.Pool
	dkrNet          *dockertest.Network
	hermesResource0 *dockertest.Resource
	hermesResource1 *dockertest.Resource

	valResources map[string][]*dockertest.Resource
}

type AddressResponse struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic"`
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up e2e integration test suite...")

	var err error
	s.chainA, err = e2einit.NewChain()
	s.Require().NoError(err)

	s.chainB, err = e2einit.NewChain()
	s.Require().NoError(err)

	s.dkrPool, err = dockertest.NewPool("")
	s.Require().NoError(err)

	s.dkrNet, err = s.dkrPool.CreateNetwork(fmt.Sprintf("%s-%s-testnet", s.chainA.ChainMeta.Id, s.chainB.ChainMeta.Id))
	s.Require().NoError(err)

	s.valResources = make(map[string][]*dockertest.Resource)

	vestingMnemonic, err := e2einit.CreateMnemonic()
	s.Require().NoError(err)

	jailedValMnemonic, err := e2einit.CreateMnemonic()
	s.Require().NoError(err)

	// The boostrapping phase is as follows:
	//
	// 1. Initialize Terra validator nodes.
	// 2. Create and initialize Terra validator genesis files (both chains)
	// 3. Start both networks.
	// 4. Create and run IBC relayer (Hermes) containers.

	s.T().Logf("starting e2e infrastructure for chain A; chain-id: %s; datadir: %s", s.chainA.ChainMeta.Id, s.chainA.ChainMeta.DataDir)
	s.initNodes(s.chainA)
	s.initGenesis(s.chainA, vestingMnemonic, jailedValMnemonic)
	s.initValidatorConfigs(s.chainA)
	s.runValidators(s.chainA, 0)

	s.T().Logf("starting e2e infrastructure for chain B; chain-id: %s; datadir: %s", s.chainB.ChainMeta.Id, s.chainB.ChainMeta.DataDir)
	s.initNodes(s.chainB)
	s.initGenesis(s.chainB, vestingMnemonic, jailedValMnemonic)
	s.initValidatorConfigs(s.chainB)
	s.runValidators(s.chainB, 10)

	time.Sleep(10 * time.Second)
	s.runIBCRelayer0()
	s.runIBCRelayer1()
}

func (s *IntegrationTestSuite) initNodes(c *e2einit.Chain) {
	s.Require().NoError(c.CreateAndInitValidators(2))
	/* Adding 4 accounts to val0 local directory
	c.genesisAccounts[0]: Relayer0 Wallet
	c.genesisAccounts[1]: ICA Owner
	c.genesisAccounts[2]: Test Account 1
	c.genesisAccounts[3]: Test Account 2
	c.genesisAccounts[4]: Relayer1 Wallet
	*/
	s.Require().NoError(c.AddAccountFromMnemonic(5))
	// Initialize a genesis file for the first validator
	val0ConfigDir := c.Validators[0].ConfigDir()
	var addrAll []sdk.AccAddress
	for _, val := range c.Validators {
		address, err := val.KeyInfo.GetAddress()
		s.Require().NoError(err)
		addrAll = append(addrAll, address)
	}

	for _, addr := range c.GenesisAccounts {
		acctAddr, err := addr.KeyInfo.GetAddress()
		s.Require().NoError(err)
		addrAll = append(addrAll, acctAddr)
	}

	s.Require().NoError(
		e2einit.ModifyGenesis(val0ConfigDir, "", initBalanceStr, addrAll, initialGlobalFeeAmt+ulunaDenom, ulunaDenom),
	)
	// copy the genesis file to the remaining validators
	for _, val := range c.Validators[1:] {
		_, err := e2eutil.CopyFile(
			filepath.Join(val0ConfigDir, "config", "genesis.json"),
			filepath.Join(val.ConfigDir(), "config", "genesis.json"),
		)
		s.Require().NoError(err)
	}
}

func (s *IntegrationTestSuite) initGenesis(c *e2einit.Chain, vestingMnemonic, jailedValMnemonic string) {
	var (
		serverCtx = server.NewDefaultContext()
		config    = serverCtx.Config
		validator = c.Validators[0]
	)

	config.SetRoot(validator.ConfigDir())
	config.Moniker = validator.Moniker

	genFilePath := config.GenesisFile()
	appGenState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFilePath)
	s.Require().NoError(err)

	appGenState = s.addGenesisVestingAndJailedAccounts(
		c,
		validator.ConfigDir(),
		vestingMnemonic,
		jailedValMnemonic,
		appGenState,
	)

	var evidenceGenState evidencetypes.GenesisState
	s.Require().NoError(e2einit.Cdc.UnmarshalJSON(appGenState[evidencetypes.ModuleName], &evidenceGenState))

	evidenceGenState.Evidence = make([]*codectypes.Any, numberOfEvidences)
	for i := range evidenceGenState.Evidence {
		pk := ed25519.GenPrivKey()
		evidence := &evidencetypes.Equivocation{
			Height:           1,
			Power:            100,
			Time:             time.Now().UTC(),
			ConsensusAddress: sdk.ConsAddress(pk.PubKey().Address().Bytes()).String(),
		}
		evidenceGenState.Evidence[i], err = codectypes.NewAnyWithValue(evidence)
		s.Require().NoError(err)
	}

	appGenState[evidencetypes.ModuleName], err = e2einit.Cdc.MarshalJSON(&evidenceGenState)
	s.Require().NoError(err)

	var genUtilGenState genutiltypes.GenesisState
	s.Require().NoError(e2einit.Cdc.UnmarshalJSON(appGenState[genutiltypes.ModuleName], &genUtilGenState))

	// generate genesis txs
	genTxs := make([]json.RawMessage, len(c.Validators))
	for i, val := range c.Validators {
		createValmsg, err := val.BuildCreateValidatorMsg(stakingAmountCoin)
		s.Require().NoError(err)
		signedTx, err := val.SignMsg(createValmsg)

		s.Require().NoError(err)

		txRaw, err := e2einit.Cdc.MarshalJSON(signedTx)
		s.Require().NoError(err)

		genTxs[i] = txRaw
	}

	genUtilGenState.GenTxs = genTxs

	appGenState[genutiltypes.ModuleName], err = e2einit.Cdc.MarshalJSON(&genUtilGenState)
	s.Require().NoError(err)

	genDoc.AppState, err = json.MarshalIndent(appGenState, "", "  ")
	s.Require().NoError(err)

	bz, err := tmjson.MarshalIndent(genDoc, "", "  ")
	s.Require().NoError(err)

	vestingPeriod, err := generateVestingPeriod()
	s.Require().NoError(err)

	rawTx, _, err := buildRawTx()
	s.Require().NoError(err)

	// write the updated genesis file to each validator.
	for _, val := range c.Validators {
		err = e2eutil.WriteFile(filepath.Join(val.ConfigDir(), "config", "genesis.json"), bz)
		s.Require().NoError(err)

		err = e2eutil.WriteFile(filepath.Join(val.ConfigDir(), vestingPeriodFile), vestingPeriod)
		s.Require().NoError(err)

		err = e2eutil.WriteFile(filepath.Join(val.ConfigDir(), rawTxFile), rawTx)
		s.Require().NoError(err)
	}
}

// TODO find a better way to manipulate accounts to add genesis accounts
func (s *IntegrationTestSuite) addGenesisVestingAndJailedAccounts(
	c *e2einit.Chain,
	valConfigDir,
	vestingMnemonic,
	jailedValMnemonic string,
	appGenState map[string]json.RawMessage,
) map[string]json.RawMessage {
	var (
		authGenState    = authtypes.GetGenesisStateFromAppState(e2einit.Cdc, appGenState)
		bankGenState    = banktypes.GetGenesisStateFromAppState(e2einit.Cdc, appGenState)
		stakingGenState = stakingtypes.GetGenesisStateFromAppState(e2einit.Cdc, appGenState)
	)

	// create genesis vesting accounts keys
	kb, err := keyring.New(e2einit.KeyringAppName, keyring.BackendTest, valConfigDir, nil, e2einit.Cdc)
	s.Require().NoError(err)

	keyringAlgos, _ := kb.SupportedAlgorithms()
	algo, err := keyring.NewSigningAlgoFromString(string(hd.Secp256k1Type), keyringAlgos)
	s.Require().NoError(err)

	// create jailed validator account keys
	jailedValKey, err := kb.NewAccount(JailedValidatorKey, jailedValMnemonic, "", sdk.FullFundraiserPath, algo)
	s.Require().NoError(err)

	// create genesis vesting accounts keys
	c.GenesisVestingAccounts = make(map[string]sdk.AccAddress)
	for i, key := range genesisVestingKeys {
		// Use the first wallet from the same mnemonic by HD path
		acc, err := kb.NewAccount(key, vestingMnemonic, "", e2eutil.HDPath(i), algo)
		s.Require().NoError(err)
		adrr, err := acc.GetAddress()
		s.Require().NoError(err)
		c.GenesisVestingAccounts[key] = adrr
		s.T().Logf("created %s genesis account %s\n", key, c.GenesisVestingAccounts[key].String())
	}
	var (
		continuousVestingAcc = c.GenesisVestingAccounts[continuousVestingKey]
		delayedVestingAcc    = c.GenesisVestingAccounts[delayedVestingKey]
	)

	// add jailed validator to staking store
	pubKey, err := jailedValKey.GetPubKey()
	s.Require().NoError(err)
	jailedValAcc, err := jailedValKey.GetAddress()
	s.Require().NoError(err)
	jailedValAddr := sdk.ValAddress(jailedValAcc)
	val, err := stakingtypes.NewValidator(
		jailedValAddr,
		pubKey,
		stakingtypes.NewDescription("jailed", "", "", "", ""),
	)
	s.Require().NoError(err)
	val.Jailed = true
	val.Tokens = sdk.NewInt(slashingShares)
	val.DelegatorShares = sdk.NewDec(slashingShares)
	stakingGenState.Validators = append(stakingGenState.Validators, val)

	// add jailed validator delegations
	stakingGenState.Delegations = append(stakingGenState.Delegations, stakingtypes.Delegation{
		DelegatorAddress: jailedValAcc.String(),
		ValidatorAddress: jailedValAddr.String(),
		Shares:           sdk.NewDec(slashingShares),
	})

	appGenState[stakingtypes.ModuleName], err = e2einit.Cdc.MarshalJSON(stakingGenState)
	s.Require().NoError(err)

	// add jailed account to the genesis
	baseJailedAccount := authtypes.NewBaseAccount(jailedValAcc, pubKey, 0, 0)
	s.Require().NoError(baseJailedAccount.Validate())

	// add continuous vesting account to the genesis
	baseVestingContinuousAccount := authtypes.NewBaseAccount(
		continuousVestingAcc, nil, 0, 0)
	vestingContinuousGenAccount := authvesting.NewContinuousVestingAccountRaw(
		authvesting.NewBaseVestingAccount(
			baseVestingContinuousAccount,
			sdk.NewCoins(vestingAmountVested),
			time.Now().Add(time.Duration(rand.Intn(80)+150)*time.Second).Unix(),
		),
		time.Now().Add(time.Duration(rand.Intn(40)+90)*time.Second).Unix(),
	)
	s.Require().NoError(vestingContinuousGenAccount.Validate())

	// add delayed vesting account to the genesis
	baseVestingDelayedAccount := authtypes.NewBaseAccount(
		delayedVestingAcc, nil, 0, 0)
	vestingDelayedGenAccount := authvesting.NewDelayedVestingAccountRaw(
		authvesting.NewBaseVestingAccount(
			baseVestingDelayedAccount,
			sdk.NewCoins(vestingAmountVested),
			time.Now().Add(time.Duration(rand.Intn(40)+90)*time.Second).Unix(),
		),
	)
	s.Require().NoError(vestingDelayedGenAccount.Validate())

	// unpack and append accounts
	accs, err := authtypes.UnpackAccounts(authGenState.Accounts)
	s.Require().NoError(err)
	accs = append(accs, vestingContinuousGenAccount, vestingDelayedGenAccount, baseJailedAccount)
	accs = authtypes.SanitizeGenesisAccounts(accs)
	genAccs, err := authtypes.PackAccounts(accs)
	s.Require().NoError(err)
	authGenState.Accounts = genAccs

	// update auth module state
	appGenState[authtypes.ModuleName], err = e2einit.Cdc.MarshalJSON(&authGenState)
	s.Require().NoError(err)

	// update balances
	vestingContinuousBalances := banktypes.Balance{
		Address: continuousVestingAcc.String(),
		Coins:   vestingBalance,
	}
	vestingDelayedBalances := banktypes.Balance{
		Address: delayedVestingAcc.String(),
		Coins:   vestingBalance,
	}
	jailedValidatorBalances := banktypes.Balance{
		Address: jailedValAcc.String(),
		Coins:   sdk.NewCoins(tokenAmount),
	}
	stakingModuleBalances := banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName).String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(ulunaDenom, sdk.NewInt(slashingShares))),
	}
	bankGenState.Balances = append(
		bankGenState.Balances,
		vestingContinuousBalances,
		vestingDelayedBalances,
		jailedValidatorBalances,
		stakingModuleBalances,
	)
	bankGenState.Balances = banktypes.SanitizeGenesisBalances(bankGenState.Balances)

	// update the denom metadata for the bank module
	bankGenState.DenomMetadata = append(bankGenState.DenomMetadata, banktypes.Metadata{
		Description: "An example stable token",
		Display:     ulunaDenom,
		Base:        ulunaDenom,
		Symbol:      ulunaDenom,
		Name:        ulunaDenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    ulunaDenom,
				Exponent: 0,
			},
		},
	})

	// update bank module state
	appGenState[banktypes.ModuleName], err = e2einit.Cdc.MarshalJSON(bankGenState)
	s.Require().NoError(err)

	return appGenState
}

// initValidatorConfigs initializes the validator configs for the given chain.
func (s *IntegrationTestSuite) initValidatorConfigs(c *e2einit.Chain) {
	for i, val := range c.Validators {
		tmCfgPath := filepath.Join(val.ConfigDir(), "config", "config.toml")

		vpr := viper.New()
		vpr.SetConfigFile(tmCfgPath)
		s.Require().NoError(vpr.ReadInConfig())

		valConfig := tmconfig.DefaultConfig()

		s.Require().NoError(vpr.Unmarshal(valConfig))

		valConfig.P2P.ListenAddress = "tcp://0.0.0.0:26656"
		valConfig.P2P.AddrBookStrict = false
		valConfig.P2P.ExternalAddress = fmt.Sprintf("%s:%d", val.InstanceName(), 26656)
		valConfig.RPC.ListenAddress = "tcp://0.0.0.0:26657"
		valConfig.StateSync.Enable = false
		valConfig.LogLevel = "info"

		var peers []string

		for j := 0; j < len(c.Validators); j++ {
			if i == j {
				continue
			}

			peer := c.Validators[j]
			peerID := fmt.Sprintf("%s@%s%d:26656", peer.NodeKey.ID(), peer.Moniker, j)
			peers = append(peers, peerID)
		}

		valConfig.P2P.PersistentPeers = strings.Join(peers, ",")

		tmconfig.WriteConfigFile(tmCfgPath, valConfig)

		// set application configuration
		appCfgPath := filepath.Join(val.ConfigDir(), "config", "app.toml")

		appConfig := srvconfig.DefaultConfig()
		appConfig.API.Enable = true
		appConfig.MinGasPrices = fmt.Sprintf("%s%s", minGasPrice, ulunaDenom)

		srvconfig.SetConfigTemplate(srvconfig.DefaultConfigTemplate)
		srvconfig.WriteConfigFile(appCfgPath, appConfig)
	}
}

// runValidators runs the validators in the chain
func (s *IntegrationTestSuite) runValidators(c *e2einit.Chain, portOffset int) {
	s.T().Logf("starting Terra %s validator containers...", c.ChainMeta.Id)

	s.valResources[c.ChainMeta.Id] = make([]*dockertest.Resource, len(c.Validators))
	for i, val := range c.Validators {
		runOpts := &dockertest.RunOptions{
			Name:      val.InstanceName(),
			NetworkID: s.dkrNet.Network.ID,
			Mounts: []string{
				fmt.Sprintf("%s/:%s", val.ConfigDir(), terraHomePath),
			},
			Repository: "cosmos/terrad-e2e",
		}

		s.Require().NoError(exec.Command("chmod", "-R", "0777", val.ConfigDir()).Run()) //nolint:gosec // this is a test

		// expose the first validator for debugging and communication
		if val.Index == 0 {
			runOpts.PortBindings = map[docker.Port][]docker.PortBinding{
				"1317/tcp":  {{HostIP: "", HostPort: fmt.Sprintf("%d", 1317+portOffset)}},
				"6060/tcp":  {{HostIP: "", HostPort: fmt.Sprintf("%d", 6060+portOffset)}},
				"6061/tcp":  {{HostIP: "", HostPort: fmt.Sprintf("%d", 6061+portOffset)}},
				"6062/tcp":  {{HostIP: "", HostPort: fmt.Sprintf("%d", 6062+portOffset)}},
				"6063/tcp":  {{HostIP: "", HostPort: fmt.Sprintf("%d", 6063+portOffset)}},
				"6064/tcp":  {{HostIP: "", HostPort: fmt.Sprintf("%d", 6064+portOffset)}},
				"6065/tcp":  {{HostIP: "", HostPort: fmt.Sprintf("%d", 6065+portOffset)}},
				"9090/tcp":  {{HostIP: "", HostPort: fmt.Sprintf("%d", 9090+portOffset)}},
				"26656/tcp": {{HostIP: "", HostPort: fmt.Sprintf("%d", 26656+portOffset)}},
				"26657/tcp": {{HostIP: "", HostPort: fmt.Sprintf("%d", 26657+portOffset)}},
			}
		}

		resource, err := s.dkrPool.RunWithOptions(runOpts, noRestart)
		s.Require().NoError(err)

		s.valResources[c.ChainMeta.Id][i] = resource
		s.T().Logf("started Terra %s validator container: %s", c.ChainMeta.Id, resource.Container.ID)
	}

	rpcClient, err := rpchttp.New("tcp://localhost:26657", "/websocket")
	s.Require().NoError(err)

	s.Require().Eventually(
		func() bool {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			status, err := rpcClient.Status(ctx)
			if err != nil {
				return false
			}

			// let the node produce a few blocks
			if status.SyncInfo.CatchingUp || status.SyncInfo.LatestBlockHeight < 3 {
				return false
			}

			return true
		},
		5*time.Minute,
		time.Second,
		"terra node failed to produce blocks",
	)
}
func noRestart(config *docker.HostConfig) {
	// in this case we don't want the nodes to restart on failure
	config.RestartPolicy = docker.RestartPolicy{
		Name: "no",
	}
}

// hermes0 is for ibc and packet-forward-middleware(PFM) test, hermes0 is keep running during the ibc and PFM test.
func (s *IntegrationTestSuite) runIBCRelayer0() {
	s.T().Log("starting Hermes relayer container 0...")

	tmpDir, err := os.MkdirTemp("", "terra-e2e-testnet-hermes-")
	s.Require().NoError(err)
	s.tmpDirs = append(s.tmpDirs, tmpDir)

	terraAVal := s.chainA.Validators[0]
	terraBVal := s.chainB.Validators[0]

	terraARly := s.chainA.GenesisAccounts[relayerAccountIndexHermes0]
	terraBRly := s.chainB.GenesisAccounts[relayerAccountIndexHermes0]

	hermesCfgPath := path.Join(tmpDir, "hermes")

	s.Require().NoError(os.MkdirAll(hermesCfgPath, 0o755))
	_, err = e2eutil.CopyFile(
		filepath.Join("./scripts/", "hermes_bootstrap.sh"),
		filepath.Join(hermesCfgPath, "hermes_bootstrap.sh"),
	)
	s.Require().NoError(err)

	s.hermesResource0, err = s.dkrPool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       fmt.Sprintf("%s-%s-relayer-0", s.chainA.ChainMeta.Id, s.chainB.ChainMeta.Id),
			Repository: "ghcr.io/cosmos/hermes-e2e",
			Tag:        "1.0.0",
			NetworkID:  s.dkrNet.Network.ID,
			Mounts: []string{
				fmt.Sprintf("%s/:/root/hermes", hermesCfgPath),
			},
			PortBindings: map[docker.Port][]docker.PortBinding{
				"3031/tcp": {{HostIP: "", HostPort: "3031"}},
			},
			Env: []string{
				fmt.Sprintf("terra_A_E2E_CHAIN_ID=%s", s.chainA.ChainMeta.Id),
				fmt.Sprintf("terra_B_E2E_CHAIN_ID=%s", s.chainB.ChainMeta.Id),
				fmt.Sprintf("terra_A_E2E_VAL_MNEMONIC=%s", terraAVal.Mnemonic),
				fmt.Sprintf("terra_B_E2E_VAL_MNEMONIC=%s", terraBVal.Mnemonic),
				fmt.Sprintf("terra_A_E2E_RLY_MNEMONIC=%s", terraARly.Mnemonic),
				fmt.Sprintf("terra_B_E2E_RLY_MNEMONIC=%s", terraBRly.Mnemonic),
				fmt.Sprintf("terra_A_E2E_VAL_HOST=%s", s.valResources[s.chainA.ChainMeta.Id][0].Container.Name[1:]),
				fmt.Sprintf("terra_B_E2E_VAL_HOST=%s", s.valResources[s.chainB.ChainMeta.Id][0].Container.Name[1:]),
			},
			Entrypoint: []string{
				"sh",
				"-c",
				"chmod +x /root/hermes/hermes_bootstrap.sh && /root/hermes/hermes_bootstrap.sh",
			},
		},
		noRestart,
	)
	s.Require().NoError(err)

	endpoint := fmt.Sprintf("http://%s/state", s.hermesResource0.GetHostPort("3031/tcp"))
	s.Require().Eventually(
		func() bool {
			resp, err := http.Get(endpoint) //nolint:gosec // this is a test
			if err != nil {
				return false
			}

			defer resp.Body.Close()

			bz, err := io.ReadAll(resp.Body)
			if err != nil {
				return false
			}

			var respBody map[string]interface{}
			if err := json.Unmarshal(bz, &respBody); err != nil {
				return false
			}

			status := respBody["status"].(string)
			result := respBody["result"].(map[string]interface{})

			return status == "success" && len(result["chains"].([]interface{})) == 2
		},
		5*time.Minute,
		time.Second,
		"hermes relayer not healthy",
	)

	s.T().Logf("started Hermes relayer 0 container: %s", s.hermesResource0.Container.ID)

	// XXX: Give time to both networks to start, otherwise we might see gRPC
	// transport errors.
	time.Sleep(10 * time.Second)

	// create the client, connection and channel between the two terra chains
	s.createConnection()
	time.Sleep(10 * time.Second)
	s.createChannel()
}

// hermes1 is for bypass-msg test. Hermes1 is to process asynchronous transactions,
// Hermes1 has access to two Hermes configurations: one configuration allows paying fees, while the other does not.
// With Hermes1, better control can be achieved regarding whether fees are paid when clearing transactions.
func (s *IntegrationTestSuite) runIBCRelayer1() {
	s.T().Log("starting Hermes relayer container 1...")

	tmpDir, err := os.MkdirTemp("", "terra-e2e-testnet-hermes-")
	s.Require().NoError(err)
	s.tmpDirs = append(s.tmpDirs, tmpDir)

	terraAVal := s.chainA.Validators[0]
	terraBVal := s.chainB.Validators[0]

	terraARly := s.chainA.GenesisAccounts[relayerAccountIndexHermes1]
	terraBRly := s.chainB.GenesisAccounts[relayerAccountIndexHermes1]

	hermesCfgPath := path.Join(tmpDir, "hermes")

	s.Require().NoError(os.MkdirAll(hermesCfgPath, 0o755))
	_, err = e2eutil.CopyFile(
		filepath.Join("./scripts/", "hermes1_bootstrap.sh"),
		filepath.Join(hermesCfgPath, "hermes1_bootstrap.sh"),
	)
	s.Require().NoError(err)

	s.hermesResource1, err = s.dkrPool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       fmt.Sprintf("%s-%s-relayer-1", s.chainA.ChainMeta.Id, s.chainB.ChainMeta.Id),
			Repository: "ghcr.io/cosmos/hermes-e2e",
			Tag:        "1.0.0",
			NetworkID:  s.dkrNet.Network.ID,
			Mounts: []string{
				fmt.Sprintf("%s/:/root/hermes", hermesCfgPath),
			},
			PortBindings: map[docker.Port][]docker.PortBinding{
				"3032/tcp": {{HostIP: "", HostPort: "3032"}},
			},
			Env: []string{
				fmt.Sprintf("terra_A_E2E_CHAIN_ID=%s", s.chainA.ChainMeta.Id),
				fmt.Sprintf("terra_B_E2E_CHAIN_ID=%s", s.chainB.ChainMeta.Id),
				fmt.Sprintf("terra_A_E2E_VAL_MNEMONIC=%s", terraAVal.Mnemonic),
				fmt.Sprintf("terra_B_E2E_VAL_MNEMONIC=%s", terraBVal.Mnemonic),
				fmt.Sprintf("terra_A_E2E_RLY_MNEMONIC=%s", terraARly.Mnemonic),
				fmt.Sprintf("terra_B_E2E_RLY_MNEMONIC=%s", terraBRly.Mnemonic),
				fmt.Sprintf("terra_A_E2E_VAL_HOST=%s", s.valResources[s.chainA.ChainMeta.Id][0].Container.Name[1:]),
				fmt.Sprintf("terra_B_E2E_VAL_HOST=%s", s.valResources[s.chainB.ChainMeta.Id][0].Container.Name[1:]),
			},
			Entrypoint: []string{
				"sh",
				"-c",
				"chmod +x /root/hermes/hermes1_bootstrap.sh && /root/hermes/hermes1_bootstrap.sh && tail -f /dev/null",
			},
		},
		noRestart,
	)
	s.Require().NoError(err)

	s.T().Logf("started Hermes relayer 1 container: %s", s.hermesResource1.Container.ID)

	// XXX: Give time to both networks to start, otherwise we might see gRPC
	// transport errors.
	time.Sleep(10 * time.Second)
}
