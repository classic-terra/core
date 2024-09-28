package taxexemption

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/classic-terra/core/v3/x/taxexemption/keeper"
	"github.com/classic-terra/core/v3/x/taxexemption/types"
)

// NewHandler creates a new handler for all market type messages.
func NewHandler(k keeper.Keeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		govModuleAddr := k.GetAuthority()

		if !msg.GetSigners()[0].Equals(govModuleAddr) {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "only governance can control exemption zones")
		}

		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case *types.MsgAddTaxExemptionZone:
			res, err := msgServer.AddTaxExemptionZone(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRemoveTaxExemptionZone:
			res, err := msgServer.RemoveTaxExemptionZone(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgModifyTaxExemptionZone:
			res, err := msgServer.ModifyTaxExemptionZone(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgAddTaxExemptionAddress:
			res, err := msgServer.AddTaxExemptionAddress(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRemoveTaxExemptionAddress:
			res, err := msgServer.RemoveTaxExemptionAddress(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized taxexemption message type: %T", msg)
		}
	}
}
