//nolint:revive
package v14_6_test

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	apptesting "github.com/classic-terra/core/v4/app/testing"
	v14_6 "github.com/classic-terra/core/v4/app/upgrades/v14_6"
	core "github.com/classic-terra/core/v4/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/suite"
)

type UpgradeTestSuite struct {
	apptesting.KeeperTestHelper
}

func TestUpgradeTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

func (s *UpgradeTestSuite) SetupTest() {
	s.Setup(s.T(), core.ColumbusChainID)

	// The test app starts without a stored module version map, so RunMigrations
	// would call InitGenesis for every module and reset wasm params to their
	// defaults. Store the current versions, as a live chain has them, so only
	// the handler's own change is observed.
	vm := module.VersionMap{}
	for name, m := range s.App.Modules() {
		if v, ok := m.(module.HasConsensusVersion); ok {
			vm[name] = v.ConsensusVersion()
		}
	}
	s.Require().NotEmpty(vm)
	s.Require().NoError(s.App.UpgradeKeeper.SetModuleVersionMap(s.Ctx, vm))
}

// applyUpgrade runs the registered v14_6 handler through the upgrade keeper,
// the same path the chain takes at the upgrade height.
func (s *UpgradeTestSuite) applyUpgrade() {
	plan := upgradetypes.Plan{Name: v14_6.UpgradeName, Height: s.Ctx.BlockHeight()}
	s.Require().NoError(s.App.UpgradeKeeper.ApplyUpgrade(s.Ctx, plan))
}

func (s *UpgradeTestSuite) setParams(p wasmtypes.Params) {
	s.Require().NoError(s.App.WasmKeeper.SetParams(s.Ctx, p))
}

// TestOpensUploadFromGovOnly mirrors mainnet: upload restricted to governance
// (Nobody), instantiation open to everybody.
func (s *UpgradeTestSuite) TestOpensUploadFromGovOnly() {
	s.setParams(wasmtypes.Params{
		CodeUploadAccess:             wasmtypes.AllowNobody,
		InstantiateDefaultPermission: wasmtypes.AccessTypeEverybody,
	})

	s.applyUpgrade()

	p := s.App.WasmKeeper.GetParams(s.Ctx)
	s.Require().Equal(wasmtypes.AllowEverybody, p.CodeUploadAccess)
	s.Require().Equal(wasmtypes.AccessTypeEverybody, p.InstantiateDefaultPermission)
}

// TestOpensUploadFromAddressList: an AnyOfAddresses allow-list is replaced and
// its addresses are dropped.
func (s *UpgradeTestSuite) TestOpensUploadFromAddressList() {
	s.setParams(wasmtypes.Params{
		CodeUploadAccess:             wasmtypes.AccessTypeAnyOfAddresses.With(s.TestAccs[0], s.TestAccs[1]),
		InstantiateDefaultPermission: wasmtypes.AccessTypeEverybody,
	})

	s.applyUpgrade()

	p := s.App.WasmKeeper.GetParams(s.Ctx)
	s.Require().Equal(wasmtypes.AllowEverybody, p.CodeUploadAccess)
	s.Require().Empty(p.CodeUploadAccess.Addresses)
}

// TestKeepsInstantiatePermission: only code_upload_access is changed.
func (s *UpgradeTestSuite) TestKeepsInstantiatePermission() {
	s.setParams(wasmtypes.Params{
		CodeUploadAccess:             wasmtypes.AllowNobody,
		InstantiateDefaultPermission: wasmtypes.AccessTypeNobody,
	})

	s.applyUpgrade()

	p := s.App.WasmKeeper.GetParams(s.Ctx)
	s.Require().Equal(wasmtypes.AllowEverybody, p.CodeUploadAccess)
	s.Require().Equal(wasmtypes.AccessTypeNobody, p.InstantiateDefaultPermission)
}

// TestIdempotent: running on params that already allow everybody is a no-op.
func (s *UpgradeTestSuite) TestIdempotent() {
	s.setParams(wasmtypes.DefaultParams())
	before := s.App.WasmKeeper.GetParams(s.Ctx)

	s.Require().NoError(v14_6.AllowCodeUploadForEverybody(s.Ctx, s.App.AppKeepers))

	s.Require().Equal(before, s.App.WasmKeeper.GetParams(s.Ctx))
}
