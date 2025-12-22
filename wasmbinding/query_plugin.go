package wasmbinding

import (
	"encoding/json"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/classic-terra/core/v4/wasmbinding/bindings"
	marketkeeper "github.com/classic-terra/core/v4/x/market/keeper"
	markettypes "github.com/classic-terra/core/v4/x/market/types"
	oracletypes "github.com/classic-terra/core/v4/x/oracle/types"
	treasurytypes "github.com/classic-terra/core/v4/x/treasury/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// TaxCapQueryResponse - tax cap query response for wasm module
type TaxCapQueryResponse struct {
	// uint64 string, eg "1000000"
	Cap string `json:"cap"`
}

// StargateQuerier dispatches whitelisted stargate queries
func StargateQuerier(queryRouter baseapp.GRPCQueryRouter, cdc codec.Codec) func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
	return func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
		protoResponseType, err := GetWhitelistedQuery(request.Path)
		if err != nil {
			return nil, err
		}

		route := queryRouter.Route(request.Path)
		if route == nil {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("No route to query '%s'", request.Path)}
		}

		res, err := route(ctx, &abci.RequestQuery{
			Data: request.Data,
			Path: request.Path,
		})
		if err != nil {
			return nil, err
		}

		bz, err := ConvertProtoToJSONMarshal(protoResponseType, res.Value, cdc)
		if err != nil {
			return nil, err
		}

		return bz, nil
	}
}

// normalizeLegacyRoutedQueryJSON transforms legacy routed shape
// {"route":"treasury|oracle","query_data":{...}}
// into the modern flat TerraQuery JSON understood by bindings.TerraQuery.
// If the request is not a legacy routed query or cannot be normalized,
// the original request is returned unchanged.
func normalizeLegacyRoutedQueryJSON(request json.RawMessage) json.RawMessage {
	type legacyRouted struct {
		Route     string                     `json:"route"`
		QueryData map[string]json.RawMessage `json:"query_data"`
	}

	// limit request size to 64kb to check for legacy (DoS)
	if len(request) > 64<<10 {
		return request
	}

	var lr legacyRouted
	// if it cannot be unmarshaled into legacyRouted, treat as modern TerraQuery
	if err := json.Unmarshal(request, &lr); err != nil || lr.Route == "" {
		return request
	}

	switch lr.Route {
	case treasurytypes.ModuleName:
		if _, ok := lr.QueryData["tax_rate"]; ok {
			// modern tax_rate has empty object
			if bz, err := json.Marshal(map[string]any{"tax_rate": struct{}{}}); err == nil {
				return bz
			}
		}
		if capRaw, ok := lr.QueryData["tax_cap"]; ok {
			// pass inner as-is (object with denom expected by old callers)
			if bz, err := json.Marshal(map[string]json.RawMessage{"tax_cap": capRaw}); err == nil {
				return bz
			}
		}
	case oracletypes.ModuleName:
		if er, ok := lr.QueryData["exchange_rates"]; ok {
			// pass inner as-is (expects {base_denom, quote_denoms})
			if bz, err := json.Marshal(map[string]json.RawMessage{"exchange_rates": er}); err == nil {
				return bz
			}
		}
	case markettypes.ModuleName:
		if sw, ok := lr.QueryData["swap"]; ok {
			// pass inner as-is ({offer_coin, ask_denom})
			if bz, err := json.Marshal(map[string]json.RawMessage{"swap": sw}); err == nil {
				return bz
			}
		}
	}

	// none of the legacy routes matched, return original request
	return request
}

// CustomQuerier dispatches custom CosmWasm bindings queries.
func CustomQuerier(qp *QueryPlugin) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		normalized := normalizeLegacyRoutedQueryJSON(request)
		var contractQuery bindings.TerraQuery
		if err := json.Unmarshal(normalized, &contractQuery); err != nil {
			return nil, errorsmod.Wrap(err, "terra query")
		}

		switch {
		case contractQuery.Swap != nil:
			q := marketkeeper.NewQuerier(*qp.marketKeeper)
			res, err := q.Swap(sdk.WrapSDKContext(ctx), &markettypes.QuerySwapRequest{
				OfferCoin: contractQuery.Swap.OfferCoin.String(),
				AskDenom:  contractQuery.Swap.AskDenom,
			})
			if err != nil {
				return nil, err
			}

			bz, err := json.Marshal(bindings.SwapQueryResponse{Receive: ConvertSdkCoinToWasmCoin(res.ReturnCoin)})
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
			}

			return bz, nil

		case contractQuery.ExchangeRates != nil:
			// LUNA / BASE_DENOM
			baseDenomExchangeRate, err := qp.oracleKeeper.GetLunaExchangeRate(ctx, contractQuery.ExchangeRates.BaseDenom)
			if err != nil {
				return nil, err
			}

			var items []bindings.ExchangeRateItem
			for _, quoteDenom := range contractQuery.ExchangeRates.QuoteDenoms {
				// LUNA / QUOTE_DENOM
				quoteDenomExchangeRate, err := qp.oracleKeeper.GetLunaExchangeRate(ctx, quoteDenom)
				if err != nil {
					continue
				}

				// (LUNA / QUOTE_DENOM) / (BASE_DENOM / LUNA) = BASE_DENOM / QUOTE_DENOM
				items = append(items, bindings.ExchangeRateItem{
					ExchangeRate: quoteDenomExchangeRate.Quo(baseDenomExchangeRate).String(),
					QuoteDenom:   quoteDenom,
				})
			}

			bz, err := json.Marshal(bindings.ExchangeRatesQueryResponse{
				BaseDenom:     contractQuery.ExchangeRates.BaseDenom,
				ExchangeRates: items,
			})
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
			}

			return bz, nil

		case contractQuery.TaxRate != nil:
			taxRate := qp.treasuryKeeper.GetTaxRate(ctx)
			bz, err := json.Marshal(bindings.TaxRateQueryResponse{Rate: taxRate.String()})
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
			}

			return bz, nil

		case contractQuery.TaxCap != nil:
			taxCap := qp.treasuryKeeper.GetTaxCap(ctx, contractQuery.TaxCap.Denom)
			bz, err := json.Marshal(TaxCapQueryResponse{Cap: taxCap.String()})
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
			}

			return bz, nil

		default:
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown terra query variant"}
		}
	}
}
