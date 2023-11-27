package client

import (
	"github.com/classic-terra/core/v2/x/taxexemption/client/cli"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

// should we support legacy rest?
// general direction of the hub seems to be moving away from legacy rest
var (
	ProposalAddTaxExemptionZoneHandler       = govclient.NewProposalHandler(cli.ProposalAddTaxExemptionZoneCmd)
	ProposalRemoveTaxExemptionZoneHandler    = govclient.NewProposalHandler(cli.ProposalRemoveTaxExemptionZoneCmd)
	ProposalModifyTaxExemptionZoneHandler    = govclient.NewProposalHandler(cli.ProposalModifyTaxExemptionZoneCmd)
	ProposalAddTaxExemptionAddressHandler    = govclient.NewProposalHandler(cli.ProposalAddTaxExemptionAddressCmd)
	ProposalRemoveTaxExemptionAddressHandler = govclient.NewProposalHandler(cli.ProposalRemoveTaxExemptionAddressCmd)
)
