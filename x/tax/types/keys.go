package types

const (
	ModuleName = "tax"
	StoreKey   = ModuleName

	RouterKey = ModuleName

	ContextKeyTaxReverseCharge = "tax.reverse_charge"
	ContextKeyWasmFunds        = "tax.wasm_funds"
	ContextKeyWasmBalance      = "tax.wasm_balance"

	EventTypeTax                  = "tax_payment"
	AttributeKeyReverseCharge     = "reverse_charge"
	AttributeValueReverseCharge   = "true"
	AttributeValueNoReverseCharge = "false"
	AttributeKeyTaxAmount         = "tax_amount"
)

// Key defines the store key for tax.
var (
	ParamsKey = []byte{0x1}
)
