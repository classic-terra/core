package types

import "encoding/binary"

// Keys for governance store
// Items are stored with the following key: values
var (
	// Minimum UUSD amount prefix
	UUSDMinKeyPrefix = []byte{0x40}
)

// GetProposalIDBytes returns the byte representation of the proposalID
func GetProposalIDBytes(proposalID uint64) (proposalIDBz []byte) {
	proposalIDBz = make([]byte, 8)
	binary.BigEndian.PutUint64(proposalIDBz, proposalID)
	return
}

// GetAmountBytes returns the byte representation of the amount
func GetAmountBytes(amount uint64) (amountBz []byte) {
	amountBz = make([]byte, 8)
	binary.BigEndian.PutUint64(amountBz, amount)
	return
}

// GetAmountFromBytes returns amount in uint64 format from a byte array
func GetAmountFromBytes(bz []byte) (amount uint64) {
	return binary.BigEndian.Uint64(bz)
}

// TotalDepositKey of the specific total amount to deposit based on the proposalID from the store
func TotalDepositKey(proposalID uint64) []byte {
	return append(UUSDMinKeyPrefix, GetProposalIDBytes(proposalID)...)
}
