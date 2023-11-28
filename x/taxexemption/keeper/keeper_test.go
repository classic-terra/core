package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto/secp256k1"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v2/x/taxexemption/types"
)

func TestTaxExemptionList(t *testing.T) {
	input := CreateTestInput(t)

	require.False(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, "", ""))
	require.Panics(t, func() { input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "", "") })
	require.Panics(t, func() { input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, "", "") })

	pubKey := secp256k1.GenPrivKey().PubKey()
	pubKey2 := secp256k1.GenPrivKey().PubKey()
	pubKey3 := secp256k1.GenPrivKey().PubKey()
	pubKey4 := secp256k1.GenPrivKey().PubKey()
	pubKey5 := secp256k1.GenPrivKey().PubKey()
	address := sdk.AccAddress(pubKey.Address())
	address2 := sdk.AccAddress(pubKey2.Address())
	address3 := sdk.AccAddress(pubKey3.Address())
	address4 := sdk.AccAddress(pubKey4.Address())
	address5 := sdk.AccAddress(pubKey5.Address())

	// add a zone
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone1", Outgoing: false, Incoming: false, CrossZone: false})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone2", Outgoing: true, Incoming: false, CrossZone: false})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone3", Outgoing: false, Incoming: true, CrossZone: true})

	// add an address
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "zone1", address.String())
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "zone1", address2.String())
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "zone2", address3.String())
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "zone3", address5.String())

	require.True(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address.String(), address2.String()))
	require.False(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address.String(), address3.String()))
	require.False(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address.String(), address4.String()))

	// zone 2 allows outgoing, address 4 is not in a zone
	require.True(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address3.String(), address4.String()))

	require.False(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address3.String(), address.String()))
	require.False(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address5.String(), address.String()))

	// zone 3 allows incoming and cross zone
	require.True(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address.String(), address5.String()))

	// add it again
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "zone1", address.String())
	require.True(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address.String(), address2.String()))

	// remove it
	input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, "zone1", address.String())
	require.False(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address.String(), address2.String()))
}
