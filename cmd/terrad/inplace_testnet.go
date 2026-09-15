package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cobra"
)

const flagSkipConfirmation = "skip-confirmation"

// genesisDocKey is the key under which CometBFT stores the genesis doc in state.db.
var genesisDocKey = []byte("genesisDoc")

// inPlaceTestnetCmd wraps the SDK in-place-testnet command. The SDK rewrites
// the chain ID in genesis.json but stores the genesis doc it loaded from
// state.db, so CometBFT would keep advertising the old network in its node
// info. The wrapper updates the stored genesis doc first.
func inPlaceTestnetCmd(appCreator servertypes.AppCreator, addFlags func(*cobra.Command)) *cobra.Command {
	cmd := server.InPlaceTestnetCreator(appCreator)
	addFlags(cmd)

	sdkRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
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

		serverCtx := server.GetServerContextFromCmd(cmd)
		if err := setStoredGenesisChainID(serverCtx.Config, args[0]); err != nil {
			return fmt.Errorf("update stored genesis doc: %w", err)
		}

		return sdkRunE(cmd, args)
	}

	return cmd
}

func setStoredGenesisChainID(config *cmtcfg.Config, chainID string) error {
	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: config})
	if err != nil {
		return err
	}
	defer stateDB.Close()

	bz, err := stateDB.Get(genesisDocKey)
	if err != nil {
		return err
	}
	if len(bz) == 0 {
		// CometBFT loads genesis.json, which the SDK already rewrites.
		return nil
	}

	var genDoc cmttypes.GenesisDoc
	if err := cmtjson.Unmarshal(bz, &genDoc); err != nil {
		return err
	}
	if genDoc.ChainID == chainID {
		return nil
	}
	genDoc.ChainID = chainID

	bz, err = cmtjson.Marshal(&genDoc)
	if err != nil {
		return err
	}
	return stateDB.SetSync(genesisDocKey, bz)
}
