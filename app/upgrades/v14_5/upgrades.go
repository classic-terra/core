//nolint:revive
package v14_5

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v4/app/keepers"
	"github.com/classic-terra/core/v4/app/upgrades"
	taxexemptiontypes "github.com/classic-terra/core/v4/x/taxexemption/types"
	treasurytypes "github.com/classic-terra/core/v4/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// BinanceZone is the tax exemption zone created in the v12 upgrade from the
// former treasury burn tax exemption list.
const BinanceZone = "Binance"

// BinanceWalletAddresses are the new Binance wallets added to the Binance
// tax exemption zone in this upgrade.
var BinanceWalletAddresses = []string{
	"terra1lsmt5a8vqqus5fwslx8pyyemgjtg4y6uvj5ry9", // new Binance cold wallet
	"terra1tf9mt0tt84mn26zccazpeduf7axvw6a7yy9u7x", // new Binance hot wallet
}

// CreateV145UpgradeHandler wires module migrations for v14_5 (core v4.2.0),
// adds the new Binance wallets to the Binance tax exemption zone and deletes
// the legacy treasury burn tax exemption list, whose gov proposals are
// removed in this release.
//
// The binary also contains state-machine-breaking changes (taxexemption msgs
// become submittable via gov), so it must not be rolled out without this
// coordinated upgrade.
func CreateV145UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	k *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versionMap, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}

		sdkCtx := sdk.UnwrapSDKContext(ctx)

		if err := AddBinanceWallets(sdkCtx, k); err != nil {
			return nil, err
		}

		DeleteLegacyBurnTaxExemptionList(sdkCtx, k)

		return versionMap, nil
	}
}

// AddBinanceWallets adds BinanceWalletAddresses to the Binance zone. If the
// zone does not exist (e.g. on a testnet that never ran v12), it is created
// with the same flags v12 used on mainnet (outgoing/incoming/cross_zone all
// false, i.e. only intra-zone transfers are exempt). Adding an address that
// is already in the zone is a no-op; an address that belongs to a different
// zone aborts the upgrade.
func AddBinanceWallets(ctx sdk.Context, k *keepers.AppKeepers) error {
	te := k.TaxExemptionKeeper

	if _, err := te.GetTaxExemptionZone(ctx, BinanceZone); err != nil {
		ctx.Logger().Info("v14_5: tax exemption zone not found, creating it", "zone", BinanceZone)
		if err := te.AddTaxExemptionZone(ctx, taxexemptiontypes.Zone{
			Name:      BinanceZone,
			Outgoing:  false,
			Incoming:  false,
			CrossZone: false,
		}); err != nil {
			return fmt.Errorf("v14_5: create zone %s: %w", BinanceZone, err)
		}
	}

	for _, address := range BinanceWalletAddresses {
		if err := te.AddTaxExemptionAddress(ctx, BinanceZone, address); err != nil {
			return fmt.Errorf("v14_5: add %s to zone %s: %w", address, BinanceZone, err)
		}
		ctx.Logger().Info("v14_5: added tax exemption address", "zone", BinanceZone, "address", address)
	}

	return nil
}

// DeleteLegacyBurnTaxExemptionList removes every entry under the treasury
// BurnTaxExemptionListPrefix. The list was copied into the Binance zone in
// v12 but never deleted, and has not been consulted for taxation since.
// Keys are collected first so the store is not mutated while iterating.
func DeleteLegacyBurnTaxExemptionList(ctx sdk.Context, k *keepers.AppKeepers) {
	store := prefix.NewStore(ctx.KVStore(k.TreasuryKeeper.GetStoreKey()), treasurytypes.BurnTaxExemptionListPrefix)

	var keys [][]byte
	iter := store.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, iter.Key())
	}
	iter.Close()

	for _, key := range keys {
		store.Delete(key)
	}

	ctx.Logger().Info("v14_5: deleted legacy treasury burn tax exemption list", "entries", len(keys))
}
