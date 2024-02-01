package initialization

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	staketypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/gogo/protobuf/proto"
	tmjson "github.com/tendermint/tendermint/libs/json"

	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/classic-terra/core/v2/tests/e2e/util"
	treasurytypes "github.com/classic-terra/core/v2/x/treasury/types"
)

// NodeConfig is a confiuration for the node supplied from the test runner
// to initialization scripts. It should be backwards compatible with earlier
// versions. If this struct is updated, the change must be backported to earlier
// branches that might be used for upgrade testing.
type NodeConfig struct {
	Name               string // name of the config that will also be assigned to Docke container.
	Pruning            string // default, nothing, everything, or custom
	PruningKeepRecent  string // keep all of the last N states (only used with custom pruning)
	PruningInterval    string // delete old states from every Nth block (only used with custom pruning)
	SnapshotInterval   uint64 // statesync snapshot every Nth block (0 to disable)
	SnapshotKeepRecent uint32 // number of recent snapshots to keep and serve (0 to keep all)
	IsValidator        bool   // flag indicating whether a node should be a validator
}

const (
	// common
	TerraDenom          = "uluna"
	AtomDenom           = "uatom"
	TerraIBCDenom       = "ibc/4627AD2524E3E0523047E35BB76CC90E37D9D57ACF14F0FCBCEB2480705F3CB8"
	MinGasPrice         = "0.000"
	IbcSendAmount       = 3300000000
	ValidatorWalletName = "val"
	// chainA
	ChainAID      = "terra-test-a"
	TerraBalanceA = 20000000000000
	StakeBalanceA = 110000000000
	StakeAmountA  = 100000000000
	// chainB
	ChainBID          = "terra-test-b"
	TerraBalanceB     = 500000000000
	StakeBalanceB     = 440000000000
	StakeAmountB      = 400000000000
	GenesisFeeBalance = 100000000000
	WalletFeeBalance  = 100000000
	// chainC
	ChainCID      = "terra-test-c"
	TerraBalanceC = 500000000000
	StakeBalanceC = 440000000000
	StakeAmountC  = 400000000000
)

var (
	StakeAmountIntA  = sdk.NewInt(StakeAmountA)
	StakeAmountCoinA = sdk.NewCoin(TerraDenom, StakeAmountIntA)
	StakeAmountIntB  = sdk.NewInt(StakeAmountB)
	StakeAmountCoinB = sdk.NewCoin(TerraDenom, StakeAmountIntB)

	InitBalanceStrA = fmt.Sprintf("%d%s", TerraBalanceA, TerraDenom)
	InitBalanceStrB = fmt.Sprintf("%d%s", TerraBalanceB, TerraDenom)
	InitBalanceStrC = fmt.Sprintf("%d%s", TerraBalanceC, TerraDenom)
	LunaToken       = sdk.NewInt64Coin(TerraDenom, IbcSendAmount) // 3,300luna
	tenTerra        = sdk.Coins{sdk.NewInt64Coin(TerraDenom, 10_000_000)}

	oneMin = time.Minute //nolint
)

func addAccount(path, moniker, amountStr string, accAddr sdk.AccAddress, forkHeight int) error {
	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config

	config.SetRoot(path)
	config.Moniker = moniker

	coins, err := sdk.ParseCoinsNormalized(amountStr)
	if err != nil {
		return fmt.Errorf("failed to parse coins: %w", err)
	}
	coins = coins.Sort()

	balances := banktypes.Balance{Address: accAddr.String(), Coins: coins.Sort()}
	genAccount := authtypes.NewBaseAccount(accAddr, nil, 0, 0)
	// TODO: Make the SDK make it far cleaner to add an account to GenesisState
	genFile := config.GenesisFile()
	appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
	if err != nil {
		return fmt.Errorf("failed to unmarshal genesis state: %w", err)
	}

	genDoc.InitialHeight = int64(forkHeight)

	authGenState := authtypes.GetGenesisStateFromAppState(util.Cdc, appState)

	accs, err := authtypes.UnpackAccounts(authGenState.Accounts)
	if err != nil {
		return fmt.Errorf("failed to get accounts from any: %w", err)
	}

	if accs.Contains(accAddr) {
		return fmt.Errorf("failed to add account to genesis state; account already exists: %s", accAddr)
	}

	// Add the new account to the set of genesis accounts and sanitize the
	// accounts afterwards.
	accs = append(accs, genAccount)
	accs = authtypes.SanitizeGenesisAccounts(accs)

	genAccs, err := authtypes.PackAccounts(accs)
	if err != nil {
		return fmt.Errorf("failed to convert accounts into any's: %w", err)
	}

	authGenState.Accounts = genAccs

	authGenStateBz, err := util.Cdc.MarshalJSON(&authGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal auth genesis state: %w", err)
	}

	appState[authtypes.ModuleName] = authGenStateBz

	bankGenState := banktypes.GetGenesisStateFromAppState(util.Cdc, appState)
	bankGenState.Balances = append(bankGenState.Balances, balances)
	bankGenState.Balances = banktypes.SanitizeGenesisBalances(bankGenState.Balances)

	bankGenStateBz, err := util.Cdc.MarshalJSON(bankGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal bank genesis state: %w", err)
	}

	appState[banktypes.ModuleName] = bankGenStateBz

	appStateJSON, err := json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("failed to marshal application genesis state: %w", err)
	}

	genDoc.AppState = appStateJSON
	return genutil.ExportGenesisFile(genDoc, genFile)
}

func updateModuleGenesis[V proto.Message](appGenState map[string]json.RawMessage, moduleName string, protoVal V, updateGenesis func(V)) error {
	if err := util.Cdc.UnmarshalJSON(appGenState[moduleName], protoVal); err != nil {
		return err
	}
	updateGenesis(protoVal)
	newGenState := protoVal

	bz, err := util.Cdc.MarshalJSON(newGenState)
	if err != nil {
		return err
	}
	appGenState[moduleName] = bz
	return nil
}

func initGenesis(chain *internalChain, forkHeight int) error {
	// initialize a genesis file
	configDir := chain.nodes[0].configDir()
	for _, val := range chain.nodes {
		accAdd, err := val.keyInfo.GetAddress()
		if err != nil {
			return err
		}

		switch chain.chainMeta.ID {
		case ChainAID:
			if err := addAccount(configDir, "", InitBalanceStrA, accAdd, forkHeight); err != nil {
				return err
			}
		case ChainBID:
			if err := addAccount(configDir, "", InitBalanceStrB, accAdd, forkHeight); err != nil {
				return err
			}
		case ChainCID:
			if err := addAccount(configDir, "", InitBalanceStrC, accAdd, forkHeight); err != nil {
				return err
			}
		}
	}

	// copy the genesis file to the remaining validators
	for _, val := range chain.nodes[1:] {
		_, err := util.CopyFile(
			filepath.Join(configDir, "config", "genesis.json"),
			filepath.Join(val.configDir(), "config", "genesis.json"),
		)
		if err != nil {
			return err
		}
	}

	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config

	config.SetRoot(chain.nodes[0].configDir())
	config.Moniker = chain.nodes[0].moniker

	genFilePath := config.GenesisFile()
	appGenState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFilePath)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, staketypes.ModuleName, &staketypes.GenesisState{}, updateStakeGenesis)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, minttypes.ModuleName, &minttypes.GenesisState{}, updateMintGenesis)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, banktypes.ModuleName, &banktypes.GenesisState{}, updateBankGenesis)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, crisistypes.ModuleName, &crisistypes.GenesisState{}, updateCrisisGenesis)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, treasurytypes.ModuleName, &treasurytypes.GenesisState{}, updateTreasuryGenesis)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, govtypes.ModuleName, &govv1.GenesisState{}, updateGovGenesis)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, genutiltypes.ModuleName, &genutiltypes.GenesisState{}, updateGenUtilGenesis(chain))
	if err != nil {
		return err
	}

	bz, err := json.MarshalIndent(appGenState, "", "  ")
	if err != nil {
		return err
	}

	genDoc.AppState = bz

	genesisJSON, err := tmjson.MarshalIndent(genDoc, "", "  ")
	if err != nil {
		return err
	}

	// write the updated genesis file to each validator
	for _, val := range chain.nodes {
		if err := util.WritePublicFile(filepath.Join(val.configDir(), "config", "genesis.json"), genesisJSON); err != nil {
			return err
		}
	}
	return nil
}

func updateMintGenesis(mintGenState *minttypes.GenesisState) {
	mintGenState.Params.MintDenom = TerraDenom
}

func updateBankGenesis(bankGenState *banktypes.GenesisState) {
	denomsToRegister := []string{TerraDenom, AtomDenom}
	for _, denom := range denomsToRegister {
		setDenomMetadata(bankGenState, denom)
	}
}

func updateStakeGenesis(stakeGenState *staketypes.GenesisState) {
	stakeGenState.Params = staketypes.Params{
		BondDenom:         TerraDenom,
		MaxValidators:     100,
		MaxEntries:        7,
		HistoricalEntries: 10000,
		UnbondingTime:     240000000000,
		MinCommissionRate: sdk.ZeroDec(),
	}
}

func updateCrisisGenesis(crisisGenState *crisistypes.GenesisState) {
	crisisGenState.ConstantFee.Denom = TerraDenom
}

func updateTreasuryGenesis(treasuryGenState *treasurytypes.GenesisState) {
	treasuryGenState.TaxRate = sdk.NewDecWithPrec(2, 2) // 0.02
	treasuryGenState.Params.TaxPolicy = treasurytypes.PolicyConstraints{
		RateMin: sdk.NewDecWithPrec(2, 2), // 0.02
		RateMax: sdk.NewDecWithPrec(1, 1), // 0.1
		Cap:     sdk.NewCoin(TerraDenom, sdk.NewInt(100000000000000)),
	}
}

func updateGovGenesis(govGenState *govv1.GenesisState) {
	govGenState.VotingParams.VotingPeriod = &oneMin
	govGenState.TallyParams.Quorum = sdk.NewDecWithPrec(2, 1).String()
	govGenState.DepositParams.MinDeposit = tenTerra
}

func updateGenUtilGenesis(c *internalChain) func(*genutiltypes.GenesisState) {
	return func(genUtilGenState *genutiltypes.GenesisState) {
		// generate genesis txs
		genTxs := make([]json.RawMessage, 0, len(c.nodes))
		for _, node := range c.nodes {
			if !node.isValidator {
				continue
			}

			stakeAmountCoin := StakeAmountCoinA
			if c.chainMeta.ID != ChainAID {
				stakeAmountCoin = StakeAmountCoinB
			}
			createValmsg, err := node.buildCreateValidatorMsg(stakeAmountCoin)
			if err != nil {
				panic("genutil genesis setup failed: " + err.Error())
			}

			signedTx, err := node.signMsg(createValmsg)
			if err != nil {
				panic("genutil genesis setup failed: " + err.Error())
			}

			txRaw, err := util.Cdc.MarshalJSON(signedTx)
			if err != nil {
				panic("genutil genesis setup failed: " + err.Error())
			}
			genTxs = append(genTxs, txRaw)
		}
		genUtilGenState.GenTxs = genTxs
	}
}

func setDenomMetadata(genState *banktypes.GenesisState, denom string) {
	genState.DenomMetadata = append(genState.DenomMetadata, banktypes.Metadata{
		Description: fmt.Sprintf("Registered denom %s for e2e testing", denom),
		Display:     denom,
		Base:        denom,
		Symbol:      denom,
		Name:        denom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom,
				Exponent: 0,
			},
		},
	})
}
