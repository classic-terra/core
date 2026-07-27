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
// - 0x20: uint64
// - 0x21: int64 (unix timestamp)
// - 0x22<denom>: PriceSnapshots (protobuf, pruned to lookback window)
// - 0x23: int64 (block height)
// - 0x24<denom>: sdk.Int (baseline balance per denom set at epoch change)
// - 0x25<denom>: sdk.Int (daily usage per denom, resets each day)
var (
	// Keys for store prefixed
	TerraPoolDeltaKey = []byte{0x01} // key for terra pool delta which gap between MintPool from BasePool

	// EpochLastHeightKey stores the last block height when an epoch processing occurred
	EpochLastHeightKey = []byte{0x20}

	// LastOracleTallyTimeKey stores the unix timestamp when oracle tally occurred
	LastOracleTallyTimeKey = []byte{0x21}

	// TWAPPriceKey prefix for TWAP price snapshots per denom
	TWAPPriceKey = []byte{0x22}

	// DailyCapResetHeightKey stores the last block height when daily cap was reset
	DailyCapResetHeightKey = []byte{0x23}

	// DailyCapBaselineKey prefix for baseline pool balance per denom (set at epoch change)
	DailyCapBaselineKey = []byte{0x24}

	// DailyCapUsageKey prefix for daily usage per denom (amount drained, resets daily)
	DailyCapUsageKey = []byte{0x25}
)

// GetDailyCapBaselineKey returns the key for daily cap baseline for a given denom
func GetDailyCapBaselineKey(denom string) []byte {
	return append(DailyCapBaselineKey, []byte(denom)...)
}

// GetDailyCapUsageKey returns the key for daily usage tracking for a given denom
func GetDailyCapUsageKey(denom string) []byte {
	return append(DailyCapUsageKey, []byte(denom)...)
}

// GetTWAPPriceKey returns the key for TWAP price snapshots for a given denom
func GetTWAPPriceKey(denom string) []byte {
	return append(TWAPPriceKey, []byte(denom)...)
}
