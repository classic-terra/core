package v12

import (
	"context"

	"cosmossdk.io/store/prefix"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v3/app/keepers"
	"github.com/classic-terra/core/v3/app/upgrades"
	taxexemptiontypes "github.com/classic-terra/core/v3/x/taxexemption/types"
	treasurytypes "github.com/classic-terra/core/v3/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateV12UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	k *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(c)
		// migrate old treasurykeeper tax exemption to new tax exemption keeper
		// tax exemption keeper is now a module

		// get old tax exemption keeper
		sub := prefix.NewStore(sdkCtx.KVStore(k.TreasuryKeeper.GetStoreKey()), treasurytypes.BurnTaxExemptionListPrefix)

		intoZone := "Binance"

		// iterate through all tax exemptions
		iterator := sub.Iterator(nil, nil)
		addresses := []string{}
		defer iterator.Close()
		for ; iterator.Valid(); iterator.Next() {
			// get tax exemption address
			address := string(iterator.Key())
			addresses = append(addresses, address)
			// delete old key
		}

		versionMap, err := mm.RunMigrations(c, cfg, fromVM)
		if err != nil {
			return nil, err
		}

		// add tax exemption address to new tax exemption keeper
		err = k.TaxExemptionKeeper.AddTaxExemptionZone(sdkCtx, taxexemptiontypes.Zone{
			Name:      intoZone,
			Outgoing:  false,
			Incoming:  false,
			CrossZone: false,
		})
		if err != nil {
			return nil, err
		}

		for _, address := range addresses {
			err = k.TaxExemptionKeeper.AddTaxExemptionAddress(sdkCtx, intoZone, address)
			if err != nil {
				return nil, err
			}
		}

		return versionMap, nil
	}
}
