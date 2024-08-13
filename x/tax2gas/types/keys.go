package types

const (
	ModuleName = "tax2gas"

	StoreKey = ModuleName

	RouterKey = ModuleName

	AnteConsumedGas = "anteConsumedGas"

	TaxGas = "taxGas"

	FinalGasPrices = "finalGasPrices"

	PaidDenom = "paidDenom"
)

// Key defines the store key for tax2gas.
var (
	ParamsKey = []byte{0x1}
)
