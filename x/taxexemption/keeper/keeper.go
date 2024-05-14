package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/classic-terra/core/v3/x/taxexemption/types"
)

// Keeper of the store
type Keeper struct {
	storeKey      storetypes.StoreKey
	cdc           codec.BinaryCodec
	paramSpace    paramstypes.Subspace
	accountKeeper accountkeeper.AccountKeeper
	authority     string
}

// NewKeeper creates a new taxexemption Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	paramSpace paramstypes.Subspace,
	accountKeeper accountkeeper.AccountKeeper,
	authority string,
) Keeper {
	// set KeyTable if it has not already been set
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		paramSpace:    paramSpace,
		accountKeeper: accountKeeper,
		authority:     authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetAuthority() sdk.AccAddress {
	return sdk.AccAddress(k.authority)
}

func (k Keeper) GetTaxExemptionZone(ctx sdk.Context, zoneName string) (types.Zone, error) {
	// Ensure the storeKey is properly set up in the Keeper
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionZonePrefix)

	// Convert the zone name to byte slice which will be used as the key
	key := []byte(zoneName)

	// Check if the zone exists
	if !store.Has(key) {
		return types.Zone{}, types.ErrNoSuchTaxExemptionZone.Wrapf("zone = %s", zoneName)
	}

	// Get the zone
	bz := store.Get(key)

	// Unmarshal the zone
	var zone types.Zone
	k.cdc.MustUnmarshal(bz, &zone)

	return zone, nil
}

// Tax exemption zone list
func (k Keeper) AddTaxExemptionZone(ctx sdk.Context, zone types.Zone) error {
	// Ensure the storeKey is properly set up in the Keeper
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionZonePrefix)

	// Convert the zone name to byte slice which will be used as the key
	key := []byte(zone.Name)

	// Marshal the zone struct to binary format
	marshaledZone := k.cdc.MustMarshal(&zone)

	// Store the marshaled zone under its name key
	store.Set(key, marshaledZone)

	return nil
}

func (k Keeper) ModifyTaxExemptionZone(ctx sdk.Context, zone types.Zone) error {
	// Ensure the storeKey is properly set up in the Keeper
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionZonePrefix)

	// Convert the zone name to byte slice which will be used as the key
	key := []byte(zone.Name)

	// Check if the zone exists
	if !store.Has(key) {
		return types.ErrNoSuchTaxExemptionZone.Wrapf("zone = %s", zone.Name)
	}

	// Marshal the zone struct to binary format
	marshaledZone := k.cdc.MustMarshal(&zone)

	// Store the marshaled zone under its name key
	store.Set(key, marshaledZone)

	return nil
}

func (k Keeper) RemoveTaxExemptionZone(ctx sdk.Context, zoneName string) error {
	// Ensure the storeKey is properly set up in the Keeper
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionZonePrefix)

	// Convert the zone name to byte slice which will be used as the key
	key := []byte(zoneName)

	// Check if the zone exists
	if !store.Has(key) {
		return types.ErrNoSuchTaxExemptionZone.Wrapf("zone = %s", zoneName)
	}

	// remove the zone from all the addresses
	// loop through all the addresses and remove the zone from their list
	sub := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionListPrefix)
	iter := sub.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		k.RemoveTaxExemptionAddress(ctx, zoneName, string(iter.Key()))
	}

	// Delete the zone
	store.Delete(key)

	return nil
}

// AddTaxExemptionAddress associates an address with a tax exemption zone
func (k Keeper) AddTaxExemptionAddress(ctx sdk.Context, zone string, address string) error {
	// Validate the address format
	if _, err := sdk.AccAddressFromBech32(address); err != nil {
		return err
	}

	zonestore := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionZonePrefix)
	zonekey := []byte(zone)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionListPrefix)
	addressKey := []byte(address)

	if !zonestore.Has(zonekey) {
		return types.ErrNoSuchTaxExemptionZone.Wrapf("zone = %s", zone)
	}

	// Check if the address is already associated with a zone
	bz := store.Get(addressKey)
	if bz != nil {
		existingZone := string(bz)

		// If the address is already associated with a different zone, raise an error
		if existingZone != zone {
			return fmt.Errorf("address %s is already associated with a different zone: %s", address, existingZone)
		}
		// If it's the same zone, no action needed
		return nil
	}

	// If the address is not associated with any zone, associate it with the new zone
	// Marshal using standard Go marshaling to bytes
	store.Set(addressKey, []byte(zone))

	return nil
}

// RemoveTaxExemptionAddress removes an address from the tax exemption list
func (k Keeper) RemoveTaxExemptionAddress(ctx sdk.Context, zone string, address string) error {
	// Validate the address format
	if _, err := sdk.AccAddressFromBech32(address); err != nil {
		return err
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionListPrefix)
	addressKey := []byte(address)

	// Check if the address is already associated with a zone
	bz := store.Get(addressKey)
	if bz == nil {
		return fmt.Errorf("address %s is not associated with any zone", address)
	}

	// If the address is associated with a different zone, raise an error
	if string(bz) != zone {
		return fmt.Errorf("address %s is associated with a different zone: %s", address, string(bz))
	}

	store.Delete(addressKey)

	return nil
}

// IsExemptedFromTax returns true if the transaction between sender and all recipients
// meets the tax exemption criteria based on their zones
func (k Keeper) IsExemptedFromTax(ctx sdk.Context, senderAddress string, recipientAddresses ...string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionListPrefix)

	// Cache for looked up zones to avoid redundant queries
	zoneCache := make(map[string]types.Zone)

	// Check and cache the sender's zone
	senderZone, senderHasZone := k.checkAndCacheZone(ctx, store, senderAddress, zoneCache)

	for _, address := range recipientAddresses {
		recipientZone, recipientHasZone := k.checkAndCacheZone(ctx, store, address, zoneCache)

		// both sender and recipient have no zone: no tax exemption
		if !senderHasZone && !recipientHasZone {
			return false
		}

		// Different zones: either sender must have CrossZone and outgoing, or recipient must have Incoming and CrossZone
		if senderHasZone && recipientHasZone && senderZone.Name != recipientZone.Name {
			if (!senderZone.Outgoing || !senderZone.CrossZone) && (!recipientZone.Incoming || !recipientZone.CrossZone) {
				return false
			}
		}

		// only sender has zone: sender must have outgoing
		if senderHasZone && !recipientHasZone && !senderZone.Outgoing {
			return false
		}

		// only recipient has zone: recipient must have incoming
		if !senderHasZone && recipientHasZone && !recipientZone.Incoming {
			return false
		}
	}

	// If all checks are passed, return true
	return true
}

// checkAndCacheZone checks and caches the zone of an address
func (k Keeper) checkAndCacheZone(ctx sdk.Context, store prefix.Store, address string, zoneCache map[string]types.Zone) (types.Zone, bool) {
	if bz := store.Get([]byte(address)); bz != nil {
		zoneName := string(bz)

		// Cache the zone
		if zone, ok := zoneCache[zoneName]; ok {
			return zone, true
		}

		zone, err := k.GetTaxExemptionZone(ctx, zoneName)
		if err != nil {
			return types.Zone{}, false
		}

		zoneCache[zoneName] = zone
		return zone, true
	}

	return types.Zone{}, false
}

func (k Keeper) ListTaxExemptionZones(c sdk.Context, req *types.QueryTaxExemptionZonesRequest) ([]types.Zone, *query.PageResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	sub := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionZonePrefix)

	var zones []types.Zone

	// Create a paginated iterator over the store
	pageRes, err := query.FilteredPaginate(sub, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		var zone types.Zone
		k.cdc.MustUnmarshal(value, &zone)
		zones = append(zones, zone)

		return true, nil
	})
	if err != nil {
		return nil, nil, err
	}

	return zones, pageRes, nil
}

func (k Keeper) ListTaxExemptionAddresses(c sdk.Context, req *types.QueryTaxExemptionAddressRequest) ([]string, *query.PageResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	sub := prefix.NewStore(ctx.KVStore(k.storeKey), types.TaxExemptionListPrefix)

	var addresses []string

	// Create an iterator over the store
	pageRes, err := query.FilteredPaginate(sub, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		if accumulate && (req.ZoneName == "" || string(value) == req.ZoneName) {
			addresses = append(addresses, string(key))
		}
		return true, nil
	})
	if err != nil {
		return nil, nil, err
	}

	return addresses, pageRes, nil
}
