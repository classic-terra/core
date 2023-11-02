package types

const (
	// ModuleName defines the module's name.
	ModuleName = "classictax"
	// StoreKey is the string store representation
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
	// CtxFeeKey is used for the context to report deducted fees to post handler
	CtxFeeKey = "classictax_fee"
)
