package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/classic-terra/core/v3/x/taxexemption/types"
)

// querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over q
type querier struct {
	Keeper
}

// NewQuerier returns an implementation of the QueryServer interface
// for the provided Keeper.
func NewQuerier(keeper Keeper) types.QueryServer {
	return &querier{Keeper: keeper}
}

var _ types.QueryServer = querier{}

// Taxable queries if a tx from one address to another is taxable
func (q querier) Taxable(c context.Context, req *types.QueryTaxableRequest) (*types.QueryTaxableResponse, error) {
	if req == nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "Request must not nil")
	}

	ctx := sdk.UnwrapSDKContext(c)
	taxable := !q.Keeper.IsExemptedFromTax(ctx, req.FromAddress, req.ToAddress)
	return &types.QueryTaxableResponse{Taxable: taxable}, nil
}

// TaxExemptionZoneList queries tax exemption zone list of taxexemption module
func (q querier) TaxExemptionZonesList(c context.Context, req *types.QueryTaxExemptionZonesRequest) (*types.QueryTaxExemptionZonesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	zones, pageRes, err := q.Keeper.ListTaxExemptionZones(ctx, req)
	if err != nil {
		return nil, err
	}

	zonePointers := make([]*types.Zone, len(zones))
	for i, zone := range zones {
		zoneCopy := zone // Make a copy to avoid referencing the loop variable
		zonePointers[i] = &zoneCopy
	}

	return &types.QueryTaxExemptionZonesResponse{Zones: zonePointers, Pagination: pageRes}, nil
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
