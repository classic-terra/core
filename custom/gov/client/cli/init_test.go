package cli_test

import (
	"io"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/stretchr/testify/suite"
	// "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
)

type CLITestSuite struct {
	suite.Suite

	kr      keyring.Keyring
	encCfg  testutilmod.TestEncodingConfig // This holds the encoding configuration, which is crucial for marshaling and unmarshaling data for transactions and messages.
	baseCtx client.Context                 // This is a base context used for all operations in the test suite.
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

// run once before any tests in the suite
func (s *CLITestSuite) SetupSuite() {
	s.encCfg = testutilmod.MakeTestEncodingConfig(gov.AppModuleBasic{})
	s.kr = keyring.NewInMemory(s.encCfg.Codec)
	s.baseCtx = client.Context{}.
		WithKeyring(s.kr).
		WithTxConfig(s.encCfg.TxConfig).
		WithCodec(s.encCfg.Codec).
		WithClient(clitestutil.MockTendermintRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(client.MockAccountRetriever{}).
		WithOutput(io.Discard).
		WithChainID("test-chain")
}
