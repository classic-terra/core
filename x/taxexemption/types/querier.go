package types

// query endpoints supported by the auth Querier
const (
	QueryParameters        = "parameters"
	QueryTaxExemptionZones = "taxExemptionZones"
	QueryTaxExemptionList  = "taxExemptionList"
)

type QueryTaxExemptionZonesParams struct {
	Page, Limit int
}

func NewQueryTaxExemptionZonesParams(page, limit int) QueryTaxExemptionZonesParams {
	return QueryTaxExemptionZonesParams{page, limit}
}

type QueryTaxExemptionListParams struct {
	Zone        string
	Page, Limit int
}

func NewQueryTaxExemptionListParams(zone string, page, limit int) QueryTaxExemptionListParams {
	return QueryTaxExemptionListParams{zone, page, limit}
}
