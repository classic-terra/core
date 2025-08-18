package taxexemption_test

import (
	"slices"
	"testing"

	taxexemption "github.com/classic-terra/core/v3/x/taxexemption"
	util "github.com/classic-terra/core/v3/x/taxexemption/keeper"
	"github.com/classic-terra/core/v3/x/taxexemption/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultGenesisState(t *testing.T) {
	genesis := taxexemption.DefaultGenesisState()
	require.NotNil(t, genesis)
}

func TestInitAndExportGenesis_Empty(t *testing.T) {
	// Setup mock context & keeper (simple zero value for now)
	input := util.CreateTestInput(t)
	k := input.TaxExemptionKeeper

	// Initialize genesis with empty state
	genesis := taxexemption.DefaultGenesisState()
	require.NotNil(t, genesis)

	taxexemption.InitGenesis(input.Ctx, k, genesis)

	// Export genesis and check that it's not nil and matches default state
	exported := taxexemption.ExportGenesis(input.Ctx, k)
	require.NotNil(t, exported)
}

func TestInitAndExportGenesis_NonEmpty(t *testing.T) {
	// Setup mock context & keeper (simple zero value for now)
	input := util.CreateTestInput(t)
	k := input.TaxExemptionKeeper

	addresses := []string{
		util.Addrs[0].String(),
		util.Addrs[1].String(),
	}
	slices.Sort(addresses)
	// Initialize genesis with empty state
	genesis := taxexemption.DefaultGenesisState()
	genesis.ZoneList = []types.Zone{
		{
			Name:      "test-zone",
			Incoming:  true,
			Outgoing:  true,
			CrossZone: true,
		},
	}
	genesis.AddressesByZone = []types.AddressesByZone{
		{
			Zone:      "test-zone",
			Addresses: addresses,
		},
	}
	require.NotNil(t, genesis)

	taxexemption.InitGenesis(input.Ctx, k, genesis)

	// Export genesis and check that it's not nil and matches default state
	exportedGenesis := taxexemption.ExportGenesis(input.Ctx, k)
	require.NotNil(t, exportedGenesis)

	require.Equal(t, genesis, exportedGenesis)
}

func TestValidateGenesis_Success(t *testing.T) {
	// Initialize genesis with empty state
	genesis := taxexemption.DefaultGenesisState()
	genesis.ZoneList = []types.Zone{
		{
			Name:      "test-zone",
			Incoming:  true,
			Outgoing:  true,
			CrossZone: true,
		},
	}
	genesis.AddressesByZone = []types.AddressesByZone{
		{
			Zone: "test-zone",
			Addresses: []string{
				util.Addrs[0].String(),
				util.Addrs[1].String(),
			},
		},
	}
	require.NotNil(t, genesis)

	err := taxexemption.ValidateGenesis(genesis)
	require.NoError(t, err)
}

func TestValidateGenesis_Failure(t *testing.T) {
	// Case 1: Zone length mismatch
	genesis := taxexemption.DefaultGenesisState()
	genesis.ZoneList = []types.Zone{
		{
			Name:      "test-zone",
			Incoming:  true,
			Outgoing:  true,
			CrossZone: true,
		},
	}
	genesis.AddressesByZone = []types.AddressesByZone{}

	err := taxexemption.ValidateGenesis(genesis)
	require.ErrorContains(t, err, "length of zone list and addresses by zone must be equal")

	// Case 2: Invalid address
	genesis = taxexemption.DefaultGenesisState()
	genesis.ZoneList = []types.Zone{
		{
			Name:      "test-zone",
			Incoming:  true,
			Outgoing:  true,
			CrossZone: true,
		},
	}
	genesis.AddressesByZone = []types.AddressesByZone{
		{
			Zone: "test-zone",
			Addresses: []string{
				"invalid-address",
			},
		},
	}

	err = taxexemption.ValidateGenesis(genesis)
	require.ErrorContains(t, err, "decoding bech32 failed")

	// Case 3: Zone not exist
	genesis = taxexemption.DefaultGenesisState()
	genesis.ZoneList = []types.Zone{
		{
			Name:      "test-zone",
			Incoming:  true,
			Outgoing:  true,
			CrossZone: true,
		},
	}
	genesis.AddressesByZone = []types.AddressesByZone{
		{
			Zone: "non-existent-zone",
			Addresses: []string{
				util.Addrs[0].String(),
			},
		},
	}

	err = taxexemption.ValidateGenesis(genesis)
	require.ErrorContains(t, err, "zone not exist")
}
