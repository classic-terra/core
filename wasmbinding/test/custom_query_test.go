package wasmbinding

import (
	"encoding/json"
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	"github.com/classic-terra/core/app"
	core "github.com/classic-terra/core/types"
	"github.com/classic-terra/core/wasmbinding/bindings"
	markettypes "github.com/classic-terra/core/x/market/types"
	treasurytypes "github.com/classic-terra/core/x/treasury/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// go test -v -run ^TestQuerySwap$ github.com/classic-terra/core/wasmbinding/test
// oracle rate: 1 uluna = 1.7 usdr
// 1000 uluna from trader goes to contract
// 1666 usdr (after 2% tax) is swapped into
func TestQuerySwap(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := CreateTestInput(t)

	// fund
	FundAccount(t, ctx, app, actor)

	// instantiate reflect contract
	contractAddr := InstantiateReflectContract(t, ctx, app, actor)
	require.NotEmpty(t, contractAddr)

	// setup swap environment
	// Set Oracle Price
	lunaPriceInSDR := sdk.NewDecWithPrec(17, 1)
	app.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroSDRDenom, lunaPriceInSDR)

	// Calculate expected swapped SDR
	expectedSwappedSDR := sdk.NewDec(1000).Mul(lunaPriceInSDR)
	tax := markettypes.DefaultMinStabilitySpread.Mul(expectedSwappedSDR)
	expectedSwappedSDR = expectedSwappedSDR.Sub(tax)

	// query swap
	query := bindings.TerraQuery{
		Swap: &markettypes.QuerySwapParams{
			OfferCoin: sdk.NewCoin(core.MicroLunaDenom, sdk.NewInt(1000)),
			AskDenom:  core.MicroSDRDenom,
		},
	}

	resp := bindings.SwapQueryResponse{}
	queryCustom(t, ctx, app, contractAddr, query, &resp)

	require.Equal(t, expectedSwappedSDR.TruncateInt().String(), resp.Receive.Amount)
}

// go test -v -run ^TestQueryExchangeRates$ github.com/classic-terra/core/wasmbinding/test
func TestQueryExchangeRates(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := CreateTestInput(t)

	// fund
	FundAccount(t, ctx, app, actor)

	// instantiate reflect contract
	contractAddr := InstantiateReflectContract(t, ctx, app, actor)
	require.NotEmpty(t, contractAddr)

	lunaPriceInSDR := sdk.NewDecWithPrec(17, 1)
	app.OracleKeeper.SetLunaExchangeRate(ctx, core.MicroSDRDenom, lunaPriceInSDR)

	query := bindings.TerraQuery{
		ExchangeRates: &bindings.ExchangeRateQueryParams{
			BaseDenom:   core.MicroLunaDenom,
			QuoteDenoms: []string{core.MicroSDRDenom},
		},
	}

	resp := bindings.ExchangeRatesQueryResponse{}
	queryCustom(t, ctx, app, contractAddr, query, &resp)

	require.Equal(t, lunaPriceInSDR.String(), resp.ExchangeRates[0].ExchangeRate)
}

// go test -v -run ^TestQueryTaxRate$ github.com/classic-terra/core/wasmbinding/test
func TestQueryTaxRate(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := CreateTestInput(t)

	// fund
	FundAccount(t, ctx, app, actor)

	// instantiate reflect contract
	contractAddr := InstantiateReflectContract(t, ctx, app, actor)
	require.NotEmpty(t, contractAddr)

	query := bindings.TerraQuery{
		TaxRate: &struct{}{},
	}

	resp := bindings.TaxRateQueryResponse{}
	queryCustom(t, ctx, app, contractAddr, query, &resp)

	require.Equal(t, treasurytypes.DefaultTaxRate.String(), resp.Rate)
}

// go test -v -run ^TestQueryTaxCap$ github.com/classic-terra/core/wasmbinding/test
func TestQueryTaxCap(t *testing.T) {
	actor := RandomAccountAddress()
	app, ctx := CreateTestInput(t)

	// fund
	FundAccount(t, ctx, app, actor)

	// instantiate reflect contract
	contractAddr := InstantiateReflectContract(t, ctx, app, actor)
	require.NotEmpty(t, contractAddr)

	query := bindings.TerraQuery{
		TaxCap: &treasurytypes.QueryTaxCapParams{
			Denom: core.MicroSDRDenom,
		},
	}

	resp := bindings.TaxCapQueryResponse{}
	queryCustom(t, ctx, app, contractAddr, query, &resp)

	require.Equal(t, treasurytypes.DefaultTaxPolicy.Cap.Amount.String(), resp.Cap)
}

type ReflectQuery struct {
	Chain *ChainRequest `json:"chain,omitempty"`
}

type ChainRequest struct {
	Request wasmvmtypes.QueryRequest `json:"request"`
}

type ChainResponse struct {
	Data []byte `json:"data"`
}

func queryCustom(t *testing.T, ctx sdk.Context, app *app.TerraApp, contract sdk.AccAddress, request bindings.TerraQuery, response interface{}) {
	t.Helper()

	msgBz, err := json.Marshal(request)
	require.NoError(t, err)

	query := ReflectQuery{
		Chain: &ChainRequest{
			Request: wasmvmtypes.QueryRequest{Custom: msgBz},
		},
	}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)

	resBz, err := app.WasmKeeper.QuerySmart(ctx, contract, queryBz)
	require.NoError(t, err)
	var resp ChainResponse
	err = json.Unmarshal(resBz, &resp)
	require.NoError(t, err)
	err = json.Unmarshal(resp.Data, response)
	require.NoError(t, err)
}
