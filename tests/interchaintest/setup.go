package interchaintest

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	"github.com/icza/dyno"

	oracle "github.com/classic-terra/core/v3/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
)

var (
	TerraClassicE2ERepo  = "terra-classic/core-e2e"
	TerraClassicMainRepo = "terra-classic/core"

	repo, version = GetDockerImageInfo()

	TerraClassicImage = ibc.DockerImage{
		Repository: repo,
		Version:    version,
		UIDGID:     "1025:1025",
	}

	pathTerraGaia        = "terra-gaia"
	pathTerraOsmo        = "terra-osmo"
	pathGaiaOsmo         = "gaia-osmo"
	genesisWalletAmount  = int64(10000000000)
	genesisWalletBalance = math.NewInt(genesisWalletAmount)
	votingPeriod         = "30s"
	maxDepositPeriod     = "10s"
	signedBlocksWindow   = int64(20)
)

func createConfig() (ibc.ChainConfig, error) {
	return ibc.ChainConfig{
			Type:                "cosmos",
			Name:                "core",
			ChainID:             "core-1",
			Images:              []ibc.DockerImage{TerraClassicImage},
			Bin:                 "terrad",
			Bech32Prefix:        "terra",
			Denom:               "uluna",
			GasPrices:           "28.325uluna",
			GasAdjustment:       2.5,
			TrustingPeriod:      "112h",
			NoHostMount:         false,
			ModifyGenesis:       ModifyGenesis(),
			ConfigFileOverrides: nil,
			EncodingConfig:      coreEncoding(),
		},
		nil
}

// coreEncoding registers the Terra Classic specific module codecs so that the associated types and msgs
// will be supported when writing to the blocksdb sqlite database.
func coreEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// register custom types
	// crypto keys (ed25519/secp256k1) so Any-encoded consensus_pubkey can be unpacked
	cryptocodec.RegisterInterfaces(cfg.InterfaceRegistry)
	govv1.RegisterInterfaces(cfg.InterfaceRegistry)
	oracle.RegisterInterfaces(cfg.InterfaceRegistry)
	return &cfg
}

func ModifyGenesis() func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}
		// Modify short proposal
		if err := dyno.Set(g, votingPeriod, "app_state", "gov", "params", "voting_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", "params", "max_deposit_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "params", "min_deposit", 0, "denom"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		// Modify signed blocks window
		if err := dyno.Set(g, signedBlocksWindow, "app_state", "slashing", "params", "signed_blocks_window"); err != nil {
			return nil, fmt.Errorf("failed to set signed blocks window in genesis json: %w", err)
		}
		// Explicitly set min_signed_per_window to 50% to avoid mass jailing on transient stalls
		if err := dyno.Set(g, "0.500000000000000000", "app_state", "slashing", "params", "min_signed_per_window"); err != nil {
			return nil, fmt.Errorf("failed to set min_signed_per_window in genesis json: %w", err)
		}
		// Shorten downtime jail duration to make the test deterministic in case of brief stalls
		if err := dyno.Set(g, "60s", "app_state", "slashing", "params", "downtime_jail_duration"); err != nil {
			return nil, fmt.Errorf("failed to set downtime_jail_duration in genesis json: %w", err)
		}
		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}
