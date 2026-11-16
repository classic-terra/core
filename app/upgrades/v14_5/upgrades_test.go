//nolint:revive
package v14_5_test

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	apptesting "github.com/classic-terra/core/v4/app/testing"
	v14_5 "github.com/classic-terra/core/v4/app/upgrades/v14_5"
	core "github.com/classic-terra/core/v4/types"
	taxexemptiontypes "github.com/classic-terra/core/v4/x/taxexemption/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
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
}

func (s *UpgradeTestSuite) members() ([]string, error) {
	addrs, _, err := s.App.TaxExemptionKeeper.ListTaxExemptionAddresses(s.Ctx,
		&taxexemptiontypes.QueryTaxExemptionAddressRequest{
			ZoneName:   v14_5.BinanceZone,
			Pagination: &query.PageRequest{Limit: 1000},
		})
	return addrs, err
}

// applyUpgrade runs the registered v14_5 handler through the upgrade keeper,
// the same path the chain takes at the upgrade height.
func (s *UpgradeTestSuite) applyUpgrade() {
	plan := upgradetypes.Plan{Name: v14_5.UpgradeName, Height: s.Ctx.BlockHeight()}
	s.Require().NoError(s.App.UpgradeKeeper.ApplyUpgrade(s.Ctx, plan))
}

func (s *UpgradeTestSuite) assertWalletsExempt(existingMember string) {
	k := s.App.TaxExemptionKeeper
	outsider := sdk.AccAddress([]byte("outsider_address____")).String()

	members, err := s.members()
	s.Require().NoError(err)
	for _, addr := range v14_5.BinanceWalletAddresses {
		s.Require().Contains(members, addr)
		// zone flags are all false: exempt within the zone only
		s.Require().True(k.IsExemptedFromTax(s.Ctx, addr, existingMember))
		s.Require().True(k.IsExemptedFromTax(s.Ctx, existingMember, addr))
		s.Require().False(k.IsExemptedFromTax(s.Ctx, addr, outsider))
		s.Require().False(k.IsExemptedFromTax(s.Ctx, outsider, addr))
	}
	s.Require().True(k.IsExemptedFromTax(s.Ctx, v14_5.BinanceWalletAddresses[0], v14_5.BinanceWalletAddresses[1]))
}

// TestAddsWalletsToExistingZone mirrors mainnet: the Binance zone already
// exists with its v12 flags and members, which must be preserved.
func (s *UpgradeTestSuite) TestAddsWalletsToExistingZone() {
	k := s.App.TaxExemptionKeeper
	existing := s.TestAccs[0].String()
	s.Require().NoError(k.AddTaxExemptionZone(s.Ctx, taxexemptiontypes.Zone{Name: v14_5.BinanceZone}))
	s.Require().NoError(k.AddTaxExemptionAddress(s.Ctx, v14_5.BinanceZone, existing))

	s.applyUpgrade()

	zone, err := k.GetTaxExemptionZone(s.Ctx, v14_5.BinanceZone)
	s.Require().NoError(err)
	s.Require().Equal(taxexemptiontypes.Zone{Name: v14_5.BinanceZone}, zone, "zone flags must be unchanged")

	members, err := s.members()
	s.Require().NoError(err)
	s.Require().Len(members, 1+len(v14_5.BinanceWalletAddresses))
	s.Require().Contains(members, existing)

	s.assertWalletsExempt(existing)
}

// TestCreatesZoneIfMissing covers networks that never ran v12.
func (s *UpgradeTestSuite) TestCreatesZoneIfMissing() {
	_, err := s.App.TaxExemptionKeeper.GetTaxExemptionZone(s.Ctx, v14_5.BinanceZone)
	s.Require().Error(err)

	s.applyUpgrade()

	zone, err := s.App.TaxExemptionKeeper.GetTaxExemptionZone(s.Ctx, v14_5.BinanceZone)
	s.Require().NoError(err)
	s.Require().False(zone.Outgoing || zone.Incoming || zone.CrossZone)
	s.assertWalletsExempt(v14_5.BinanceWalletAddresses[0])
}

// TestIdempotentForWalletAlreadyInZone: a wallet already in the Binance zone
// does not abort the upgrade.
func (s *UpgradeTestSuite) TestIdempotentForWalletAlreadyInZone() {
	k := s.App.TaxExemptionKeeper
	s.Require().NoError(k.AddTaxExemptionZone(s.Ctx, taxexemptiontypes.Zone{Name: v14_5.BinanceZone}))
	s.Require().NoError(k.AddTaxExemptionAddress(s.Ctx, v14_5.BinanceZone, v14_5.BinanceWalletAddresses[0]))

	s.applyUpgrade()

	members, err := s.members()
	s.Require().NoError(err)
	s.Require().Len(members, len(v14_5.BinanceWalletAddresses))
}

// TestFailsIfWalletInOtherZone: the handler must not silently move an
// address out of another zone.
func (s *UpgradeTestSuite) TestFailsIfWalletInOtherZone() {
	k := s.App.TaxExemptionKeeper
	s.Require().NoError(k.AddTaxExemptionZone(s.Ctx, taxexemptiontypes.Zone{Name: "Other"}))
	s.Require().NoError(k.AddTaxExemptionAddress(s.Ctx, "Other", v14_5.BinanceWalletAddresses[1]))

	err := v14_5.AddBinanceWallets(s.Ctx, s.App.AppKeepers)
	s.Require().ErrorContains(err, "already associated with a different zone")
}

// TestAddressesAreValid guards against typos in the hardcoded list.
func (s *UpgradeTestSuite) TestAddressesAreValid() {
	seen := map[string]bool{}
	for _, addr := range v14_5.BinanceWalletAddresses {
		_, err := sdk.AccAddressFromBech32(addr)
		s.Require().NoError(err, addr)
		s.Require().False(seen[addr], "duplicate %s", addr)
		seen[addr] = true
	}
}
