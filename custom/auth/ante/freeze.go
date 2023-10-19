package ante

import (
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	markettypes "github.com/classic-terra/core/v2/x/market/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	hostkeeper "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/keeper"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
)

/*var (
	BlockedAddr = map[string]bool{}
)*/

// FreezeDecorator freezes wallets that should
// not interact with the blockchain
type FreezeDecorator struct {
	cdc            codec.BinaryCodec
	hostkeeper     hostkeeper.Keeper
	treasurykeeper TreasuryKeeper
}

func NewFreezeDecorator(
	cdc codec.BinaryCodec,
	keeper hostkeeper.Keeper,
	tk TreasuryKeeper,
) FreezeDecorator {
	return FreezeDecorator{
		cdc:            cdc,
		hostkeeper:     keeper,
		treasurykeeper: tk,
	}
}

func (fd FreezeDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {

	msgs := tx.GetMsgs()
	return ctx, fd.FilterMsgs(ctx, msgs)

}

func (fd FreezeDecorator) FilterMsgs(ctx sdk.Context, msgs []sdk.Msg) error {

	BlockedAddr := fd.treasurykeeper.GetFreezeAddrs(ctx)

	for _, msg := range msgs {

		switch v := msg.(type) {

		case *banktypes.MsgSend:
			addr, _ := sdk.AccAddressFromBech32(v.FromAddress)
			if BlockedAddr.Contains(addr) {
				return fmt.Errorf("blocked address %s", v.FromAddress)
			}
		case *banktypes.MsgMultiSend:
			for _, input := range v.Inputs {
				addr, _ := sdk.AccAddressFromBech32(input.Address)
				if BlockedAddr.Contains(addr) {
					return fmt.Errorf("blocked address %s", addr)
				}
			}
		case *markettypes.MsgSwapSend:
			addr, _ := sdk.AccAddressFromBech32(v.FromAddress)
			if BlockedAddr.Contains(addr) {
				return fmt.Errorf("blocked address %s", v.FromAddress)
			}
		case *markettypes.MsgSwap:
			addr, _ := sdk.AccAddressFromBech32(v.Trader)
			if BlockedAddr.Contains(addr) {
				return fmt.Errorf("blocked address %s", v.Trader)
			}
		case *wasmtypes.MsgExecuteContract:
			addr, _ := sdk.AccAddressFromBech32(v.Sender)
			if BlockedAddr.Contains(addr) {
				return fmt.Errorf("blocked address %s", v.Sender)
			}
		case *wasmtypes.MsgInstantiateContract:
			addr, _ := sdk.AccAddressFromBech32(v.Sender)
			if BlockedAddr.Contains(addr) {
				return fmt.Errorf("blocked address %s", v.Sender)
			}
		case *ibctransfertypes.MsgTransfer:
			addr, _ := sdk.AccAddressFromBech32(v.Sender)
			if BlockedAddr.Contains(addr) {
				return fmt.Errorf("blocked address %s", v.Sender)
			}
		case *authztypes.MsgExec:
			msgs, err := v.GetMessages()
			if err != nil {
				continue
			}
			return fd.FilterMsgs(ctx, msgs)
		case *channeltypes.MsgRecvPacket:
			return fd.FilterIbcPacket(ctx, v.Packet)
		default:
			return nil
		}
	}

	return nil

}

// FilterIbcPacket filters ICA Host messages
func (fd FreezeDecorator) FilterIbcPacket(ctx sdk.Context, packet channeltypes.Packet) error {

	var data icatypes.InterchainAccountPacketData
	if icatypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data) != nil {
		// jibberish or no ICA packet - we are good
		return nil
	}

	switch data.Type {
	case icatypes.EXECUTE_TX:
		msgs, err := icatypes.DeserializeCosmosTx(fd.cdc, data.Data)
		if err != nil {
			return nil
		}
		return fd.FilterMsgs(ctx, msgs)
	default:
		return nil
	}

}
