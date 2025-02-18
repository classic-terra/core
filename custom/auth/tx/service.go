package tx

import (
	"context"

	gogogrpc "github.com/gogo/protobuf/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	customante "github.com/classic-terra/core/v3/custom/auth/ante"
	taxexemptionkeeper "github.com/classic-terra/core/v3/x/taxexemption/keeper"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ ServiceServer = txServer{}

// txServer is the server for the protobuf Tx service.
type txServer struct {
	clientCtx          client.Context
	treasuryKeeper     customante.TreasuryKeeper
	taxexemptionKeeper taxexemptionkeeper.Keeper
	taxKeeper          customante.TaxKeeper
}

// NewTxServer creates a new Tx service server.
func NewTxServer(clientCtx client.Context, treasuryKeeper customante.TreasuryKeeper, taxKeeper customante.TaxKeeper) ServiceServer {
	return txServer{
		clientCtx:      clientCtx,
		treasuryKeeper: treasuryKeeper,
		taxKeeper:      taxKeeper,
	}
}

// ComputeTax implements the ServiceServer.ComputeTax RPC method.
func (ts txServer) ComputeTax(c context.Context, req *ComputeTaxRequest) (*ComputeTaxResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	var msgs []sdk.Msg
	switch {
	case len(req.TxBytes) != 0:
		tx, err := ts.clientCtx.TxConfig.TxDecoder()(req.TxBytes)
		if err != nil {
			return nil, err
		}
		msgs = tx.GetMsgs()
	case req.Tx != nil:
		msgs = req.Tx.GetMsgs()
	default:
		return nil, status.Errorf(codes.InvalidArgument, "empty txBytes is not allowed")
	}

	taxAmount, _ := customante.FilterMsgAndComputeTax(ctx, ts.taxexemptionKeeper, ts.treasuryKeeper, ts.taxKeeper, false, msgs...)
	return &ComputeTaxResponse{
		TaxAmount: taxAmount,
	}, nil
}

// RegisterTxService registers the tx service on the gRPC router.
func RegisterTxService(
	qrt gogogrpc.Server,
	clientCtx client.Context,
	treasuryKeeper customante.TreasuryKeeper,
	taxKeeper customante.TaxKeeper,
) {
	RegisterServiceServer(
		qrt,
		NewTxServer(clientCtx, treasuryKeeper, taxKeeper),
	)
}

// RegisterGRPCGatewayRoutes mounts the tx service's GRPC-gateway routes on the
// given Mux.
func RegisterGRPCGatewayRoutes(clientConn gogogrpc.ClientConn, mux *runtime.ServeMux) {
	_ = RegisterServiceHandlerClient(context.Background(), mux, NewServiceClient(clientConn))
}

var _ codectypes.UnpackInterfacesMessage = ComputeTaxRequest{}

// UnpackInterfaces implements the UnpackInterfacesMessage interface.
func (m ComputeTaxRequest) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	if m.Tx != nil {
		return m.Tx.UnpackInterfaces(unpacker)
	}

	return nil
}
