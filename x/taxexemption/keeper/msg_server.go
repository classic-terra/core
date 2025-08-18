package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/classic-terra/core/v3/x/taxexemption/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the market MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) AddTaxExemptionZone(goCtx context.Context, msg *types.MsgAddTaxExemptionZone) (*types.MsgAddTaxExemptionZoneResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	err := k.Keeper.AddTaxExemptionZone(ctx, types.Zone{Name: msg.Zone, Outgoing: msg.Outgoing, Incoming: msg.Incoming, CrossZone: msg.CrossZone})
	if err != nil {
		return nil, err
	}

	for _, address := range msg.Addresses {
		err := k.Keeper.AddTaxExemptionAddress(ctx, msg.Zone, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgAddTaxExemptionZoneResponse{}, nil
}

func (k msgServer) RemoveTaxExemptionZone(goCtx context.Context, msg *types.MsgRemoveTaxExemptionZone) (*types.MsgRemoveTaxExemptionZoneResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	err := k.Keeper.RemoveTaxExemptionZone(ctx, msg.Zone)
	if err != nil {
		return nil, err
	}

	return &types.MsgRemoveTaxExemptionZoneResponse{}, nil
}

func (k msgServer) ModifyTaxExemptionZone(goCtx context.Context, msg *types.MsgModifyTaxExemptionZone) (*types.MsgModifyTaxExemptionZoneResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	err := k.Keeper.ModifyTaxExemptionZone(ctx, types.Zone{Name: msg.Zone, Outgoing: msg.Outgoing, Incoming: msg.Incoming, CrossZone: msg.CrossZone})
	if err != nil {
		return nil, err
	}

	return &types.MsgModifyTaxExemptionZoneResponse{}, nil
}

func (k msgServer) AddTaxExemptionAddress(goCtx context.Context, msg *types.MsgAddTaxExemptionAddress) (*types.MsgAddTaxExemptionAddressResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	for _, address := range msg.Addresses {
		err := k.Keeper.AddTaxExemptionAddress(ctx, msg.Zone, address)
		if err != nil {
			return nil, err
		}
	}
	return &types.MsgAddTaxExemptionAddressResponse{}, nil
}

func (k msgServer) RemoveTaxExemptionAddress(goCtx context.Context, msg *types.MsgRemoveTaxExemptionAddress) (*types.MsgRemoveTaxExemptionAddressResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	for _, address := range msg.Addresses {
		err := k.Keeper.RemoveTaxExemptionAddress(ctx, msg.Zone, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgRemoveTaxExemptionAddressResponse{}, nil
}
