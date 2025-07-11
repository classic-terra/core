package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ultil "github.com/classic-terra/core/v3/x/taxexemption/keeper"
	"github.com/classic-terra/core/v3/x/taxexemption/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/stretchr/testify/require"
)

func Test_AddTaxExemptionZone(t *testing.T) {
	input := ultil.CreateTestInput(t)
	k := input.TaxExemptionKeeper
	ctx := input.Ctx

	server := ultil.NewMsgServerImpl(k)
	authority := k.GetAuthority()
	// Check empty zones
	msgEmpty := types.MsgAddTaxExemptionZone{
		Zone:      "",
		Authority: authority,
	}
	resp, err := server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &msgEmpty)
	require.Error(t, err)
	require.Contains(t, err.Error(), "zone name cannot be empty")
	require.Nil(t, resp)
	// Test case 1: Adding a zone with valid authority
	msg := types.MsgAddTaxExemptionZone{
		Zone:      "zone1",
		Authority: authority,
		Outgoing:  false,
		Incoming:  false,
		CrossZone: false,
	}

	resp, err = server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify zone was added with correct properties
	zone, err := k.GetTaxExemptionZone(ctx, "zone1")
	require.NoError(t, err)
	require.Equal(t, "zone1", zone.Name)
	require.False(t, zone.Outgoing)
	require.False(t, zone.Incoming)
	require.False(t, zone.CrossZone)

	// Test case 2: Adding a zone with invalid authority
	invalidMsg := types.MsgAddTaxExemptionZone{
		Zone:      "invalid_zone",
		Authority: "invalid_authority",
	}

	resp, err = server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &invalidMsg)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "invalid authority")

	// Test case 3: Adding another valid zone with different properties
	msg2 := types.MsgAddTaxExemptionZone{
		Zone:      "zone2",
		Authority: authority,
		Outgoing:  false,
		Incoming:  true,
		CrossZone: true,
	}

	resp2, err := server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &msg2)
	require.NoError(t, err)
	require.NotNil(t, resp2)

	// Verify zone2 was added with correct properties
	zone2, err := k.GetTaxExemptionZone(ctx, "zone2")
	require.NoError(t, err)
	require.Equal(t, "zone2", zone2.Name)
	require.False(t, zone2.Outgoing)
	require.True(t, zone2.Incoming)
	require.True(t, zone2.CrossZone)

	// Test case 4: Try adding an existing zone update the zone properties
	duplicateMsg := types.MsgAddTaxExemptionZone{
		Zone:      "zone1",
		Authority: authority,
		Outgoing:  true,
		Incoming:  true,
		CrossZone: true,
	}

	resp, err = server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &duplicateMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify zone1 was updated with new properties
	updatedZone, err := k.GetTaxExemptionZone(ctx, "zone1")
	require.NoError(t, err)
	require.Equal(t, "zone1", updatedZone.Name)
	require.True(t, updatedZone.Outgoing)
	require.True(t, updatedZone.Incoming)
	require.True(t, updatedZone.CrossZone)
}

func TestMsgServer_RemoveTaxExemptionZone(t *testing.T) {
	input := ultil.CreateTestInput(t)
	k := input.TaxExemptionKeeper
	ctx := input.Ctx

	server := ultil.NewMsgServerImpl(k)
	authority := k.GetAuthority()
	// Check empty zones
	msgEmpty := types.MsgRemoveTaxExemptionZone{
		Zone:      "",
		Authority: authority,
	}
	respEmpty, errEmpty := server.RemoveTaxExemptionZone(sdk.WrapSDKContext(ctx), &msgEmpty)
	require.Error(t, errEmpty)
	require.Contains(t, errEmpty.Error(), "zone name cannot be empty")
	require.Nil(t, respEmpty)

	// First add a zone
	addMsg := types.MsgAddTaxExemptionZone{
		Zone:      "zone1",
		Authority: authority,
		Outgoing:  true,
		Incoming:  true,
		CrossZone: false,
	}

	_, err := server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &addMsg)
	require.NoError(t, err)

	// Verify zone was added
	zone, err := k.GetTaxExemptionZone(ctx, "zone1")
	require.NoError(t, err)
	require.Equal(t, "zone1", zone.Name)

	// Test removing the zone
	removeMsg := types.MsgRemoveTaxExemptionZone{
		Zone:      "zone1",
		Authority: authority,
	}

	resp, err := server.RemoveTaxExemptionZone(sdk.WrapSDKContext(ctx), &removeMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify zone was removed (should return error as zone doesn't exist anymore)
	_, err = k.GetTaxExemptionZone(ctx, "zone1")
	require.Error(t, err)

	// Test removing non-existent zone
	removeMsg = types.MsgRemoveTaxExemptionZone{
		Zone:      "nonexistent",
		Authority: authority,
	}

	resp, err = server.RemoveTaxExemptionZone(sdk.WrapSDKContext(ctx), &removeMsg)
	require.Error(t, err)
	require.Nil(t, resp)

	// Test with invalid authority
	invalidMsg := types.MsgRemoveTaxExemptionZone{
		Zone:      "zone1",
		Authority: "invalid_authority",
	}

	resp, err = server.RemoveTaxExemptionZone(sdk.WrapSDKContext(ctx), &invalidMsg)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "invalid authority")
}

func TestMsgServer_ModifyTaxExemptionZone(t *testing.T) {
	input := ultil.CreateTestInput(t)
	k := input.TaxExemptionKeeper
	ctx := input.Ctx

	server := ultil.NewMsgServerImpl(k)
	authority := k.GetAuthority()

	// Check empty zones
	msgEmpty := types.MsgModifyTaxExemptionZone{
		Zone:      "",
		Authority: authority,
	}
	respEmpty, errEmpty := server.ModifyTaxExemptionZone(sdk.WrapSDKContext(ctx), &msgEmpty)
	require.Error(t, errEmpty)
	require.Contains(t, errEmpty.Error(), "zone name cannot be empty")
	require.Nil(t, respEmpty)

	// First add a zone
	addMsg := types.MsgAddTaxExemptionZone{
		Zone:      "zone1",
		Authority: authority,
		Outgoing:  true,
		Incoming:  true,
		CrossZone: false,
	}

	_, err := server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &addMsg)
	require.NoError(t, err)

	// Verify initial zone properties
	zone, err := k.GetTaxExemptionZone(ctx, "zone1")
	require.NoError(t, err)
	require.Equal(t, "zone1", zone.Name)
	require.True(t, zone.Outgoing)
	require.True(t, zone.Incoming)
	require.False(t, zone.CrossZone)

	// Test modifying the zone
	modifyMsg := types.MsgModifyTaxExemptionZone{
		Zone:      "zone1",
		Authority: authority,
		Outgoing:  false,
		Incoming:  false,
		CrossZone: true,
	}

	resp, err := server.ModifyTaxExemptionZone(sdk.WrapSDKContext(ctx), &modifyMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify zone was modified with new properties
	modifiedZone, err := k.GetTaxExemptionZone(ctx, "zone1")
	require.NoError(t, err)
	require.Equal(t, "zone1", modifiedZone.Name)
	require.False(t, modifiedZone.Outgoing)
	require.False(t, modifiedZone.Incoming)
	require.True(t, modifiedZone.CrossZone)

	// Test modifying non-existent zone
	modifyMsg = types.MsgModifyTaxExemptionZone{
		Zone:      "nonexistent",
		Authority: authority,
		Outgoing:  true,
		Incoming:  true,
		CrossZone: true,
	}

	resp, err = server.ModifyTaxExemptionZone(sdk.WrapSDKContext(ctx), &modifyMsg)
	require.Error(t, err)
	require.Nil(t, resp)

	// Test with invalid authority
	invalidMsg := types.MsgModifyTaxExemptionZone{
		Zone:      "zone1",
		Authority: "invalid_authority",
		Outgoing:  true,
		Incoming:  true,
		CrossZone: true,
	}

	resp, err = server.ModifyTaxExemptionZone(sdk.WrapSDKContext(ctx), &invalidMsg)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "invalid authority")
}

func TestMsgServer_AddTaxExemptionAddress(t *testing.T) {
	input := ultil.CreateTestInput(t)
	k := input.TaxExemptionKeeper
	ctx := input.Ctx

	server := ultil.NewMsgServerImpl(k)
	authority := k.GetAuthority()
	pubKey1 := secp256k1.GenPrivKey().PubKey()
	pubKey2 := secp256k1.GenPrivKey().PubKey()
	pubKey3 := secp256k1.GenPrivKey().PubKey()
	address1 := sdk.AccAddress(pubKey1.Address())
	address2 := sdk.AccAddress(pubKey2.Address())
	address3 := sdk.AccAddress(pubKey3.Address())

	// First create a zone to add addresses to
	zoneMsg := types.MsgAddTaxExemptionZone{
		Zone:      "zone1",
		Authority: authority,
		Outgoing:  true,
		Incoming:  true,
		CrossZone: false,
		Addresses: []string{address1.String()},
	}

	_, err := server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &zoneMsg)
	require.NoError(t, err)

	// Test adding addresses to the zone
	addAddressMsg := types.MsgAddTaxExemptionAddress{
		Zone:      "zone1",
		Authority: authority,
		Addresses: []string{address2.String()},
	}

	resp, err := server.AddTaxExemptionAddress(sdk.WrapSDKContext(ctx), &addAddressMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Test adding addresses to non-existent zone
	invalidZoneMsg := types.MsgAddTaxExemptionAddress{
		Zone:      "nonexistent",
		Authority: authority,
		Addresses: []string{address3.String()},
	}

	resp, err = server.AddTaxExemptionAddress(sdk.WrapSDKContext(ctx), &invalidZoneMsg)
	require.Error(t, err)
	require.Nil(t, resp)

	// Test with invalid authority
	invalidAuthMsg := types.MsgAddTaxExemptionAddress{
		Zone:      "zone1",
		Authority: "invalid_authority",
		Addresses: []string{address3.String()},
	}

	resp, err = server.AddTaxExemptionAddress(sdk.WrapSDKContext(ctx), &invalidAuthMsg)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "invalid authority")

	// Test adding same address again should pass without error
	duplicateMsg := types.MsgAddTaxExemptionAddress{
		Zone:      "zone1",
		Authority: authority,
		Addresses: []string{address1.String()},
	}

	_, err = server.AddTaxExemptionAddress(sdk.WrapSDKContext(ctx), &duplicateMsg)
	require.NoError(t, err)

	// Test add same address to different zone should already associated error
	zoneMsg2 := types.MsgAddTaxExemptionZone{
		Zone:      "zone2",
		Authority: authority,
		Outgoing:  true,
		Incoming:  true,
		CrossZone: false,
		Addresses: []string{address1.String()},
	}

	_, err1 := server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &zoneMsg2)
	require.Error(t, err1)
	require.Contains(t, err1.Error(), "already associated")
	otherZoneMsg := types.MsgAddTaxExemptionAddress{
		Zone:      "zone2",
		Authority: authority,
		Addresses: []string{address1.String()},
	}
	resp1, err2 := server.AddTaxExemptionAddress(sdk.WrapSDKContext(ctx), &otherZoneMsg)
	require.Error(t, err2)
	require.Nil(t, resp1)
	require.Contains(t, err2.Error(), "already associated")
}

func TestMsgServer_RemoveTaxExemptionAddress(t *testing.T) {
	input := ultil.CreateTestInput(t)
	k := input.TaxExemptionKeeper
	ctx := input.Ctx

	server := ultil.NewMsgServerImpl(k)
	authority := k.GetAuthority()
	pubKey1 := secp256k1.GenPrivKey().PubKey()
	pubKey2 := secp256k1.GenPrivKey().PubKey()
	pubKey3 := secp256k1.GenPrivKey().PubKey()
	address1 := sdk.AccAddress(pubKey1.Address())
	address2 := sdk.AccAddress(pubKey2.Address())
	address3 := sdk.AccAddress(pubKey3.Address())
	// First create a zone with addresses
	zoneMsg := types.MsgAddTaxExemptionZone{
		Zone:      "zone1",
		Authority: authority,
		Outgoing:  true,
		Incoming:  true,
		CrossZone: false,
		Addresses: []string{address1.String(), address2.String(), address3.String()},
	}

	_, err := server.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), &zoneMsg)
	require.NoError(t, err)

	// Test removing addresses from the zone
	removeMsg := types.MsgRemoveTaxExemptionAddress{
		Zone:      "zone1",
		Authority: authority,
		Addresses: []string{address1.String(), address2.String()},
	}

	resp, err := server.RemoveTaxExemptionAddress(sdk.WrapSDKContext(ctx), &removeMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Test removing addresses from non-existent zone
	invalidZoneMsg := types.MsgRemoveTaxExemptionAddress{
		Zone:      "nonexistent",
		Authority: authority,
		Addresses: []string{address3.String()},
	}

	resp, err = server.RemoveTaxExemptionAddress(sdk.WrapSDKContext(ctx), &invalidZoneMsg)
	require.Error(t, err)
	require.Nil(t, resp)

	// Test with invalid authority
	invalidAuthMsg := types.MsgRemoveTaxExemptionAddress{
		Zone:      "zone1",
		Authority: "invalid_authority",
		Addresses: []string{address3.String()},
	}

	resp, err = server.RemoveTaxExemptionAddress(sdk.WrapSDKContext(ctx), &invalidAuthMsg)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "invalid authority")

	// Test removing non-existent address
	nonExistentAddrMsg := types.MsgRemoveTaxExemptionAddress{
		Zone:      "zone1",
		Authority: authority,
		Addresses: []string{"terra1nonexistent"},
	}

	resp, err = server.RemoveTaxExemptionAddress(sdk.WrapSDKContext(ctx), &nonExistentAddrMsg)
	// If removing a non-existent address should error:
	require.Error(t, err)
	require.Nil(t, resp)
}
