package e2e

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	configurer "github.com/classic-terra/core/v2/tests/e2e/configurer"
	"github.com/classic-terra/core/v2/types/util"
)

const (
	// Environment variable signifying whether to run e2e tests.
	e2eEnabledEnv = "TERRA_E2E"
	// Environment variable name to skip the upgrade tests
	skipUpgradeEnv = "TERRA_E2E_SKIP_UPGRADE"
	// Environment variable name to skip the IBC tests
	skipIBCEnv = "TERRA_E2E_SKIP_IBC"
	// Environment variable name to determine if this upgrade is a fork
	forkHeightEnv = "TERRA_E2E_FORK_HEIGHT"
	// Environment variable name to skip cleaning up Docker resources in teardown
	skipCleanupEnv = "TERRA_E2E_SKIP_CLEANUP"
	// Environment variable name to determine what version we are upgrading to
	upgradeVersionEnv = "TERRA_E2E_UPGRADE_VERSION"
)

func init() {
	SetAddressPrefixes()
}

// SetAddressPrefixes builds the Config with Bech32 addressPrefix and publKeyPrefix for accounts, validators, and consensus nodes and verifies that addreeses have correct format.
func SetAddressPrefixes() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(util.Bech32PrefixAccAddr, util.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(util.Bech32PrefixValAddr, util.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(util.Bech32PrefixConsAddr, util.Bech32PrefixConsPub)

	// This is copied from the cosmos sdk v0.43.0-beta1
	// source: https://github.com/cosmos/cosmos-sdk/blob/v0.43.0-beta1/types/address.go#L141
	config.SetAddressVerifier(func(bytes []byte) error {
		if len(bytes) == 0 {
			return sdkerrors.Wrap(sdkerrors.ErrUnknownAddress, "addresses cannot be empty")
		}

		if len(bytes) > address.MaxAddrLen {
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "address max length is %d, got %d", address.MaxAddrLen, len(bytes))
		}

		// TODO: Do we want to allow addresses of lengths other than 20 and 32 bytes?
		if len(bytes) != 20 && len(bytes) != 32 {
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "address length must be 20 or 32 bytes, got %d", len(bytes))
		}

		return nil
	})
}

type IntegrationTestSuite struct {
	suite.Suite

	configurer    configurer.Configurer
	skipUpgrade   bool
	skipIBC       bool
	skipStateSync bool
}

func TestIntegrationTestSuite(t *testing.T) {
	isEnabled := os.Getenv(e2eEnabledEnv)
	if isEnabled != "True" {
		t.Skipf("e2e test is disabled. To run, set %s to True", e2eEnabledEnv)
	}
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up e2e integration test suite...")
	var (
		err             error
		upgradeSettings configurer.UpgradeSettings
	)

	// The e2e test flow is as follows:
	//
	// 1. Configure two chains - chan A and chain B.
	//   * For each chain, set up several validator nodes
	//   * Initialize configs and genesis for all them.
	// 2. Start both networks.
	// 3. Run IBC relayer betweeen the two chains.
	// 4. Execute various e2e tests, including IBC, upgrade, superfluid.
	if str := os.Getenv(skipUpgradeEnv); len(str) > 0 {
		s.skipUpgrade, err = strconv.ParseBool(str)
		s.Require().NoError(err)
		if s.skipUpgrade {
			s.T().Logf("%s was true, skipping upgrade tests", skipUpgradeEnv)
		}
	}
	upgradeSettings.IsEnabled = !s.skipUpgrade

	if str := os.Getenv(forkHeightEnv); len(str) > 0 {
		upgradeSettings.ForkHeight, err = strconv.ParseInt(str, 0, 64)
		s.Require().NoError(err)
		s.T().Logf("fork upgrade is enabled, %s was set to height %d", forkHeightEnv, upgradeSettings.ForkHeight)
	}

	if str := os.Getenv(skipIBCEnv); len(str) > 0 {
		s.skipIBC, err = strconv.ParseBool(str)
		s.Require().NoError(err)
		if s.skipIBC {
			s.T().Logf("%s was true, skipping IBC tests", skipIBCEnv)
		}
	}

	if str := os.Getenv("TERRA_E2E_SKIP_STATE_SYNC"); len(str) > 0 {
		s.skipStateSync, err = strconv.ParseBool(str)
		s.Require().NoError(err)
		if s.skipStateSync {
			s.T().Log("skipping state sync testing")
		}
	}

	isDebugLogEnabled := false
	if str := os.Getenv("TERRA_E2E_DEBUG_LOG"); len(str) > 0 {
		isDebugLogEnabled, err = strconv.ParseBool(str)
		s.Require().NoError(err)
		if isDebugLogEnabled {
			s.T().Log("debug logging is enabled. container logs from running cli commands will be printed to stdout")
		}
	}

	if str := os.Getenv(upgradeVersionEnv); len(str) > 0 {
		upgradeSettings.Version = str
		s.T().Logf("upgrade version set to %s", upgradeSettings.Version)
	}

	s.configurer, err = configurer.New(s.T(), !s.skipIBC, isDebugLogEnabled, upgradeSettings)
	s.Require().NoError(err)

	err = s.configurer.ConfigureChains()
	s.Require().NoError(err)

	err = s.configurer.RunSetup()
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if str := os.Getenv(skipCleanupEnv); len(str) > 0 {
		skipCleanup, err := strconv.ParseBool(str)
		s.Require().NoError(err)

		if skipCleanup {
			s.T().Log("skipping e2e resources clean up...")
			return
		}
	}

	err := s.configurer.ClearResources()
	s.Require().NoError(err)
}
