package taxexemption

import (
	"fmt"
	"slices"
	"sort"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v3/x/taxexemption/keeper"
	"github.com/classic-terra/core/v3/x/taxexemption/types"
)

// DefaultGenesisState gets raw genesis raw message for testing
func DefaultGenesisState() *types.GenesisState {
	return &types.GenesisState{
		ZoneList:        []types.Zone{},
		AddressesByZone: []types.AddressesByZone{},
	}
}

// ValidateGenesis validates the provided oracle genesis state to ensure the
// expected invariants holds. (i.e. params in correct bounds, no duplicate validators)
func ValidateGenesis(data *types.GenesisState) error {
	if len(data.ZoneList) != len(data.AddressesByZone) {
		return types.ErrZoneLengthInvalid
	}

	zones := []string{}
	for _, zone := range data.ZoneList {
		zones = append(zones, zone.Name)
	}

	for _, addressesByZone := range data.AddressesByZone {
		if !slices.Contains(zones, addressesByZone.Zone) {
			return types.ErrZoneNotExist
		}

		for _, address := range addressesByZone.Addresses {
			if _, err := sdk.AccAddressFromBech32(address); err != nil {
				return err
			}
		}
	}
	return nil
}

// InitGenesis initializes default parameters
// and the keeper's address to pubkey map
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, data *types.GenesisState) {
	// Set the zone list
	for _, zone := range data.ZoneList {
		keeper.AddTaxExemptionZone(ctx, zone)
	}

	// Set the addresses by zone
	for _, addressesByZone := range data.AddressesByZone {
		for _, address := range addressesByZone.Addresses {
			keeper.AddTaxExemptionAddress(ctx, addressesByZone.Zone, address)
		}
	}
}

// ExportGenesis writes the current store values
// to a genesis file, which can be imported again
// with InitGenesis
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) (data *types.GenesisState) {
	zonePrefix := prefix.NewStore(ctx.KVStore(keeper.StoreKey()), types.TaxExemptionZonePrefix)
	iterator := zonePrefix.Iterator(nil, nil)
	defer iterator.Close()

	var zones []types.Zone
	zoneAddresses := make(map[string][]string)
	var addresesByZone []types.AddressesByZone

	for ; iterator.Valid(); iterator.Next() {
		var zone types.Zone
		keeper.Codec().MustUnmarshal(iterator.Value(), &zone)
		zones = append(zones, zone)
		zoneAddresses[zone.Name] = []string{}
	}

	addressesPrefix := prefix.NewStore(ctx.KVStore(keeper.StoreKey()), types.TaxExemptionListPrefix)
	iterator = addressesPrefix.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		zoneName := string(iterator.Value())
		if _, ok := zoneAddresses[zoneName]; ok {
			zoneAddresses[zoneName] = append(zoneAddresses[zoneName], string(iterator.Key()))
		}
	}

	var zoneNames []string
	for zoneName := range zoneAddresses {
		zoneNames = append(zoneNames, zoneName)
	}
	sort.Strings(zoneNames)

	for _, zoneName := range zoneNames {
		addresses := zoneAddresses[zoneName]
		addresesByZone = append(addresesByZone, types.AddressesByZone{
			Zone:      zoneName,
			Addresses: addresses,
		})
	}

	state := &types.GenesisState{
		ZoneList:        zones,
		AddressesByZone: addresesByZone,
	}
	err := ValidateGenesis(state)
	if err != nil {
		panic(fmt.Sprint("error validate genesis: ", err))
	}
	return state
}
