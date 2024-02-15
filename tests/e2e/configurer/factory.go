package configurer

import (
	"testing"

	"github.com/classic-terra/core/v2/tests/e2e/configurer/chain"
	"github.com/classic-terra/core/v2/tests/e2e/containers"
	"github.com/classic-terra/core/v2/tests/e2e/initialization"
)

type Configurer interface {
	ConfigureChains() error

	ClearResources() error

	GetChainConfig(chainIndex int) *chain.Config

	RunSetup() error

	RunValidators() error

	RunIBC() error
}

var (
	// each started validator containers corresponds to one of
	// the configurations below.
	validatorConfigsChainA = []*initialization.NodeConfig{
		{
			// this is a node that is used to state-sync from so its snapshot-interval
			// is frequent.
			Name:               "prune-default-snapshot-state-sync-from",
			Pruning:            "default",
			PruningKeepRecent:  "0",
			PruningInterval:    "0",
			SnapshotInterval:   25,
			SnapshotKeepRecent: 10,
			IsValidator:        true,
		},
		{
			Name:               "prune-nothing-snapshot",
			Pruning:            "nothing",
			PruningKeepRecent:  "0",
			PruningInterval:    "0",
			SnapshotInterval:   1500,
			SnapshotKeepRecent: 2,
			IsValidator:        true,
		},
		{
			Name:               "prune-custom-10000-13-snapshot",
			Pruning:            "custom",
			PruningKeepRecent:  "10000",
			PruningInterval:    "13",
			SnapshotInterval:   1500,
			SnapshotKeepRecent: 2,
			IsValidator:        true,
		},
		{
			Name:               "prune-everything-no-snapshot",
			Pruning:            "everything",
			PruningKeepRecent:  "0",
			PruningInterval:    "0",
			SnapshotInterval:   0,
			SnapshotKeepRecent: 0,
			IsValidator:        true,
		},
	}
	validatorConfigsChainB = []*initialization.NodeConfig{
		{
			Name:               "prune-default-snapshot",
			Pruning:            "default",
			PruningKeepRecent:  "0",
			PruningInterval:    "0",
			SnapshotInterval:   1500,
			SnapshotKeepRecent: 2,
			IsValidator:        true,
		},
		{
			Name:               "prune-nothing-snapshot",
			Pruning:            "nothing",
			PruningKeepRecent:  "0",
			PruningInterval:    "0",
			SnapshotInterval:   1500,
			SnapshotKeepRecent: 2,
			IsValidator:        true,
		},
		{
			Name:               "prune-custom-snapshot",
			Pruning:            "custom",
			PruningKeepRecent:  "10000",
			PruningInterval:    "13",
			SnapshotInterval:   1500,
			SnapshotKeepRecent: 2,
			IsValidator:        true,
		},
	}
	validatorConfigsChainC = []*initialization.NodeConfig{
		{
			Name:               "prune-default-snapshot",
			Pruning:            "default",
			PruningKeepRecent:  "0",
			PruningInterval:    "0",
			SnapshotInterval:   1500,
			SnapshotKeepRecent: 2,
			IsValidator:        true,
		},
		{
			Name:               "prune-nothing-snapshot",
			Pruning:            "nothing",
			PruningKeepRecent:  "0",
			PruningInterval:    "0",
			SnapshotInterval:   1500,
			SnapshotKeepRecent: 2,
			IsValidator:        true,
		},
		{
			Name:               "prune-custom-snapshot",
			Pruning:            "custom",
			PruningKeepRecent:  "10000",
			PruningInterval:    "13",
			SnapshotInterval:   1500,
			SnapshotKeepRecent: 2,
			IsValidator:        true,
		},
	}
)

// New returns a new Configurer depending on the values of its parameters.
// - If only isIBCEnabled, we want to have 2 chains initialized at the current
// Git branch version of Terra  codebase.
func New(t *testing.T, isIBCEnabled, isDebugLogEnabled bool) (Configurer, error) {
	containerManager, err := containers.NewManager(isDebugLogEnabled)
	if err != nil {
		return nil, err
	}
	if isIBCEnabled {
		// configure two chains from current Git branch
		return NewCurrentBranchConfigurer(t,
			[]*chain.Config{
				chain.New(t, containerManager, initialization.ChainAID, validatorConfigsChainA),
				chain.New(t, containerManager, initialization.ChainBID, validatorConfigsChainB),
				chain.New(t, containerManager, initialization.ChainCID, validatorConfigsChainC),
			},
			withIBC(baseSetup), // base set up with IBC
			containerManager,
		), nil
	}

	// configure one chain from current Git branch
	return NewCurrentBranchConfigurer(t,
		[]*chain.Config{
			chain.New(t, containerManager, initialization.ChainAID, validatorConfigsChainA),
		},
		baseSetup, // base set up only
		containerManager,
	), nil
}
