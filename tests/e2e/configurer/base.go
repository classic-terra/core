package configurer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/classic-terra/core/v2/tests/e2e/configurer/chain"
	"github.com/classic-terra/core/v2/tests/e2e/containers"
	"github.com/classic-terra/core/v2/tests/e2e/initialization"
	"github.com/classic-terra/core/v2/tests/e2e/util"
)

// baseConfigurer is the base implementation for the
// other 2 types of configurers. It is not meant to be used
// on its own. Instead, it is meant to be embedded
// by composition into more concrete configurers.
type baseConfigurer struct {
	chainConfigs     []*chain.Config
	containerManager *containers.Manager
	setupTests       setupFn
	syncUntilHeight  int64 // the height until which to wait for validators to sync when first started.
	t                *testing.T
}

// defaultSyncUntilHeight arbitrary small height to make sure the chain is making progress.
const defaultSyncUntilHeight = 3

func (bc *baseConfigurer) ClearResources() error {
	bc.t.Log("tearing down e2e integration test suite...")

	if err := bc.containerManager.ClearResources(); err != nil {
		return err
	}

	for _, chainConfig := range bc.chainConfigs {
		os.RemoveAll(chainConfig.DataDir)
	}
	return nil
}

func (bc *baseConfigurer) GetChainConfig(chainIndex int) *chain.Config {
	return bc.chainConfigs[chainIndex]
}

func (bc *baseConfigurer) RunValidators() error {
	for _, chainConfig := range bc.chainConfigs {
		if err := bc.runValidators(chainConfig); err != nil {
			return err
		}
	}
	return nil
}

func (bc *baseConfigurer) runValidators(chainConfig *chain.Config) error {
	bc.t.Logf("starting %s validator containers...", chainConfig.ID)
	for _, node := range chainConfig.NodeConfigs {
		if err := node.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (bc *baseConfigurer) RunIBC() error {
	bc.t.Log("Run relayer 1 between chain a and chain b")
	if err := bc.runIBCRelayer1(bc.chainConfigs[0], bc.chainConfigs[1]); err != nil {
		return err
	}
	bc.t.Log("Run relayer 2 between chain b and chain c")
	if err := bc.runIBCRelayer2(bc.chainConfigs[1], bc.chainConfigs[2]); err != nil {
		return err
	}

	return nil
}

func (bc *baseConfigurer) runIBCRelayer1(chainConfigA *chain.Config, chainConfigB *chain.Config) error {
	bc.t.Log("starting Hermes relayer 1 container...")

	tmpDir, err := os.MkdirTemp("", "terra-e2e-testnet-hermes-1")
	if err != nil {
		return err
	}

	hermesCfgPath := path.Join(tmpDir, "hermes")

	if err := os.MkdirAll(hermesCfgPath, 0o755); err != nil {
		return err
	}

	_, err = util.CopyFile(
		filepath.Join("./scripts/", "hermes_bootstrap.sh"),
		filepath.Join(hermesCfgPath, "hermes_bootstrap.sh"),
	)
	if err != nil {
		return err
	}

	relayerNodeA := chainConfigA.NodeConfigs[0]
	relayerNodeB := chainConfigB.NodeConfigs[0]

	err = util.WritePublicFile(filepath.Join(hermesCfgPath, "mnemonicA.json"), []byte(relayerNodeA.Mnemonic))
	if err != nil {
		return err
	}

	err = util.WritePublicFile(filepath.Join(hermesCfgPath, "mnemonicB.json"), []byte(relayerNodeB.Mnemonic))
	if err != nil {
		return err
	}

	hermesResource, err := bc.containerManager.RunHermesResource1(
		chainConfigA.ID,
		relayerNodeA.Name,
		filepath.Join("/root/hermes", "mnemonicA.json"),
		chainConfigB.ID,
		relayerNodeB.Name,
		filepath.Join("/root/hermes", "mnemonicB.json"),
		hermesCfgPath)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("http://%s/state", hermesResource.GetHostPort("3031/tcp"))

	require.Eventually(bc.t, func() bool {
		resp, err := http.Get(endpoint) //nolint
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

		status, ok := respBody["status"].(string)
		require.True(bc.t, ok)
		result, ok := respBody["result"].(map[string]interface{})
		require.True(bc.t, ok)

		chains, ok := result["chains"].([]interface{})
		require.True(bc.t, ok)

		return status == "success" && len(chains) == 2
	},
		1*time.Minute,
		time.Second,
		"hermes relayer not healthy")

	bc.t.Logf("started Hermes relayer container: %s", hermesResource.Container.ID)

	// XXX: Give time to both networks to start, otherwise we might see gRPC
	// transport errors.
	time.Sleep(10 * time.Second)

	// create the client, connection and channel between the two Terra chains
	return bc.connectIBCChains(chainConfigA, chainConfigB)
}

func (bc *baseConfigurer) runIBCRelayer2(chainConfigA *chain.Config, chainConfigB *chain.Config) error {
	bc.t.Log("starting Hermes relayer 2 container...")

	tmpDir, err := os.MkdirTemp("", "terra-e2e-testnet-hermes-2")
	if err != nil {
		return err
	}

	hermesCfgPath := path.Join(tmpDir, "hermes")

	if err := os.MkdirAll(hermesCfgPath, 0o755); err != nil {
		return err
	}

	_, err = util.CopyFile(
		filepath.Join("./scripts/", "hermes_bootstrap.sh"),
		filepath.Join(hermesCfgPath, "hermes_bootstrap.sh"),
	)
	if err != nil {
		return err
	}

	relayerNodeA := chainConfigA.NodeConfigs[0]
	relayerNodeB := chainConfigB.NodeConfigs[0]

	err = util.WritePublicFile(filepath.Join(hermesCfgPath, "mnemonicA.json"), []byte(relayerNodeA.Mnemonic))
	if err != nil {
		return err
	}
	err = util.WritePublicFile(filepath.Join(hermesCfgPath, "mnemonicB.json"), []byte(relayerNodeB.Mnemonic))
	if err != nil {
		return err
	}

	hermesResource, err := bc.containerManager.RunHermesResource2(
		chainConfigA.ID,
		relayerNodeA.Name,
		filepath.Join("/root/hermes", "mnemonicA.json"),
		chainConfigB.ID,
		relayerNodeB.Name,
		filepath.Join("/root/hermes", "mnemonicB.json"),
		hermesCfgPath)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("http://%s/state", hermesResource.GetHostPort("3031/tcp"))

	require.Eventually(bc.t, func() bool {
		resp, err := http.Get(endpoint) //nolint
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

		status, ok := respBody["status"].(string)
		require.True(bc.t, ok)
		result, ok := respBody["result"].(map[string]interface{})
		require.True(bc.t, ok)

		chains, ok := result["chains"].([]interface{})
		require.True(bc.t, ok)

		return status == "success" && len(chains) == 2
	},
		1*time.Minute,
		time.Second,
		"hermes relayer not healthy")

	bc.t.Logf("started Hermes relayer container: %s", hermesResource.Container.ID)

	// XXX: Give time to both networks to start, otherwise we might see gRPC
	// transport errors.
	time.Sleep(10 * time.Second)

	// create the client, connection and channel between the two Terra chains
	return bc.connectIBCChains2(chainConfigA, chainConfigB)
}

func (bc *baseConfigurer) connectIBCChains(chainA *chain.Config, chainB *chain.Config) error {
	bc.t.Logf("connecting %s and %s chains via IBC", chainA.ChainMeta.ID, chainB.ChainMeta.ID)

	cmd := []string{"hermes", "create", "channel", "--a-chain", chainA.ChainMeta.ID, "--b-chain", chainB.ChainMeta.ID, "--a-port", "transfer", "--b-port", "transfer", "--new-client-connection", "--yes"}
	bc.t.Log(cmd)
	_, _, err := bc.containerManager.ExecHermesCmd1(bc.t, cmd, "SUCCESS")
	if err != nil {
		return err
	}
	bc.t.Logf("connected %s and %s chains via IBC", chainA.ChainMeta.ID, chainB.ChainMeta.ID)
	return nil
}

func (bc *baseConfigurer) connectIBCChains2(chainA *chain.Config, chainB *chain.Config) error {
	bc.t.Logf("connecting %s and %s chains via IBC", chainA.ChainMeta.ID, chainB.ChainMeta.ID)

	cmd := []string{"hermes", "create", "channel", "--a-chain", chainA.ChainMeta.ID, "--b-chain", chainB.ChainMeta.ID, "--a-port", "transfer", "--b-port", "transfer", "--new-client-connection", "--yes"}
	bc.t.Log(cmd)
	_, _, err := bc.containerManager.ExecHermesCmd2(bc.t, cmd, "SUCCESS")
	if err != nil {
		return err
	}
	bc.t.Logf("connected %s and %s chains via IBC", chainA.ChainMeta.ID, chainB.ChainMeta.ID)
	return nil
}

func (bc *baseConfigurer) initializeChainConfigFromInitChain(initializedChain *initialization.Chain, chainConfig *chain.Config) {
	chainConfig.ChainMeta = initializedChain.ChainMeta
	chainConfig.NodeConfigs = make([]*chain.NodeConfig, 0, len(initializedChain.Nodes))
	setupTime := time.Now()
	for i, validator := range initializedChain.Nodes {
		conf := chain.NewNodeConfig(bc.t, validator, chainConfig.ValidatorInitConfigs[i], chainConfig.ID, bc.containerManager).WithSetupTime(setupTime)
		chainConfig.NodeConfigs = append(chainConfig.NodeConfigs, conf)
	}
}
