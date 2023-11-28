package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v2/x/taxexemption/types"
)

// querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over q
type querier struct {
	Keeper
}

// NewQuerier returns an implementation of the market QueryServer interface
// for the provided Keeper.
func NewQuerier(keeper Keeper) types.QueryServer {
	return &querier{Keeper: keeper}
}

var _ types.QueryServer = querier{}

// TaxExemptionZoneList queries tax exemption zone list of taxexemption module
func (q querier) TaxExemptionZonesList(c context.Context, req *types.QueryTaxExemptionZonesRequest) (*types.QueryTaxExemptionZonesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	zones, err := q.Keeper.ListTaxExemptionZones(ctx, req)
	if err != nil {
		return nil, err
	}
	return &types.QueryTaxExemptionZonesResponse{Zones: zones.Zones}, nil
}

// TaxExemptionAddressList queries tax exemption address list of taxexemption module
func (q querier) TaxExemptionAddressList(c context.Context, req *types.QueryTaxExemptionAddressRequest) (*types.QueryTaxExemptionAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	addresses, pageRes, err := q.Keeper.ListTaxExemptionAddresses(ctx, req)
	if err != nil {
		return nil, err
	}
	return &types.QueryTaxExemptionAddressResponse{Addresses: addresses, Pagination: pageRes}, nil
}
