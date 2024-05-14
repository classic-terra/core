package client

import (
	"github.com/classic-terra/core/v3/x/taxexemption/client/cli"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var (
	ProposalAddTaxExemptionZoneHandler       = govclient.NewProposalHandler(cli.ProposalAddTaxExemptionZoneCmd)
	ProposalRemoveTaxExemptionZoneHandler    = govclient.NewProposalHandler(cli.ProposalRemoveTaxExemptionZoneCmd)
	ProposalModifyTaxExemptionZoneHandler    = govclient.NewProposalHandler(cli.ProposalModifyTaxExemptionZoneCmd)
	ProposalAddTaxExemptionAddressHandler    = govclient.NewProposalHandler(cli.ProposalAddTaxExemptionAddressCmd)
	ProposalRemoveTaxExemptionAddressHandler = govclient.NewProposalHandler(cli.ProposalRemoveTaxExemptionAddressCmd)
)
