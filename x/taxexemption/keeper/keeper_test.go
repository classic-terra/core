package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"

	"github.com/classic-terra/core/v3/x/taxexemption/types"
)

func TestTaxExemptionList(t *testing.T) {
	input := CreateTestInput(t)

	require.False(t, input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, "", ""))
	require.Error(t, input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "", ""))
	require.Error(t, input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, "", ""))

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

	// add an address to an invalid zone
	require.Error(t, input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "zone4", address.String()))

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

// TestAddTaxExemptionZone tests the AddTaxExemptionZone function
func TestAddTaxExemptionZone(t *testing.T) {
	input := CreateTestInput(t)

	// Define a test zone
	testZone := types.Zone{
		Name:      "test_zone",
		Outgoing:  true,
		Incoming:  true,
		CrossZone: false,
	}

	// Add the zone and verify no error
	err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, testZone)
	require.NoError(t, err, "Adding a zone should not error")

	// Retrieve the zone and verify it matches
	retrievedZone, err := input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, testZone.Name)
	require.NoError(t, err, "Getting the zone should not error")
	require.Equal(t, testZone.Name, retrievedZone.Name, "Zone name should match")
	require.Equal(t, testZone.Outgoing, retrievedZone.Outgoing, "Outgoing flag should match")
	require.Equal(t, testZone.Incoming, retrievedZone.Incoming, "Incoming flag should match")
	require.Equal(t, testZone.CrossZone, retrievedZone.CrossZone, "CrossZone flag should match")

	// Test adding another zone
	anotherZone := types.Zone{
		Name:      "another_zone",
		Outgoing:  false,
		Incoming:  true,
		CrossZone: true,
	}

	err = input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, anotherZone)
	require.NoError(t, err, "Adding another zone should not error")

	// Retrieve the second zone and verify
	retrievedZone, err = input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, anotherZone.Name)
	require.NoError(t, err, "Getting the second zone should not error")
	require.Equal(t, anotherZone.Name, retrievedZone.Name, "Zone name should match")
	require.Equal(t, anotherZone.Outgoing, retrievedZone.Outgoing, "Outgoing flag should match")
	require.Equal(t, anotherZone.Incoming, retrievedZone.Incoming, "Incoming flag should match")
	require.Equal(t, anotherZone.CrossZone, retrievedZone.CrossZone, "CrossZone flag should match")

	// Test overwriting an existing zone (should not error)
	modifiedZone := types.Zone{
		Name:      "test_zone", // Same name as first zone
		Outgoing:  false,
		Incoming:  false,
		CrossZone: true,
	}

	err = input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, modifiedZone)
	require.NoError(t, err, "Overwriting an existing zone should not error")

	// Verify the zone was overwritten
	retrievedZone, err = input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, modifiedZone.Name)
	require.NoError(t, err, "Getting the overwritten zone should not error")
	require.Equal(t, modifiedZone.Outgoing, retrievedZone.Outgoing, "Outgoing flag should be updated")
	require.Equal(t, modifiedZone.Incoming, retrievedZone.Incoming, "Incoming flag should be updated")
	require.Equal(t, modifiedZone.CrossZone, retrievedZone.CrossZone, "CrossZone flag should be updated")
}

// TestRemoveTaxExemptionZone tests the RemoveTaxExemptionZone function
func TestRemoveTaxExemptionZone(t *testing.T) {
	input := CreateTestInput(t)

	// Create test keys/addresses
	pubKey := secp256k1.GenPrivKey().PubKey()
	address := sdk.AccAddress(pubKey.Address())

	// Define a test zone
	testZone := types.Zone{
		Name:      "test_zone",
		Outgoing:  true,
		Incoming:  true,
		CrossZone: false,
	}

	// Add the zone
	err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, testZone)
	require.NoError(t, err, "Adding a zone should not error")

	// Add an address to the zone
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, testZone.Name, address.String())
	require.NoError(t, err, "Adding an address to the zone should not error")

	// Verify the address is in the zone
	isExempt := input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address.String(), address.String())
	require.True(t, isExempt, "Address should be in the zone (exempt from tax to itself)")

	// Remove the zone
	err = input.TaxExemptionKeeper.RemoveTaxExemptionZone(input.Ctx, testZone.Name)
	require.NoError(t, err, "Removing an existing zone should not error")

	// Verify the zone is gone
	_, err = input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, testZone.Name)
	require.Error(t, err, "Getting a removed zone should error")

	// Verify the address is no longer in any zone
	isExempt = input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, address.String(), address.String())
	require.False(t, isExempt, "Address should no longer be in any zone")

	// Try to remove a non-existent zone
	err = input.TaxExemptionKeeper.RemoveTaxExemptionZone(input.Ctx, "nonexistent_zone")
	require.Error(t, err, "Removing a non-existent zone should error")
	require.Contains(t, err.Error(), "no such zone in exemption list", "Error should indicate zone doesn't exist")

	// Add multiple zones
	zone1 := types.Zone{
		Name:      "zone1",
		Outgoing:  true,
		Incoming:  false,
		CrossZone: false,
	}

	zone2 := types.Zone{
		Name:      "zone2",
		Outgoing:  false,
		Incoming:  true,
		CrossZone: false,
	}

	err = input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, zone1)
	require.NoError(t, err, "Adding zone1 should not error")

	err = input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, zone2)
	require.NoError(t, err, "Adding zone2 should not error")

	// Remove one zone
	err = input.TaxExemptionKeeper.RemoveTaxExemptionZone(input.Ctx, zone1.Name)
	require.NoError(t, err, "Removing zone1 should not error")

	// Verify zone1 is gone but zone2 remains
	_, err = input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, zone1.Name)
	require.Error(t, err, "Getting removed zone1 should error")

	retrievedZone, err := input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, zone2.Name)
	require.NoError(t, err, "Getting zone2 should not error")
	require.Equal(t, zone2.Name, retrievedZone.Name, "Zone2 should still exist")
}

// TestModifyTaxExemptionZone tests the ModifyTaxExemptionZone function
func TestModifyTaxExemptionZone(t *testing.T) {
	input := CreateTestInput(t)

	// Define a test zone
	originalZone := types.Zone{
		Name:      "test_zone",
		Outgoing:  true,
		Incoming:  false,
		CrossZone: false,
	}

	// Add the zone
	err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, originalZone)
	require.NoError(t, err, "Adding a zone should not error")

	// Verify the zone exists with original values
	retrievedZone, err := input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, originalZone.Name)
	require.NoError(t, err, "Getting the zone should not error")
	require.Equal(t, originalZone.Outgoing, retrievedZone.Outgoing, "Outgoing flag should match")
	require.Equal(t, originalZone.Incoming, retrievedZone.Incoming, "Incoming flag should match")
	require.Equal(t, originalZone.CrossZone, retrievedZone.CrossZone, "CrossZone flag should match")

	// Modify the zone
	modifiedZone := types.Zone{
		Name:      "test_zone", // Same name as original
		Outgoing:  false,       // Changed
		Incoming:  true,        // Changed
		CrossZone: true,        // Changed
	}

	err = input.TaxExemptionKeeper.ModifyTaxExemptionZone(input.Ctx, modifiedZone)
	require.NoError(t, err, "Modifying an existing zone should not error")

	// Verify the zone was modified
	retrievedZone, err = input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, modifiedZone.Name)
	require.NoError(t, err, "Getting the modified zone should not error")
	require.Equal(t, modifiedZone.Outgoing, retrievedZone.Outgoing, "Outgoing flag should be updated")
	require.Equal(t, modifiedZone.Incoming, retrievedZone.Incoming, "Incoming flag should be updated")
	require.Equal(t, modifiedZone.CrossZone, retrievedZone.CrossZone, "CrossZone flag should be updated")

	// Try to modify a non-existent zone
	nonExistentZone := types.Zone{
		Name:      "nonexistent_zone",
		Outgoing:  true,
		Incoming:  true,
		CrossZone: true,
	}

	err = input.TaxExemptionKeeper.ModifyTaxExemptionZone(input.Ctx, nonExistentZone)
	require.Error(t, err, "Modifying a non-existent zone should error")
	require.Contains(t, err.Error(), "no such zone in exemption list", "Error should indicate zone doesn't exist")
}

// TestAddTaxExemptionAddress tests the AddTaxExemptionAddress function
func TestAddTaxExemptionAddress(t *testing.T) {
	input := CreateTestInput(t)

	// Create test keys/addresses
	pubKey1 := secp256k1.GenPrivKey().PubKey()
	pubKey2 := secp256k1.GenPrivKey().PubKey()
	pubKey3 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pubKey1.Address())
	addr2 := sdk.AccAddress(pubKey2.Address())
	addr3 := sdk.AccAddress(pubKey3.Address())

	// Define test zones
	zone1 := types.Zone{
		Name:      "zone1",
		Outgoing:  true,
		Incoming:  true,
		CrossZone: false,
	}

	zone2 := types.Zone{
		Name:      "zone2",
		Outgoing:  false,
		Incoming:  true,
		CrossZone: true,
	}

	// Try to add address to non-existent zone
	err := input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "nonexistent_zone", addr1.String())
	require.Error(t, err, "Adding address to non-existent zone should error")
	require.Contains(t, err.Error(), "no such zone in exemption list", "Error should indicate zone doesn't exist")

	// Add the zones
	err = input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, zone1)
	require.NoError(t, err, "Adding zone1 should not error")

	err = input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, zone2)
	require.NoError(t, err, "Adding zone2 should not error")

	// Test adding address with invalid format
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone1.Name, "invalid-address")
	require.Error(t, err, "Adding invalid address should error")

	// Add address to zone1
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone1.Name, addr1.String())
	require.NoError(t, err, "Adding address to zone1 should not error")

	// Verify address is in zone1
	isExempt := input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, addr1.String(), addr1.String())
	require.True(t, isExempt, "Address1 should be exempt from tax to itself")

	// Add multiple addresses to zone2
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone2.Name, addr2.String())
	require.NoError(t, err, "Adding address2 to zone2 should not error")

	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone2.Name, addr3.String())
	require.NoError(t, err, "Adding address3 to zone2 should not error")

	// Verify addresses are in zone2
	isExempt = input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, addr2.String(), addr2.String())
	require.True(t, isExempt, "Address2 should be exempt from tax to itself")

	isExempt = input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, addr3.String(), addr3.String())
	require.True(t, isExempt, "Address3 should be exempt from tax to itself")

	// Test tax exemption between addresses in the same zone
	isExempt = input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, addr2.String(), addr3.String())
	require.True(t, isExempt, "Addresses in the same zone should be exempt from tax to each other")

	// Test tax exemption between addresses in different zones
	// zone1: outgoing=true, incoming=true, crossZone=false
	// zone2: outgoing=false, incoming=true, crossZone=true
	// addr1 -> addr2: Exempt (zone2 has incoming and crossZone)
	isExempt = input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, addr1.String(), addr2.String())
	require.True(t, isExempt, "Address1 -> Address2 should be exempt (zone2 has incoming and crossZone)")

	// addr2 -> addr1: Not exempt (zone2 has crossZone but not outgoing)
	isExempt = input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, addr2.String(), addr1.String())
	require.False(t, isExempt, "Address2 -> Address1 should not be exempt (zone2 has crossZone but not outgoing)")

	// Add same address again (should not error)
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone1.Name, addr1.String())
	require.NoError(t, err, "Adding the same address again should not error")

	// Try to add address to a different zone
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone2.Name, addr1.String())
	require.Error(t, err, "Adding address already in zone1 to zone2 should error")
	require.Contains(t, err.Error(), "already associated with a different zone", "Error should indicate address is already in a zone")
}

// TestRemoveTaxExemptionAddress tests the RemoveTaxExemptionAddress function
func TestRemoveTaxExemptionAddress(t *testing.T) {
	input := CreateTestInput(t)

	// Create test keys/addresses
	pubKey1 := secp256k1.GenPrivKey().PubKey()
	pubKey2 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pubKey1.Address())
	addr2 := sdk.AccAddress(pubKey2.Address())

	// Define test zones
	zone1 := types.Zone{
		Name:      "zone1",
		Outgoing:  true,
		Incoming:  false,
		CrossZone: false,
	}

	zone2 := types.Zone{
		Name:      "zone2",
		Outgoing:  false,
		Incoming:  true,
		CrossZone: false,
	}

	// Add the zones
	err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, zone1)
	require.NoError(t, err, "Adding zone1 should not error")

	err = input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, zone2)
	require.NoError(t, err, "Adding zone2 should not error")

	// Add addresses to the zones
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone1.Name, addr1.String())
	require.NoError(t, err, "Adding address to zone1 should not error")

	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone2.Name, addr2.String())
	require.NoError(t, err, "Adding address to zone2 should not error")

	// Verify addresses are in their zones
	isExempt := input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, addr1.String(), addr1.String())
	require.True(t, isExempt, "Address1 should be exempt from tax to itself")

	// Test removing address from zone
	err = input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone1.Name, addr1.String())
	require.NoError(t, err, "Removing address from zone should not error")

	// Verify address was removed
	isExempt = input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, addr1.String(), addr1.String())
	require.False(t, isExempt, "Address1 should no longer be exempt after removal")

	// Second address should still be in its zone
	isExempt = input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, addr2.String(), addr2.String())
	require.True(t, isExempt, "Address2 should still be exempt")

	// Test removing an address from the wrong zone
	err = input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone1.Name, addr2.String())
	require.Error(t, err, "Removing address from wrong zone should error")

	// Test removing an address that doesn't exist in any zone
	err = input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone1.Name, addr1.String())
	require.Error(t, err, "Removing an address not in any zone should error")

	// Test removing address from non-existent zone
	err = input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, "nonexistent_zone", addr2.String())
	require.Error(t, err, "Removing address from non-existent zone should error")

	// Test removing invalid address format
	err = input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone2.Name, "invalid-address")
	require.Error(t, err, "Removing invalid address should error")
}

// TestGetTaxExemptionZone tests the GetTaxExemptionZone function
func TestGetTaxExemptionZone(t *testing.T) {
	input := CreateTestInput(t)

	// Test getting non-existent zone
	_, err := input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, "nonexistent")
	require.Error(t, err, "Getting non-existent zone should error")
	require.Contains(t, err.Error(), "no such zone in exemption list")

	// Add a test zone
	testZone := types.Zone{
		Name:      "test_zone",
		Outgoing:  true,
		Incoming:  false,
		CrossZone: true,
	}
	err = input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, testZone)
	require.NoError(t, err)

	// Test getting existing zone
	zone, err := input.TaxExemptionKeeper.GetTaxExemptionZone(input.Ctx, testZone.Name)
	require.NoError(t, err)
	require.Equal(t, testZone.Name, zone.Name)
	require.Equal(t, testZone.Outgoing, zone.Outgoing)
	require.Equal(t, testZone.Incoming, zone.Incoming)
	require.Equal(t, testZone.CrossZone, zone.CrossZone)
}

// TestCheckAndCacheZone tests the checkAndCacheZone function
func TestCheckAndCacheZone(t *testing.T) {
	input := CreateTestInput(t)
	store := prefix.NewStore(input.Ctx.KVStore(input.TaxExemptionKeeper.storeKey), types.TaxExemptionListPrefix)
	zoneCache := make(map[string]types.Zone)

	// Create test address and zone
	pubKey := secp256k1.GenPrivKey().PubKey()
	address := sdk.AccAddress(pubKey.Address())
	testZone := types.Zone{
		Name:      "test_zone",
		Outgoing:  true,
		Incoming:  false,
		CrossZone: true,
	}

	// Test with non-existent address
	zone, exists := input.TaxExemptionKeeper.checkAndCacheZone(input.Ctx, store, address.String(), zoneCache)
	require.False(t, exists)
	require.Empty(t, zone)

	// Add zone and associate address with it
	err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, testZone)
	require.NoError(t, err)
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, testZone.Name, address.String())
	require.NoError(t, err)

	// Test with existing address
	zone, exists = input.TaxExemptionKeeper.checkAndCacheZone(input.Ctx, store, address.String(), zoneCache)
	require.True(t, exists)
	require.Equal(t, testZone.Name, zone.Name)

	// Test cache functionality
	cachedZone, exists := zoneCache[testZone.Name]
	require.True(t, exists)
	require.Equal(t, testZone.Name, cachedZone.Name)
}

// TestListTaxExemptionAddresses tests the ListTaxExemptionAddresses function
func TestListTaxExemptionAddresses(t *testing.T) {
	input := CreateTestInput(t)

	// Create test addresses
	addresses := make([]sdk.AccAddress, 3)
	for i := 0; i < 3; i++ {
		pubKey := secp256k1.GenPrivKey().PubKey()
		addresses[i] = sdk.AccAddress(pubKey.Address())
	}

	// Create test zones
	zones := []types.Zone{
		{
			Name:      "zone1",
			Outgoing:  true,
			Incoming:  false,
			CrossZone: false,
		},
		{
			Name:      "zone2",
			Outgoing:  false,
			Incoming:  true,
			CrossZone: true,
		},
	}

	// Add zones
	for _, zone := range zones {
		err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, zone)
		require.NoError(t, err)
	}

	// Add addresses to zones
	err := input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zones[0].Name, addresses[0].String())
	require.NoError(t, err)
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zones[0].Name, addresses[1].String())
	require.NoError(t, err)
	err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zones[1].Name, addresses[2].String())
	require.NoError(t, err)

	// Test listing all addresses
	req := &types.QueryTaxExemptionAddressRequest{
		ZoneName:   "",
		Pagination: nil,
	}
	listedAddresses, pageRes, err := input.TaxExemptionKeeper.ListTaxExemptionAddresses(input.Ctx, req)
	require.NoError(t, err)
	require.Len(t, listedAddresses, len(addresses))
	require.NotNil(t, pageRes)

	// Test listing addresses for specific zone
	req = &types.QueryTaxExemptionAddressRequest{
		ZoneName: zones[0].Name,
		Pagination: &sdkquery.PageRequest{
			Limit: 2,
		},
	}
	listedAddresses, pageRes, err = input.TaxExemptionKeeper.ListTaxExemptionAddresses(input.Ctx, req)
	require.NoError(t, err)
	require.Len(t, listedAddresses, 2)
	require.NotNil(t, pageRes)
}

// TestListTaxExemptionZones tests the ListTaxExemptionZones function
func TestListTaxExemptionZones(t *testing.T) {
	input := CreateTestInput(t)

	// Add multiple test zones
	testZones := []types.Zone{
		{
			Name:      "zone1",
			Outgoing:  true,
			Incoming:  false,
			CrossZone: false,
		},
		{
			Name:      "zone2",
			Outgoing:  false,
			Incoming:  true,
			CrossZone: false,
		},
		{
			Name:      "zone3",
			Outgoing:  false,
			Incoming:  false,
			CrossZone: true,
		},
		{
			Name:      "zone4",
			Outgoing:  true,
			Incoming:  true,
			CrossZone: false,
		},
		{
			Name:      "zone5",
			Outgoing:  true,
			Incoming:  true,
			CrossZone: true,
		},
	}

	// Add all test zones
	for _, zone := range testZones {
		err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, zone)
		require.NoError(t, err, "Adding zone should not error")
	}

	// Test case 1: List all zones without pagination
	t.Run("List all zones without pagination", func(t *testing.T) {
		req := &types.QueryTaxExemptionZonesRequest{
			Pagination: nil,
		}

		zones, _, err := input.TaxExemptionKeeper.ListTaxExemptionZones(input.Ctx, req)
		require.NoError(t, err, "Listing zones should not error")
		require.Len(t, zones, len(testZones), "Should return all zones")

		// Verify zone contents
		for i, zone := range zones {
			require.Equal(t, testZones[i].Name, zone.Name, "Zone name should match")
			require.Equal(t, testZones[i].Outgoing, zone.Outgoing, "Outgoing flag should match")
			require.Equal(t, testZones[i].Incoming, zone.Incoming, "Incoming flag should match")
			require.Equal(t, testZones[i].CrossZone, zone.CrossZone, "CrossZone flag should match")
		}
	})

	// Test case 2: List zones with pagination (2 items per page)
	t.Run("List zones with pagination", func(t *testing.T) {
		pageReq := &sdkquery.PageRequest{
			Limit:  2,
			Offset: 0,
		}
		req := &types.QueryTaxExemptionZonesRequest{
			Pagination: pageReq,
		}

		zones, pageRes, err := input.TaxExemptionKeeper.ListTaxExemptionZones(input.Ctx, req)
		require.NoError(t, err, "Listing zones with pagination should not error")
		require.Len(t, zones, 2, "Should return 2 zones")
		require.NotNil(t, pageRes, "Page response should not be nil")
		require.NotNil(t, pageRes.NextKey, "Next key should be present")
	})

	// Test case 3: List zones with offset
	t.Run("List zones with offset", func(t *testing.T) {
		pageReq := &sdkquery.PageRequest{
			Limit:  2,
			Offset: 3, // Skip first 3 zones
		}
		req := &types.QueryTaxExemptionZonesRequest{
			Pagination: pageReq,
		}

		zones, pageRes, err := input.TaxExemptionKeeper.ListTaxExemptionZones(input.Ctx, req)
		require.NoError(t, err, "Listing zones with offset should not error")
		require.Len(t, zones, 2, "Should return 2 zones")
		require.NotNil(t, pageRes, "Page response should not be nil")
	})

	// Test case 4: List zones with limit larger than remaining items
	t.Run("List zones with large limit", func(t *testing.T) {
		pageReq := &sdkquery.PageRequest{
			Limit:  10, // Larger than total number of zones
			Offset: 0,
		}
		req := &types.QueryTaxExemptionZonesRequest{
			Pagination: pageReq,
		}

		zones, pageRes, err := input.TaxExemptionKeeper.ListTaxExemptionZones(input.Ctx, req)
		require.NoError(t, err, "Listing zones with large limit should not error")
		require.Len(t, zones, len(testZones), "Should return all zones")
		require.NotNil(t, pageRes, "Page response should not be nil")
		require.Nil(t, pageRes.NextKey, "Next key should be nil as all items are returned")
	})

	// Test case 5: List zones with zero limit
	t.Run("List zones with zero limit", func(t *testing.T) {
		pageReq := &sdkquery.PageRequest{
			Limit:  0, // Should return all zones
			Offset: 0,
		}
		req := &types.QueryTaxExemptionZonesRequest{
			Pagination: pageReq,
		}

		zones, pageRes, err := input.TaxExemptionKeeper.ListTaxExemptionZones(input.Ctx, req)
		require.NoError(t, err, "Listing zones with zero limit should not error")
		require.Len(t, zones, len(testZones), "Should return all zones")
		require.NotNil(t, pageRes, "Page response should not be nil")
	})
}

// TestIsExemptedFromTaxAllCases tests all possible combinations of Outgoing, CrossZone, and Incoming flags
func TestIsExemptedFromTaxAllCases(t *testing.T) {
	input := CreateTestInput(t)

	// Create test addresses
	pubKey1 := secp256k1.GenPrivKey().PubKey()
	pubKey2 := secp256k1.GenPrivKey().PubKey()
	pubKey3 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pubKey1.Address())
	addr2 := sdk.AccAddress(pubKey2.Address())
	addr3 := sdk.AccAddress(pubKey3.Address()) // Address not in any zone

	// Test cases for zone configurations
	testCases := []struct {
		name         string
		zone1        types.Zone
		zone2        types.Zone
		sender       string
		recipient    string
		expectExempt bool
		description  string
	}{
		{
			name:         "Same Zone - All Flags False",
			zone1:        types.Zone{Name: "zone1", Outgoing: false, Incoming: false, CrossZone: false},
			zone2:        types.Zone{Name: "zone1", Outgoing: false, Incoming: false, CrossZone: false},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: true, // Same zone always exempt
			description:  "Transactions within the same zone are always exempt",
		},
		{
			name:         "Different Zones - All Flags False",
			zone1:        types.Zone{Name: "zone1", Outgoing: false, Incoming: false, CrossZone: false},
			zone2:        types.Zone{Name: "zone2", Outgoing: false, Incoming: false, CrossZone: false},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: false,
			description:  "No cross-zone permissions",
		},
		{
			name:         "Sender Zone Only - Outgoing True, CrossZone True",
			zone1:        types.Zone{Name: "zone1", Outgoing: true, Incoming: false, CrossZone: true},
			zone2:        types.Zone{Name: "zone2", Outgoing: false, Incoming: false, CrossZone: false},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: true,
			description:  "Sender zone allows outgoing transactions",
		},
		{
			name:         "Sender Zone Only - Outgoing True, CrossZone True, Recipient Zone Don't Have Zone",
			zone1:        types.Zone{Name: "zone1", Outgoing: true, Incoming: false, CrossZone: true},
			zone2:        types.Zone{},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: true,
			description:  "Sender zone allows outgoing transactions to recipient don't have zone",
		},
		{
			name:         "Recipient Zone Only - Incoming True, Sender Zone Outgoing False",
			zone1:        types.Zone{Name: "zone1", Outgoing: false, Incoming: false, CrossZone: true},
			zone2:        types.Zone{Name: "zone2", Outgoing: false, Incoming: true, CrossZone: true},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: true,
			description:  "Recipient zone allows incoming transactions",
		},
		{
			name:         "Recipient Zone Only - Incoming True, Sender Zone Outgoing True, CrossZone False",
			zone1:        types.Zone{Name: "zone1", Outgoing: true, Incoming: false, CrossZone: false},
			zone2:        types.Zone{Name: "zone2", Outgoing: false, Incoming: true, CrossZone: true},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: true,
			description:  "Recipient zone allows incoming transactions",
		},
		{
			name:         "Recipient Zone Only - Incoming True CrossZone True, Sender Zone Don't Have Zone",
			zone1:        types.Zone{},
			zone2:        types.Zone{Name: "zone2", Outgoing: false, Incoming: true, CrossZone: true},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: true,
			description:  "Recipient zone allows incoming transactions",
		},
		{
			name:         "Cross-Zone - Sender Outgoing & CrossZone True",
			zone1:        types.Zone{Name: "zone1", Outgoing: true, Incoming: false, CrossZone: true},
			zone2:        types.Zone{Name: "zone2", Outgoing: false, Incoming: false, CrossZone: false},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: true,
			description:  "Sender zone allows outgoing and cross-zone transactions",
		},
		{
			name:         "Cross-Zone - Recipient Incoming & CrossZone True",
			zone1:        types.Zone{Name: "zone1", Outgoing: false, Incoming: false, CrossZone: false},
			zone2:        types.Zone{Name: "zone2", Outgoing: false, Incoming: true, CrossZone: true},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: true,
			description:  "Recipient zone allows incoming and cross-zone transactions",
		},
		{
			name:         "Cross-Zone - Both Zones CrossZone True but No Incoming/Outgoing",
			zone1:        types.Zone{Name: "zone1", Outgoing: false, Incoming: false, CrossZone: true},
			zone2:        types.Zone{Name: "zone2", Outgoing: false, Incoming: false, CrossZone: true},
			sender:       addr1.String(),
			recipient:    addr2.String(),
			expectExempt: false,
			description:  "Cross-zone enabled but no incoming/outgoing permissions",
		},
		{
			name:         "No Zones - Both Addresses",
			zone1:        types.Zone{},
			zone2:        types.Zone{},
			sender:       addr3.String(),
			recipient:    addr3.String(),
			expectExempt: false,
			description:  "Neither address is in a zone",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset keeper state for each test case
			input = CreateTestInput(t)

			if tc.zone1.Name != "" {
				err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, tc.zone1)
				require.NoError(t, err)
				if tc.sender != addr3.String() {
					err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, tc.zone1.Name, addr1.String())
					require.NoError(t, err)
				}
			}

			if tc.zone2.Name != "" {
				err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, tc.zone2)
				require.NoError(t, err)
				if tc.recipient != addr3.String() {
					err = input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, tc.zone2.Name, addr2.String())
					require.NoError(t, err)
				}
			}

			// Test tax exemption
			isExempt := input.TaxExemptionKeeper.IsExemptedFromTax(input.Ctx, tc.sender, tc.recipient)
			require.Equal(t, tc.expectExempt, isExempt, tc.description)
		})
	}
}
