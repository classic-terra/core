package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalTypeSetWhitelistAddressProposal    = "SetWhitelistAddressProposal"
	ProposalTypeRemoveWhitelistAddressProposal = "RemoveWhitelistAddressProposal"
)

func init() {
	govtypes.RegisterProposalType(ProposalTypeSetWhitelistAddressProposal)
	govtypes.RegisterProposalTypeCodec(&SetWhitelistAddressProposal{}, "terra/SetWhitelistAddressProposal")
	govtypes.RegisterProposalType(ProposalTypeRemoveWhitelistAddressProposal)
	govtypes.RegisterProposalTypeCodec(&RemoveWhitelistAddressProposal{}, "terra/RemoveWhitelistAddressProposal")
}

var (
	_ govtypes.Content = &SetWhitelistAddressProposal{}
	_ govtypes.Content = &RemoveWhitelistAddressProposal{}
)

//======SetWhitelistAddressProposal======

func NewSetWhitelistAddressProposal(title, description string, whitelist []string) govtypes.Content {
	return &SetWhitelistAddressProposal{
		Title:            title,
		Description:      description,
		WhitelistAddress: whitelist,
	}
}

func (p *SetWhitelistAddressProposal) GetTitle() string { return p.Title }

func (p *SetWhitelistAddressProposal) GetDescription() string { return p.Description }

func (p *SetWhitelistAddressProposal) ProposalRoute() string { return RouterKey }

func (p *SetWhitelistAddressProposal) ProposalType() string {
	return ProposalTypeSetWhitelistAddressProposal
}

func (p SetWhitelistAddressProposal) String() string {
	return fmt.Sprintf(`SetWhitelistAddressProposal:
	Title:       		     %s
	Description: 		     %s
	WhitelistAddress: 		 %v
  `, p.Title, p.Description, p.WhitelistAddress)
}

func (p *SetWhitelistAddressProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(p)
	if err != nil {
		return err
	}

	for _, address := range p.WhitelistAddress {
		_, err = sdk.AccAddressFromBech32(address)
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address (%s)", err)
		}
	}

	return nil
}

//======RemoveWhitelistAddressProposal======

func NewRemoveWhitelistAddressProposal(title, description string, whitelist []string) govtypes.Content {
	return &RemoveWhitelistAddressProposal{
		Title:            title,
		Description:      description,
		WhitelistAddress: whitelist,
	}
}

func (p *RemoveWhitelistAddressProposal) GetTitle() string { return p.Title }

func (p *RemoveWhitelistAddressProposal) GetDescription() string { return p.Description }

func (p *RemoveWhitelistAddressProposal) ProposalRoute() string { return RouterKey }

func (p *RemoveWhitelistAddressProposal) ProposalType() string {
	return ProposalTypeRemoveWhitelistAddressProposal
}

func (p RemoveWhitelistAddressProposal) String() string {
	return fmt.Sprintf(`RemoveWhitelistAddressProposal:
	Title:       		 	 %s
	Description: 		 	 %s
	WhitelistAddress: 		 %v
  `, p.Title, p.Description, p.WhitelistAddress)
}

func (p *RemoveWhitelistAddressProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(p)
	if err != nil {
		return err
	}

	for _, address := range p.WhitelistAddress {
		_, err = sdk.AccAddressFromBech32(address)
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address (%s)", err)
		}
	}

	return nil
}
