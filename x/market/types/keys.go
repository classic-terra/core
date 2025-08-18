package types

const (
	// ModuleName is the name of the market module
	ModuleName = "market"

	// AccumulatorModuleName is the module account that accumulates redirected tax proceeds
	AccumulatorModuleName = "market_accumulator"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the msg router key for the market module
	RouterKey = ModuleName

	// QuerierRoute is the query router key for the market module
	QuerierRoute = ModuleName
)

// Keys for market store
// Items are stored with the following key: values
//
// - 0x01: sdk.Dec
var (
	// TerraPoolDeltaKey represents terra seigniorage pool.
	TerraPoolDeltaKey = []byte{0x01}

	// EpochLastHeightKey stores the last block height when an epoch processing occurred
	EpochLastHeightKey = []byte{0x20}
)
