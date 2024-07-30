package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgAddTaxExemptionZone{}
	_ sdk.Msg = &MsgRemoveTaxExemptionZone{}
	_ sdk.Msg = &MsgModifyTaxExemptionZone{}
	_ sdk.Msg = &MsgAddTaxExemptionAddress{}
	_ sdk.Msg = &MsgRemoveTaxExemptionAddress{}
)

// ======MsgAddTaxExemptionZone======

func NewMsgAddTaxExemptionZone(title, description, zone string, outgoing, incoming, crossZone bool, addresses []string, authority sdk.AccAddress) *MsgAddTaxExemptionZone {
	return &MsgAddTaxExemptionZone{
		Zone:      zone,
		Outgoing:  outgoing,
		Incoming:  incoming,
		CrossZone: crossZone,
		Addresses: addresses,
		Authority: authority.String(),
	}
}

func (msg MsgAddTaxExemptionZone) Type() string { return "AddTaxExemptionZone" }

func (msg *MsgAddTaxExemptionZone) GetZone() string { return msg.Zone }

func (msg *MsgAddTaxExemptionZone) Route() string { return RouterKey }

func (msg MsgAddTaxExemptionZone) String() string {
	return fmt.Sprintf(`MsgAddTaxExemptionZone:
	  Authority:	   %s
	  Zone:        %s
	  Outgoing:    %t
	  Incoming:    %t
	  CrossZone:   %t
	  Addresses:   %s`,
		msg.Authority, msg.Zone, msg.Outgoing, msg.Incoming, msg.CrossZone, msg.Addresses)
}

func (msg MsgAddTaxExemptionZone) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return []sdk.AccAddress{}
	}

	return []sdk.AccAddress{addr}
}

func (msg MsgAddTaxExemptionZone) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

func (msg MsgAddTaxExemptionZone) ValidateBasic() error {
	if len(msg.Zone) == 0 {
		return fmt.Errorf("zone cannot be empty")
	}
	if len(msg.Addresses) == 0 {
		return fmt.Errorf("addresses cannot be empty")
	}
	return nil
}

// ======MsgRemoveTaxExemptionZone======

func NewMsgRemoveTaxExemptionZone(title, description, zone string, authority sdk.AccAddress) *MsgRemoveTaxExemptionZone {
	return &MsgRemoveTaxExemptionZone{
		Zone:      zone,
		Authority: authority.String(),
	}
}

func (msg MsgRemoveTaxExemptionZone) Type() string { return "RemoveTaxExemptionZone" }

func (msg *MsgRemoveTaxExemptionZone) GetZone() string { return msg.Zone }

func (msg *MsgRemoveTaxExemptionZone) Route() string { return RouterKey }

func (msg MsgRemoveTaxExemptionZone) String() string {
	return fmt.Sprintf(`MsgRemoveTaxExemptionZone:
	  Authority:	   %s
	  Zone:        %s`,
		msg.Authority, msg.Zone)
}

func (msg MsgRemoveTaxExemptionZone) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return []sdk.AccAddress{}
	}

	return []sdk.AccAddress{addr}
}

func (msg MsgRemoveTaxExemptionZone) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

func (msg MsgRemoveTaxExemptionZone) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if len(msg.Zone) == 0 {
		return fmt.Errorf("zone cannot be empty")
	}
	return nil
}

// ======MsgModifyTaxExemptionZone======

func NewMsgModifyTaxExemptionZone(title, description, zone string, outgoing, incoming, crossZone bool, authority sdk.AccAddress) *MsgModifyTaxExemptionZone {
	return &MsgModifyTaxExemptionZone{
		Zone:      zone,
		Outgoing:  outgoing,
		Incoming:  incoming,
		CrossZone: crossZone,
		Authority: authority.String(),
	}
}

func (msg MsgModifyTaxExemptionZone) Type() string { return "ModifyTaxExemptionZone" }

func (msg *MsgModifyTaxExemptionZone) GetZone() string { return msg.Zone }

func (msg *MsgModifyTaxExemptionZone) Route() string { return RouterKey }

func (msg MsgModifyTaxExemptionZone) String() string {
	return fmt.Sprintf(`MsgModifyTaxExemptionZone:
	  Authority: 	   %s
	  Zone:        %s
	  Outgoing:    %t
	  Incoming:    %t
	  CrossZone:   %t`,
		msg.Authority, msg.Zone, msg.Outgoing, msg.Incoming, msg.CrossZone)
}

func (msg MsgModifyTaxExemptionZone) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return []sdk.AccAddress{}
	}

	return []sdk.AccAddress{addr}
}

func (msg MsgModifyTaxExemptionZone) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

func (msg MsgModifyTaxExemptionZone) ValidateBasic() error {
	if len(msg.Zone) == 0 {
		return fmt.Errorf("zone cannot be empty")
	}
	return nil
}

// ======MsgAddTaxExemptionAddress======

func NewMsgAddTaxExemptionAddress(title, description, zone string, addresses []string, authority sdk.AccAddress) *MsgAddTaxExemptionAddress {
	return &MsgAddTaxExemptionAddress{
		Zone:      zone,
		Addresses: addresses,
		Authority: authority.String(),
	}
}

func (msg MsgAddTaxExemptionAddress) Type() string { return "AddTaxExemptionAddress" }

func (msg *MsgAddTaxExemptionAddress) GetZone() string { return msg.Zone }

func (msg *MsgAddTaxExemptionAddress) Route() string { return RouterKey }

func (msg MsgAddTaxExemptionAddress) String() string {
	return fmt.Sprintf(`MsgAddTaxExemptionAddress:
	  Authority:	   %s
	  Zone:        %s
	  Addresses:   %s`,
		msg.Authority, msg.Zone, msg.Addresses)
}

func (msg MsgAddTaxExemptionAddress) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return []sdk.AccAddress{}
	}

	return []sdk.AccAddress{addr}
}

func (msg MsgAddTaxExemptionAddress) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

func (msg MsgAddTaxExemptionAddress) ValidateBasic() error {
	if len(msg.Zone) == 0 {
		return fmt.Errorf("zone cannot be empty")
	}
	if len(msg.Addresses) == 0 {
		return fmt.Errorf("addresses cannot be empty")
	}
	return nil
}

// ======MsgRemoveTaxExemptionAddress======

func NewMsgRemoveTaxExemptionAddress(title, description, zone string, addresses []string, authority sdk.AccAddress) *MsgRemoveTaxExemptionAddress {
	return &MsgRemoveTaxExemptionAddress{
		Zone:      zone,
		Addresses: addresses,
		Authority: authority.String(),
	}
}

func (msg MsgRemoveTaxExemptionAddress) Type() string { return "RemoveTaxExemptionAddress" }

func (msg *MsgRemoveTaxExemptionAddress) GetZone() string { return msg.Zone }

func (msg *MsgRemoveTaxExemptionAddress) Route() string { return RouterKey }

func (msg MsgRemoveTaxExemptionAddress) String() string {
	return fmt.Sprintf(`MsgRemoveTaxExemptionAddress:
	  Authority:	   %s
	  Zone:        %s
	  Addresses:   %s`,
		msg.Authority, msg.Zone, msg.Addresses)
}

func (msg MsgRemoveTaxExemptionAddress) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return []sdk.AccAddress{}
	}

	return []sdk.AccAddress{addr}
}

func (msg MsgRemoveTaxExemptionAddress) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

func (msg MsgRemoveTaxExemptionAddress) ValidateBasic() error {
	if len(msg.Zone) == 0 {
		return fmt.Errorf("zone cannot be empty")
	}
	if len(msg.Addresses) == 0 {
		return fmt.Errorf("addresses cannot be empty")
	}
	return nil
}
