package bindings

import(
	markettypes "github.com/classic-terra/core/x/market/types"
//	oracletypes "github.com/classic-terra/core/x/oracle/types"
	treasurytypes "github.com/classic-terra/core/x/treasury/types"
)

// ExchangeRateQueryParams query request params for exchange rates
type ExchangeRateQueryParams struct {
	BaseDenom   string   `json:"base_denom"`
	QuoteDenoms []string `json:"quote_denoms"`
}

// TerraQuery contains terra custom queries.
type TerraQuery struct {
	Swap *markettypes.QuerySwapParams `json:"swap,omitempty"`
	ExchangeRates *ExchangeRateQueryParams `json:"exchange_rates,omitempty"`
	TaxRate *struct{}                `json:"tax_rate,omitempty"`
	TaxCap  *treasurytypes.QueryTaxCapParams `json:"tax_cap,omitempty"`
}

