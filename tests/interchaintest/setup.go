package interchaintest

import (
	"encoding/json"
	"fmt"

	"github.com/icza/dyno"

	"cosmossdk.io/math"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"
)

var (
	TerraClassicE2ERepo  = "terra-classic/core-e2e"
	TerraClassicMainRepo = "terra-classic/core"

	repo, version = GetDockerImageInfo()

	TerraClassicImage = ibc.DockerImage{
		Repository: repo,
		Version:    version,
		UidGid:     "1025:1025",
	}

	pathTerraGaia       = "terra-gaia"
	genesisWalletAmount = math.NewInt(10_000_000_000)
	votingPeriod        = "30s"
	maxDepositPeriod    = "10s"
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
			GasPrices:           "0.0uluna",
			GasAdjustment:       1.1,
			TrustingPeriod:      "112h",
			NoHostMount:         false,
			ModifyGenesis:       nil,
			ConfigFileOverrides: nil,
			EncodingConfig:      coreEncoding(),
		},
		nil
}

// coreEncoding registers the Terra Classic specific module codecs so that the associated types and msgs
// will be supported when writing to the blocksdb sqlite database.
func coreEncoding() *simappparams.EncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// register custom types
	govv1.RegisterInterfaces(cfg.InterfaceRegistry)
	return &cfg
}

func ModifyGenesisShortProposals(votingPeriod, maxDepositPeriod string) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}
		if err := dyno.Set(g, votingPeriod, "app_state", "gov", "params", "voting_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", "params", "max_deposit_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "params", "min_deposit", 0, "denom"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}
