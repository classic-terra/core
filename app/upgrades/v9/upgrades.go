package v9

import (
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	treasurytypes "github.com/classic-terra/core/v3/x/treasury/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func CreateV9UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	k *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(c sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// migrate old treasurykeeper tax exemption to new tax exemption keeper
		// tax exemption keeper is now a module

		// get old tax exemption keeper
		sub := prefix.NewStore(c.KVStore(k.TreasuryKeeper.GetStoreKey()), treasurytypes.BurnTaxExemptionListPrefix)

		intoZone := "Binance"

		// iterate through all tax exemptions
		iterator := sub.Iterator(nil, nil)
		defer iterator.Close()
		for ; iterator.Valid(); iterator.Next() {
			// get tax exemption address
			address := string(iterator.Key())

			// add tax exemption address to new tax exemption keeper
			k.TaxExemptionKeeper.AddTaxExemptionAddress(c, intoZone, address)
		}

		return mm.RunMigrations(c, cfg, fromVM)
	}
}
