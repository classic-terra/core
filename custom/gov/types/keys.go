package types

import "encoding/binary"

// Keys for governance store
// Items are stored with the following key: values
var (
	// Minimum USTC amount prefix
	USTCMinKeyPrefix = []byte{0x40}
)

// GetProposalIDBytes returns the byte representation of the proposalID
func GetProposalIDBytes(proposalID uint64) (proposalIDBz []byte) {
	proposalIDBz = make([]byte, 8)
	binary.BigEndian.PutUint64(proposalIDBz, proposalID)
	return
}

// TotalDepositKey of the specific total amount to deposit based on the proposalID from the store
func TotalDepositKey(proposalID uint64) []byte {
	return append(USTCMinKeyPrefix, GetProposalIDBytes(proposalID)...)
}
