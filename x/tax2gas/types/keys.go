package types

const (
	ModuleName = "tax2gas"

	StoreKey = ModuleName

	RouterKey = ModuleName

	ConsumedGasFee = "consumedGasFee"

	TaxGas = "taxGas"

	FeeDenom = "feeDenom"
)

// Key defines the store key for tax2gas.
var (
	ParamsKey = []byte{0x1}
)