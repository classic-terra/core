package client

import (
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/rest"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	"github.com/terra-money/core/x/treasury/client/cli"
)

// should we support legacy rest?
// general direction of the hub seems to be moving away from legacy rest
var (
	ProposalAddWhitelistHandler    = govclient.NewProposalHandler(cli.ProposalAddWhitelistCmd, emptyRestHandler)
	ProposalRemoveWhitelistHandler = govclient.NewProposalHandler(cli.ProposalRemoveWhitelistCmd, emptyRestHandler)
)

func emptyRestHandler(client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "unsupported-whitelist-service",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			rest.WriteErrorResponse(w, http.StatusBadRequest, "Legacy REST Routes are not supported for whitelist proposals")
		},
	}
}
