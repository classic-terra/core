package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/terra-money/core/x/treasury/client/cli"
)

// should we support legacy rest?
// general direction of the hub seems to be moving away from legacy rest
var (
	ProposalAddWhitelistHandler    = govclient.NewProposalHandler(cli.ProposalAddWhitelistCmd, nil)
	ProposalRemoveWhitelistHandler = govclient.NewProposalHandler(cli.ProposalRemoveWhitelistCmd, nil)
)
