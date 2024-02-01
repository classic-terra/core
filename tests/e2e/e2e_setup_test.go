package e2e

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"

	configurer "github.com/classic-terra/core/v2/tests/e2e/configurer"
)

const (
	// Environment variable signifying whether to run e2e tests.
	e2eEnabledEnv = "TERRA_E2E"
	// Environment variable name to skip the IBC tests
	skipIBCEnv = "TERRA_E2E_SKIP_IBC"
	// Environment variable name to skip cleaning up Docker resources in teardown
	skipCleanupEnv = "TERRA_E2E_SKIP_CLEANUP"
)

type IntegrationTestSuite struct {
	suite.Suite

	configurer    configurer.Configurer
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
	var err error

	// The e2e test flow is as follows:
	//
	// 1. Configure two chains - chan A and chain B.
	//   * For each chain, set up several validator nodes
	//   * Initialize configs and genesis for all them.
	// 2. Start both networks.
	// 3. Run IBC relayer betweeen the two chains.
	// 4. Execute various e2e tests, including IBC.
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

	s.configurer, err = configurer.New(s.T(), !s.skipIBC, isDebugLogEnabled)
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
