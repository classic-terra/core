package types

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "taxexemption"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the message route for treasury
	RouterKey = ModuleName

	// QuerierRoute is the querier route for treasury
	QuerierRoute = ModuleName
)

var (
	// Keys for store prefixes
	TaxExemptionZonePrefix = []byte{0x10} // prefix for burn tax zone list
	TaxExemptionListPrefix = []byte{0x20} // prefix for burn tax exemption list
)
