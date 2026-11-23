package app_test

import (
	"encoding/json"
	"os"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/classic-terra/core/v4/app"
	apptesting "github.com/classic-terra/core/v4/app/testing"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type TestnetWasmAdminSuite struct {
	apptesting.KeeperTestHelper
}

func TestTestnetWasmAdminSuite(t *testing.T) {
	suite.Run(t, new(TestnetWasmAdminSuite))
}

func (s *TestnetWasmAdminSuite) SetupTest() {
	s.Setup(s.T(), apptesting.SimAppChainID)
}

// instantiateHackatom stores hackatom and instantiates it with the given admin.
func (s *TestnetWasmAdminSuite) instantiateHackatom(creator, admin sdk.AccAddress) sdk.AccAddress {
	code, err := os.ReadFile("../contrib/wasm/hackatom.wasm")
	s.Require().NoError(err)
	k := wasmkeeper.NewDefaultPermissionKeeper(s.App.WasmKeeper)
	codeID, _, err := k.Create(s.Ctx, creator, code, &wasmtypes.AllowEverybody)
	s.Require().NoError(err)
	msg, err := json.Marshal(map[string]string{"verifier": creator.String(), "beneficiary": creator.String()})
	s.Require().NoError(err)
	addr, _, err := k.Instantiate(s.Ctx, codeID, creator, admin, msg, "hackatom", nil)
	s.Require().NoError(err)
	return addr
}

func (s *TestnetWasmAdminSuite) TestReplacesAdmins() {
	accs := s.RandomAccountAddresses(4)
	creator, originalAdmin, tester, other := accs[0], accs[1], accs[2], accs[3]
	s.FundAcc(creator, sdk.NewCoins(sdk.NewInt64Coin("uluna", 1_000_000_000)))

	withAdmin := s.instantiateHackatom(creator, originalAdmin)
	withoutAdmin := s.instantiateHackatom(creator, nil)
	untouched := s.instantiateHackatom(creator, originalAdmin)

	err := app.ApplyWasmAdminOverrides(s.Ctx, s.App.WasmKeeper, []app.WasmAdminOverride{
		{Contract: withAdmin, NewAdmin: tester},
		{Contract: withoutAdmin, NewAdmin: other},
	})
	s.Require().NoError(err)

	s.Require().Equal(tester.String(), s.App.WasmKeeper.GetContractInfo(s.Ctx, withAdmin).Admin)
	s.Require().Equal(other.String(), s.App.WasmKeeper.GetContractInfo(s.Ctx, withoutAdmin).Admin)
	s.Require().Equal(originalAdmin.String(), s.App.WasmKeeper.GetContractInfo(s.Ctx, untouched).Admin)

	// The new admin can now migrate the contract with an ordinary admin check.
	codeID := s.App.WasmKeeper.GetContractInfo(s.Ctx, withAdmin).CodeID
	k := wasmkeeper.NewDefaultPermissionKeeper(s.App.WasmKeeper)
	_, err = k.Migrate(s.Ctx, withAdmin, originalAdmin, codeID, []byte(`{"verifier":"`+tester.String()+`"}`))
	s.Require().Error(err, "the old admin must lose migrate rights")
	_, err = k.Migrate(s.Ctx, withAdmin, tester, codeID, []byte(`{"verifier":"`+tester.String()+`"}`))
	s.Require().NoError(err)
}

func (s *TestnetWasmAdminSuite) TestUnknownContractFails() {
	accs := s.RandomAccountAddresses(2)
	err := app.ApplyWasmAdminOverrides(s.Ctx, s.App.WasmKeeper, []app.WasmAdminOverride{
		{Contract: accs[0], NewAdmin: accs[1]},
	})
	s.Require().ErrorContains(err, "not found")
}

func (s *TestnetWasmAdminSuite) TestNoOverridesIsNoop() {
	s.Require().NoError(app.ApplyWasmAdminOverrides(s.Ctx, s.App.WasmKeeper, nil))
}
