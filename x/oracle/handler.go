package oracle

// Deprecated in SDK 0.50 - handler pattern replaced with msg servers

// import (
//     errorsmod "cosmossdk.io/errors"
//     sdk "github.com/cosmos/cosmos-sdk/types"
//     sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
//
//     "github.com/classic-terra/core/v3/x/oracle/keeper"
//     "github.com/classic-terra/core/v3/x/oracle/types"
// )
//
// // NewHandler returns a handler for "oracle" type messages.
// // Deprecated in SDK 0.50 - use msg servers instead
// func NewHandler(k keeper.Keeper) sdk.Handler {
//     msgServer := keeper.NewMsgServerImpl(k)
//
//     return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
//         ctx = ctx.WithEventManager(sdk.NewEventManager())
//
//         switch msg := msg.(type) {
//         case *types.MsgDelegateFeedConsent:
//             res, err := msgServer.DelegateFeedConsent(sdk.WrapSDKContext(ctx), msg)
//             return sdk.WrapServiceResult(ctx, res, err)
//         case *types.MsgAggregateExchangeRatePrevote:
//             res, err := msgServer.AggregateExchangeRatePrevote(sdk.WrapSDKContext(ctx), msg)
//             return sdk.WrapServiceResult(ctx, res, err)
//         case *types.MsgAggregateExchangeRateVote:
//             res, err := msgServer.AggregateExchangeRateVote(sdk.WrapSDKContext(ctx), msg)
//             return sdk.WrapServiceResult(ctx, res, err)
//         default:
//             return nil, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized %s message type: %T", types.ModuleName, msg)
//         }
//     }
// }
