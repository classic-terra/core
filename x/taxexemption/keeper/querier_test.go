package keeper_test

import (
	"testing"

	ultil "github.com/classic-terra/core/v3/x/taxexemption/keeper"
	"github.com/classic-terra/core/v3/x/taxexemption/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestQueryTaxable(t *testing.T) {
	input := ultil.CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := ultil.NewQuerier(input.TaxExemptionKeeper)

	pubKey := secp256k1.GenPrivKey().PubKey()
	pubKey2 := secp256k1.GenPrivKey().PubKey()
	pubKey3 := secp256k1.GenPrivKey().PubKey()
	pubKey4 := secp256k1.GenPrivKey().PubKey()
	pubKey5 := secp256k1.GenPrivKey().PubKey()
	pubKey6 := secp256k1.GenPrivKey().PubKey()
	pubKey7 := secp256k1.GenPrivKey().PubKey()
	address := sdk.AccAddress(pubKey.Address())
	address2 := sdk.AccAddress(pubKey2.Address())
	address3 := sdk.AccAddress(pubKey3.Address())
	address4 := sdk.AccAddress(pubKey4.Address())
	address5 := sdk.AccAddress(pubKey5.Address())
	address6 := sdk.AccAddress(pubKey6.Address())
	address7 := sdk.AccAddress(pubKey7.Address())

	// Add a zone
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone1", Outgoing: false, Incoming: false, CrossZone: false})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone2", Outgoing: false, Incoming: false, CrossZone: true})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone3", Outgoing: false, Incoming: true, CrossZone: false})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone4", Outgoing: false, Incoming: true, CrossZone: true})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone5", Outgoing: true, Incoming: false, CrossZone: false})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone6", Outgoing: true, Incoming: false, CrossZone: true})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone7", Outgoing: true, Incoming: true, CrossZone: false})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: "zone8", Outgoing: true, Incoming: true, CrossZone: true})

	zones := []string{
		"zone1",
		"zone2",
		"zone3",
		"zone4",
		"zone5",
		"zone6",
		"zone7",
		"zone8",
	}

	// Case 1: Empty request
	_, err := querier.Taxable(ctx, nil)
	require.Error(t, err)

	// Query to grpc
	// Case 2: Sender & recipient have no zone
	res, err := querier.Taxable(ctx, &types.QueryTaxableRequest{
		FromAddress: address.String(),
		ToAddress:   address2.String(),
	})
	require.NoError(t, err)
	require.Equal(t, true, res.Taxable)

	// Case 3: Sender & recipient have different zone
	// 3.1: Sender have CrossZone and Outgoing, recipient have any zone
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "zone6", address3.String()) // Sender
	for _, zone := range zones {
		input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone, address4.String()) // Recipient
		res, err = querier.Taxable(ctx, &types.QueryTaxableRequest{
			FromAddress: address3.String(),
			ToAddress:   address4.String(),
		})
		require.NoError(t, err)
		require.Equal(t, false, res.Taxable)

		input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone, address4.String())
	}

	// 3.2: Recipient have CrossZone and Incoming, sender have any zone
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, "zone4", address5.String()) // Recipient
	for _, zone := range zones {
		input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone, address6.String()) // Sender
		res, err = querier.Taxable(ctx, &types.QueryTaxableRequest{
			FromAddress: address6.String(),
			ToAddress:   address5.String(),
		})
		require.NoError(t, err)
		require.Equal(t, false, res.Taxable)

		input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone, address6.String())
	}

	// Case 4: Only sender has zone
	// 4.1: Sender doesn't have Outcoming, recipient doesn't matter
	zones1 := []string{
		"zone1",
		"zone2",
		"zone3",
		"zone4",
	}
	for _, zone := range zones1 {
		input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone, address7.String())
		res, err = querier.Taxable(ctx, &types.QueryTaxableRequest{
			FromAddress: address7.String(),
		})
		require.NoError(t, err)
		require.Equal(t, true, res.Taxable)

		input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone, address7.String())
	}
	// 4.2: Sender has Outcoming, recipient doesn't matter
	zones2 := []string{
		"zone5",
		"zone6",
		"zone7",
		"zone8",
	}
	for _, zone := range zones2 {
		input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone, address7.String())
		res, err = querier.Taxable(ctx, &types.QueryTaxableRequest{
			FromAddress: address7.String(),
		})
		require.NoError(t, err)
		require.Equal(t, false, res.Taxable)

		input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone, address7.String())
	}

	// Case 5: Only recipient has zone
	// 5.1: Recipient doesn't have Incoming, sender doesn't matter
	zones3 := []string{
		"zone1",
		"zone2",
		"zone5",
		"zone6",
	}
	for _, zone := range zones3 {
		input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone, address7.String())
		res, err = querier.Taxable(ctx, &types.QueryTaxableRequest{
			ToAddress: address7.String(),
		})
		require.NoError(t, err)
		require.Equal(t, true, res.Taxable)

		input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone, address7.String())
	}
	// 5.2: Recipient has Incoming, sender doesn't matter
	zones4 := []string{
		"zone3",
		"zone4",
		"zone7",
		"zone8",
	}
	for _, zone := range zones4 {
		input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zone, address7.String())
		res, err = querier.Taxable(ctx, &types.QueryTaxableRequest{
			ToAddress: address7.String(),
		})
		require.NoError(t, err)
		require.Equal(t, false, res.Taxable)

		input.TaxExemptionKeeper.RemoveTaxExemptionAddress(input.Ctx, zone, address7.String())
	}
}

func TestTaxExemptionZonesList(t *testing.T) {
	input := ultil.CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := ultil.NewQuerier(input.TaxExemptionKeeper)

	// Create zone tests
	zones := []types.Zone{
		{
			Name:      "zone1",
			Outgoing:  true,
			Incoming:  true,
			CrossZone: true,
		},
		{
			Name:      "zone2",
			Outgoing:  false,
			Incoming:  true,
			CrossZone: false,
		},
		{
			Name:      "zone3",
			Outgoing:  true,
			Incoming:  false,
			CrossZone: false,
		},
		{
			Name:      "zone4",
			Outgoing:  false,
			Incoming:  false,
			CrossZone: true,
		},
		{
			Name:      "zone5",
			Outgoing:  false,
			Incoming:  false,
			CrossZone: false,
		},
	}

	for _, zone := range zones {
		err := input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, zone)
		require.NoError(t, err)
	}

	// Case 1: Query without pagination
	res, err := querier.TaxExemptionZonesList(ctx, &types.QueryTaxExemptionZonesRequest{})
	require.NoError(t, err)
	require.Equal(t, 5, len(res.Zones))

	// Case 2: Query with pagination
	res, err = querier.TaxExemptionZonesList(ctx, &types.QueryTaxExemptionZonesRequest{
		Pagination: &query.PageRequest{
			Limit: 2,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(res.Zones))
}

func TestTaxExemptionAddressList(t *testing.T) {
	input := ultil.CreateTestInput(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := ultil.NewQuerier(input.TaxExemptionKeeper)

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

	zoneName1 := "zone1"
	zoneName2 := "zone2"

	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: zoneName1, Outgoing: true, Incoming: true, CrossZone: true})
	input.TaxExemptionKeeper.AddTaxExemptionZone(input.Ctx, types.Zone{Name: zoneName2, Outgoing: false, Incoming: false, CrossZone: false})

	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zoneName1, address.String())
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zoneName1, address2.String())
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zoneName1, address3.String())
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zoneName2, address4.String())
	input.TaxExemptionKeeper.AddTaxExemptionAddress(input.Ctx, zoneName2, address5.String())

	// Case 1: Query all addresses without zone
	res, err := querier.TaxExemptionAddressList(ctx, &types.QueryTaxExemptionAddressRequest{})
	require.NoError(t, err)
	require.Equal(t, 5, len(res.Addresses))

	// Case 2: Query addresses with zone
	res, err = querier.TaxExemptionAddressList(ctx, &types.QueryTaxExemptionAddressRequest{
		ZoneName: zoneName1,
	})
	require.NoError(t, err)
	require.Equal(t, 3, len(res.Addresses))

	// Case 3: Query addresses with pagination
	res, err = querier.TaxExemptionAddressList(ctx, &types.QueryTaxExemptionAddressRequest{
		Pagination: &query.PageRequest{
			Limit: 2,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(res.Addresses))

	// Case 4: Query addresses with zone and pagination
	res, err = querier.TaxExemptionAddressList(ctx, &types.QueryTaxExemptionAddressRequest{
		ZoneName: zoneName1,
		Pagination: &query.PageRequest{
			Limit: 1,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Addresses))

	// Case 5: Query addresses with non-existent zone
	res, err = querier.TaxExemptionAddressList(ctx, &types.QueryTaxExemptionAddressRequest{
		ZoneName: "zone3",
	})
	require.NoError(t, err)
	require.Equal(t, 0, len(res.Addresses))

	// Case 6: Query addresses with zone, pagination (limit=2), and offset=1. Should return 2 addresses starting from the second address.
	res, err = querier.TaxExemptionAddressList(ctx, &types.QueryTaxExemptionAddressRequest{
		ZoneName: zoneName1,
		Pagination: &query.PageRequest{
			Limit:  2,
			Offset: 1,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(res.Addresses))

	// Case 7: Query addresses with zone, pagination (limit=2), and offset=2. Should return 1 address starting from the third address.
	res, err = querier.TaxExemptionAddressList(ctx, &types.QueryTaxExemptionAddressRequest{
		ZoneName: zoneName1,
		Pagination: &query.PageRequest{
			Limit:  2,
			Offset: 2,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Addresses))

	// Case 8: Query addresses with zone, pagination (limit=2), and offset greater than total addresses. Should return 0 addresses.
	res, err = querier.TaxExemptionAddressList(ctx, &types.QueryTaxExemptionAddressRequest{
		ZoneName: zoneName1,
		Pagination: &query.PageRequest{
			Limit:  2,
			Offset: 3,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 0, len(res.Addresses))
}
