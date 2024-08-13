package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TypeMsgUpdateParams = "update_params"
)

var _ sdk.Msg = &MsgUpdateParams{}

func (msg MsgUpdateParams) Route() string { return ModuleName }
func (msg MsgUpdateParams) Type() string  { return TypeMsgUpdateParams }
func (msg MsgUpdateParams) ValidateBasic() error {
	return msg.Params.Validate()
}

func (msg MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}
