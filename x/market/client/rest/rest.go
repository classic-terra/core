package rest

import (
	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client"
	clientrest "github.com/cosmos/cosmos-sdk/client/rest"
)

// RestDenom is the wildcard part of the request path
const RestDenom = "denom"

// RegisterRoutes registers market-related REST handlers to a router
func RegisterRoutes(cliCtx client.Context, rtr *mux.Router) {
	r := clientrest.WithHTTPDeprecationHeaders(rtr)

	registerQueryRoutes(cliCtx, r)
	registerTxHandlers(cliCtx, r)
}
