package configurer

import (
	"os"
	"testing"

	"github.com/classic-terra/core/v2/tests/e2e/configurer/chain"
	"github.com/classic-terra/core/v2/tests/e2e/containers"
	"github.com/classic-terra/core/v2/tests/e2e/initialization"
)

type CurrentBranchConfigurer struct {
	baseConfigurer
}

var _ Configurer = (*CurrentBranchConfigurer)(nil)

func NewCurrentBranchConfigurer(t *testing.T, chainConfigs []*chain.Config, setupTests setupFn, containerManager *containers.Manager) Configurer {
	return &CurrentBranchConfigurer{
		baseConfigurer: baseConfigurer{
			chainConfigs:     chainConfigs,
			containerManager: containerManager,
			setupTests:       setupTests,
			syncUntilHeight:  defaultSyncUntilHeight,
			t:                t,
		},
	}
}

func (cb *CurrentBranchConfigurer) ConfigureChains() error {
	for _, chainConfig := range cb.chainConfigs {
		if err := cb.ConfigureChain(chainConfig); err != nil {
			return err
		}
	}
	return nil
}

func (cb *CurrentBranchConfigurer) ConfigureChain(chainConfig *chain.Config) error {
	cb.t.Logf("starting e2e infrastructure from current branch for chain-id: %s", chainConfig.ID)
	tmpDir, err := os.MkdirTemp("", "terra-e2e-testnet-")
	if err != nil {
		return err
	}
	cb.t.Logf("temp directory for chain-id %v: %v", chainConfig.ID, tmpDir)

	initializedChain, err := initialization.InitChain(chainConfig.ID, tmpDir, chainConfig.ValidatorInitConfigs, 0)
	if err != nil {
		return err
	}
	cb.initializeChainConfigFromInitChain(initializedChain, chainConfig)
	return nil
}

func (cb *CurrentBranchConfigurer) RunSetup() error {
	return cb.setupTests(cb)
}
