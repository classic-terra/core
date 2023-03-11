package bindings

import (
//	sdk "github.com/cosmos/cosmos-sdk/types"
	markettypes "github.com/classic-terra/core/x/market/types"
)

type TerraMsg struct {
	Swap     *markettypes.MsgSwap     `json:"swap,omitempty"`
	SwapSend *markettypes.MsgSwapSend `json:"swap_send,omitempty"`
}
