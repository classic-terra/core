package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
)

const flagSkipConfirmation = "skip-confirmation"

// genesisDocKey is the key under which CometBFT stores the genesis doc in state.db.
var genesisDocKey = []byte("genesisDoc")

// inPlaceTestnetCmd wraps the SDK in-place-testnet command. The SDK rewrites
// the chain ID in genesis.json but stores the genesis doc it loaded from
// state.db, so CometBFT would keep advertising the old network in its node
// info. The wrapper updates the stored genesis doc before handing over to the
// SDK, after validating everything the SDK checks before its own first write,
// so a rejected run leaves the data folder untouched.
func inPlaceTestnetCmd(appCreator servertypes.AppCreator, addFlags func(*cobra.Command)) *cobra.Command {
	cmd := server.InPlaceTestnetCreator(appCreator)
	addFlags(cmd)

	sdkRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		serverCtx := server.GetServerContextFromCmd(cmd)
		if err := validateInPlaceTestnet(cmd, serverCtx, args[0], args[1]); err != nil {
			return err
		}
		genDocBz, err := storedGenesisDocWithChainID(serverCtx.Config, args[0])
		if err != nil {
			return fmt.Errorf("read stored genesis doc: %w", err)
		}

		skip, _ := cmd.Flags().GetBool(flagSkipConfirmation)
		if !skip {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("This operation will modify state in your data folder and cannot be undone. Do you want to continue? (y/n)")
			text, _ := reader.ReadString('\n')
			response := strings.TrimSpace(strings.ToLower(text))
			if response != "y" && response != "yes" {
				fmt.Println("Operation canceled.")
				return nil
			}
			if err := cmd.Flags().Set(flagSkipConfirmation, "true"); err != nil {
				return err
			}
		}

		if genDocBz != nil {
			if err := writeStoredGenesisDoc(serverCtx.Config, genDocBz); err != nil {
				return fmt.Errorf("update stored genesis doc: %w", err)
			}
		}

		return sdkRunE(cmd, args)
	}

	return cmd
}

// validateInPlaceTestnet runs the checks the SDK performs before it first
// writes to the data folder (flags, app config, genesis.json with the new
// chain ID), plus the operator address that the app decodes later.
func validateInPlaceTestnet(cmd *cobra.Command, serverCtx *server.Context, chainID, operatorAddress string) error {
	if _, err := server.GetPruningOptionsFromFlags(serverCtx.Viper); err != nil {
		return err
	}
	if _, err := client.GetClientQueryContext(cmd); err != nil {
		return err
	}

	svrCfg, err := serverconfig.GetConfig(serverCtx.Viper)
	if err != nil {
		return err
	}
	if err := svrCfg.ValidateBasic(); err != nil {
		return err
	}

	appGen, err := genutiltypes.AppGenesisFromFile(serverCtx.Config.GenesisFile())
	if err != nil {
		return err
	}
	appGen.ChainID = chainID
	if err := appGen.ValidateAndComplete(); err != nil {
		return err
	}

	if _, _, err := bech32.DecodeAndConvert(operatorAddress); err != nil {
		return fmt.Errorf("decode operator address: %w", err)
	}
	return nil
}

// storedGenesisDocWithChainID returns the genesis doc stored in state.db
// re-encoded with the new chain ID, or nil if nothing has to be written.
func storedGenesisDocWithChainID(config *cmtcfg.Config, chainID string) ([]byte, error) {
	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: config})
	if err != nil {
		return nil, err
	}
	defer stateDB.Close()

	bz, err := stateDB.Get(genesisDocKey)
	if err != nil {
		return nil, err
	}
	if len(bz) == 0 {
		// CometBFT loads genesis.json, which the SDK already rewrites.
		return nil, nil
	}

	var genDoc cmttypes.GenesisDoc
	if err := cmtjson.Unmarshal(bz, &genDoc); err != nil {
		return nil, err
	}
	if genDoc.ChainID == chainID {
		return nil, nil
	}
	genDoc.ChainID = chainID
	if err := genDoc.ValidateAndComplete(); err != nil {
		return nil, err
	}

	return cmtjson.Marshal(&genDoc)
}

func writeStoredGenesisDoc(config *cmtcfg.Config, bz []byte) error {
	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: config})
	if err != nil {
		return err
	}
	defer stateDB.Close()

	return stateDB.SetSync(genesisDocKey, bz)
}
