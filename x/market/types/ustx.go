package types

// USTX represents the new stable coin pegged to $1.00.
type USTX struct {
    Amount int64  `json:"amount"`
    Denom  string `json:"denom"`
}

// NewUSTX creates a new USTX token with the given amount.
func NewUSTX(amount int64) USTX {
    return USTX{
        Amount: amount,
        Denom:  "ustx",
    }
}
