package types

const (
	ModuleName = "tax"
	StoreKey   = ModuleName

	RouterKey = ModuleName

	ContextKeyTaxReverseCharge = "tax.reverse_charge"
	ContextKeyTaxDue           = "tax.due"
	ContextKeyTaxPayer         = "tax.payer"

	EventTypeTax                  = "tax_payment"
	EventTypeTaxRefund            = "tax_refund"
	AttributeKeyReverseCharge     = "reverse_charge"
	AttributeValueReverseCharge   = "true"
	AttributeValueNoReverseCharge = "false"
	AttributeKeyTaxAmount         = "tax_amount"
)

// Key defines the store key for tax.
var (
	ParamsKey = []byte{0x1}
)
