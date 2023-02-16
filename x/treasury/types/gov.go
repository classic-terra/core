package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalTypeAddBurnTaxExemptionAddress    = "AddBurnTaxExemptionAddress"
	ProposalTypeRemoveBurnTaxExemptionAddress = "RemoveBurnTaxExemptionAddress"
)

func init() {
	govtypes.RegisterProposalType(ProposalTypeAddBurnTaxExemptionAddress)
	govtypes.RegisterProposalTypeCodec(&AddBurnTaxExemptionAddressProposal{}, "terra/AddBurnTaxExemptionAddressProposal")
	govtypes.RegisterProposalType(ProposalTypeRemoveBurnTaxExemptionAddress)
	govtypes.RegisterProposalTypeCodec(&RemoveBurnTaxExemptionAddressProposal{}, "terra/RemoveBurnTaxExemptionAddressProposal")
}

var (
	_ govtypes.Content = &AddBurnTaxExemptionAddressProposal{}
	_ govtypes.Content = &RemoveBurnTaxExemptionAddressProposal{}
)

// ======AddBurnTaxExemptionAddressProposal======

func NewAddBurnTaxExemptionAddressProposal(title, description string, exempt_list []string) govtypes.Content {
	return &AddBurnTaxExemptionAddressProposal{
		Title:            title,
		Description:      description,
		ExemptionAddress: exempt_list,
	}
}

func (p *AddBurnTaxExemptionAddressProposal) GetTitle() string { return p.Title }

func (p *AddBurnTaxExemptionAddressProposal) GetDescription() string { return p.Description }

func (p *AddBurnTaxExemptionAddressProposal) ProposalRoute() string { return RouterKey }

func (p *AddBurnTaxExemptionAddressProposal) ProposalType() string {
	return ProposalTypeAddBurnTaxExemptionAddress
}

func (p AddBurnTaxExemptionAddressProposal) String() string {
	return fmt.Sprintf(`AddBurnTaxExemptionAddressProposal:
	Title:       		     %s
	Description: 		     %s
	ExtemptionAddress: 		 %v
  `, p.Title, p.Description, p.ExemptionAddress)
}

func (p *AddBurnTaxExemptionAddressProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(p)
	if err != nil {
		return err
	}

	for _, address := range p.ExemptionAddress {
		_, err = sdk.AccAddressFromBech32(address)
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address (%s)", err)
		}
	}

	return nil
}

// ======RemoveBurnTaxExemptionAddressProposal======

func NewRemoveBurnTaxExemptionAddressProposal(title, description string, exempt_list []string) govtypes.Content {
	return &RemoveBurnTaxExemptionAddressProposal{
		Title:            title,
		Description:      description,
		ExemptionAddress: exempt_list,
	}
}

func (p *RemoveBurnTaxExemptionAddressProposal) GetTitle() string { return p.Title }

func (p *RemoveBurnTaxExemptionAddressProposal) GetDescription() string { return p.Description }

func (p *RemoveBurnTaxExemptionAddressProposal) ProposalRoute() string { return RouterKey }

func (p *RemoveBurnTaxExemptionAddressProposal) ProposalType() string {
	return ProposalTypeRemoveBurnTaxExemptionAddress
}

func (p RemoveBurnTaxExemptionAddressProposal) String() string {
	return fmt.Sprintf(`RemoveBurnTaxExemptionAddressProposal:
	Title:       		 	 %s
	Description: 		 	 %s
	ExtemptionAddress: 		 %v
  `, p.Title, p.Description, p.ExemptionAddress)
}

func (p *RemoveBurnTaxExemptionAddressProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(p)
	if err != nil {
		return err
	}

	for _, address := range p.ExemptionAddress {
		_, err = sdk.AccAddressFromBech32(address)
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address (%s)", err)
		}
	}

	return nil
}
