package v13

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

// GetContractByCreatedSecondaryIndexKey returns the key for the contract's created secondary index.
// It is classic-terra forked version https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L64
func GetContractByCreatedSecondaryIndexKeyLegacy(contractAddr sdk.AccAddress, c wasmtypes.ContractCodeHistoryEntry) []byte {
	prefix := GetContractByCodeIDSecondaryIndexPrefixLegacy(c.CodeID)
	prefixLen := len(prefix)
	contractAddrLen := len(contractAddr)
	r := make([]byte, prefixLen+LegacyPrefixes.AbsoluteTxPositionLen+contractAddrLen)
	copy(r[0:], prefix)
	copy(r[prefixLen:], c.Updated.Bytes())
	copy(r[prefixLen+LegacyPrefixes.AbsoluteTxPositionLen:], contractAddr)
	return r
}

// GetContractByCodeIDSecondaryIndexPrefix returns the prefix for the second index: `<prefix><codeID>`
// https://github.com/classic-terra/wasmd/blob/release/v0.46.x-classic/x/wasm/types/keys.go#L75C1-L83C2
func GetContractByCodeIDSecondaryIndexPrefixLegacy(codeID uint64) []byte {
	prefixLen := len(LegacyPrefixes.ContractByCodeIDAndCreatedSecondaryIndexPrefix)
	const codeIDLen = 8
	r := make([]byte, prefixLen+codeIDLen)
	copy(r[0:], LegacyPrefixes.ContractByCodeIDAndCreatedSecondaryIndexPrefix)
	copy(r[prefixLen:], sdk.Uint64ToBigEndian(codeID))
	return r
}

// GetContractsByCreatorPrefix returns the contracts by creator prefix for the WASM contract instance
func GetContractsByCreatorPrefixLegacy(addr sdk.AccAddress) []byte {
	bz := address.MustLengthPrefix(addr)
	return append(LegacyPrefixes.ContractsByCreatorPrefix, bz...)
}

// GetPinnedCodeIndexPrefix returns the key prefix for a code id pinned into the wasmvm cache
func GetPinnedCodeIndexPrefixLegacy(codeID uint64) []byte {
	prefixLen := len(LegacyPrefixes.PinnedCodeIndexPrefix)
	r := make([]byte, prefixLen+8)
	copy(r[0:], LegacyPrefixes.PinnedCodeIndexPrefix)
	copy(r[prefixLen:], sdk.Uint64ToBigEndian(codeID))
	return r
}

// GetCodeKeyLegacy returns the key for the WASM contract instance
func GetCodeKeyLegacy(addr sdk.AccAddress) []byte {
	return append(LegacyPrefixes.CodeKeyPrefix, address.MustLengthPrefix(addr)...)
}

// GetContractAddressKeyLegacy returns the key for the WASM contract store
func GetContractAddressKeyLegacy(addr sdk.AccAddress) []byte {
	return append(LegacyPrefixes.ContractKeyPrefix, address.MustLengthPrefix(addr)...)
}

// GetContractStorePrefixLegacy returns the store prefix for the WASM contract instance
func GetContractStorePrefixLegacy(addr sdk.AccAddress) []byte {
	return append(LegacyPrefixes.ContractStorePrefix, address.MustLengthPrefix(addr)...)
}

// GetContractCodeHistoryElementKey returns the key a contract code history entry: `<prefix><contractAddr><position>`
func GetContractCodeHistoryElementKeyLegacy(contractAddr sdk.AccAddress, pos uint64) []byte {
	prefix := GetContractCodeHistoryElementPrefixLegacy(contractAddr)
	prefixLen := len(prefix)
	r := make([]byte, prefixLen+8)
	copy(r[0:], prefix)
	copy(r[prefixLen:], sdk.Uint64ToBigEndian(pos))
	return r
}

// GetContractCodeHistoryElementPrefix returns the key prefix for a contract code history entry: `<prefix><contractAddr>`
func GetContractCodeHistoryElementPrefixLegacy(contractAddr sdk.AccAddress) []byte {
	prefixLen := len(LegacyPrefixes.ContractCodeHistoryElementPrefix)
	contractAddrLen := len(contractAddr)
	r := make([]byte, prefixLen+contractAddrLen)
	copy(r[0:], LegacyPrefixes.ContractCodeHistoryElementPrefix)
	copy(r[prefixLen:], contractAddr)
	return r
}
