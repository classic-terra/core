package types

import (
	fmt "fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	ProposalTypeAddTaxExemptionZone       = "AddTaxExemptionZone"
	ProposalTypeRemoveTaxExemptionZone    = "RemoveTaxExemptionZone"
	ProposalTypeModifyTaxExemptionZone    = "ModifyTaxExemptionZone"
	ProposalTypeAddTaxExemptionAddress    = "AddTaxExemptionAddress"
	ProposalTypeRemoveTaxExemptionAddress = "RemoveTaxExemptionAddress"
)

var (
	ErrUnknownAddress = errors.Register(ModuleName, 1001, "unknown address")
	ErrInvalidAddress = errors.Register(ModuleName, 1002, "invalid address")
)

func init() {
	govv1beta1.RegisterProposalType(ProposalTypeAddTaxExemptionZone)
	govv1beta1.RegisterProposalType(ProposalTypeRemoveTaxExemptionZone)
	govv1beta1.RegisterProposalType(ProposalTypeModifyTaxExemptionZone)
	govv1beta1.RegisterProposalType(ProposalTypeAddTaxExemptionAddress)
	govv1beta1.RegisterProposalType(ProposalTypeRemoveTaxExemptionAddress)
}

var (
	_ govv1beta1.Content = &AddTaxExemptionZoneProposal{}
	_ govv1beta1.Content = &RemoveTaxExemptionZoneProposal{}
	_ govv1beta1.Content = &ModifyTaxExemptionZoneProposal{}
	_ govv1beta1.Content = &AddTaxExemptionAddressProposal{}
	_ govv1beta1.Content = &RemoveTaxExemptionAddressProposal{}
)

// ======AddTaxExemptionZoneProposal======

func (p *AddTaxExemptionZoneProposal) GetTitle() string { return p.Title }

func (p *AddTaxExemptionZoneProposal) GetDescription() string { return p.Description }

func (p *AddTaxExemptionZoneProposal) GetZone() string { return p.Zone }

func (p *AddTaxExemptionZoneProposal) ProposalRoute() string { return RouterKey }

func (p *AddTaxExemptionZoneProposal) ProposalType() string {
	return ProposalTypeAddTaxExemptionZone
}

func (p AddTaxExemptionZoneProposal) String() string {
	return fmt.Sprintf(`AddTaxExemptionZoneProposal:
	Title:       %s
	Description: %s
	Zone: 	  	 %s
	Outgoing:    %t
	Incoming:    %t
	CrossZone:   %t
  `, p.Title, p.Description, p.Zone, p.Outgoing, p.Incoming, p.CrossZone)
}

func (p *AddTaxExemptionZoneProposal) ValidateBasic() error {
	err := govv1beta1.ValidateAbstract(p)
	if err != nil {
		return err
	}

	if p.Zone == "" {
		return errors.Wrap(ErrUnknownAddress, "zone name cannot be empty")
	}

	for _, address := range p.Addresses {
		_, err = sdk.AccAddressFromBech32(address)
		if err != nil {
			return errors.Wrapf(ErrInvalidAddress, "%s: %s", err, address)
		}
	}

	return nil
}

// ======RemoveTaxExemptionZoneProposal======

func (p *RemoveTaxExemptionZoneProposal) GetTitle() string { return p.Title }

func (p *RemoveTaxExemptionZoneProposal) GetDescription() string { return p.Description }

func (p *RemoveTaxExemptionZoneProposal) GetZone() string { return p.Zone }

func (p *RemoveTaxExemptionZoneProposal) ProposalRoute() string { return RouterKey }

func (p *RemoveTaxExemptionZoneProposal) ProposalType() string {
	return ProposalTypeRemoveTaxExemptionZone
}

func (p RemoveTaxExemptionZoneProposal) String() string {
	return fmt.Sprintf(`RemoveTaxExemptionZoneProposal:
	Title:       %s
	Description: %s
	Zone: 	  	 %s
  `, p.Title, p.Description, p.Zone)
}

func (p *RemoveTaxExemptionZoneProposal) ValidateBasic() error {
	err := govv1beta1.ValidateAbstract(p)
	if err != nil {
		return err
	}

	if p.Zone == "" {
		return errors.Wrap(ErrUnknownAddress, "zone name cannot be empty")
	}

	return nil
}

// ======ModifyTaxExemptionZoneProposal======

func (p *ModifyTaxExemptionZoneProposal) GetTitle() string { return p.Title }

func (p *ModifyTaxExemptionZoneProposal) GetDescription() string { return p.Description }

func (p *ModifyTaxExemptionZoneProposal) GetZone() string { return p.Zone }

func (p *ModifyTaxExemptionZoneProposal) ProposalRoute() string { return RouterKey }

func (p *ModifyTaxExemptionZoneProposal) ProposalType() string {
	return ProposalTypeModifyTaxExemptionZone
}

func (p ModifyTaxExemptionZoneProposal) String() string {
	return fmt.Sprintf(`ModifyTaxExemptionZoneProposal:
	Title:       %s
	Description: %s
	Zone: 	  	 %s
	Outgoing:    %t
	Incoming:    %t
	CrossZone:   %t
  `, p.Title, p.Description, p.Zone, p.Outgoing, p.Incoming, p.CrossZone)
}

func (p *ModifyTaxExemptionZoneProposal) ValidateBasic() error {
	err := govv1beta1.ValidateAbstract(p)
	if err != nil {
		return err
	}

	if p.Zone == "" {
		return errors.Wrap(ErrUnknownAddress, "zone name cannot be empty")
	}

	return nil
}

// ======AddTaxExemptionAddressProposal======

func (p *AddTaxExemptionAddressProposal) GetTitle() string { return p.Title }

func (p *AddTaxExemptionAddressProposal) GetDescription() string { return p.Description }

func (p *AddTaxExemptionAddressProposal) GetZone() string { return p.Zone }

func (p *AddTaxExemptionAddressProposal) ProposalRoute() string { return RouterKey }

func (p *AddTaxExemptionAddressProposal) ProposalType() string {
	return ProposalTypeAddTaxExemptionAddress
}

func (p AddTaxExemptionAddressProposal) String() string {
	return fmt.Sprintf(`AddTaxExemptionAddressProposal:
	Title:       %s
	Description: %s
	Zone: 	  	 %s
	Addresses:   %v
  `, p.Title, p.Description, p.Zone, p.Addresses)
}

func (p *AddTaxExemptionAddressProposal) ValidateBasic() error {
	err := govv1beta1.ValidateAbstract(p)
	if err != nil {
		return err
	}

	for _, address := range p.Addresses {
		_, err = sdk.AccAddressFromBech32(address)
		if err != nil {
			return errors.Wrapf(ErrInvalidAddress, "%s: %s", err, address)
		}
	}

	return nil
}

// ======RemoveTaxExemptionAddressProposal======

func (p *RemoveTaxExemptionAddressProposal) GetTitle() string { return p.Title }

func (p *RemoveTaxExemptionAddressProposal) GetDescription() string { return p.Description }

func (p *RemoveTaxExemptionAddressProposal) GetZone() string { return p.Zone }

func (p *RemoveTaxExemptionAddressProposal) ProposalRoute() string { return RouterKey }

func (p *RemoveTaxExemptionAddressProposal) ProposalType() string {
	return ProposalTypeRemoveTaxExemptionAddress
}

func (p RemoveTaxExemptionAddressProposal) String() string {
	return fmt.Sprintf(`RemoveTaxExemptionAddressProposal:
	Title:       %s
	Description: %s
	Addresses:   %v
  `, p.Title, p.Description, p.Addresses)
}

func (p *RemoveTaxExemptionAddressProposal) ValidateBasic() error {
	err := govv1beta1.ValidateAbstract(p)
	if err != nil {
		return err
	}

	for _, address := range p.Addresses {
		_, err = sdk.AccAddressFromBech32(address)
		if err != nil {
			return errors.Wrapf(ErrInvalidAddress, "%s: %s", err, address)
		}
	}

	return nil
}
