package types

import (
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type LegacyParams struct {
	// unbonding_time is the time duration of unbonding.
	UnbondingTime time.Duration `protobuf:"bytes,1,opt,name=unbonding_time,json=unbondingTime,proto3,stdduration" json:"unbonding_time"`
	// max_validators is the maximum number of validators.
	MaxValidators uint32 `protobuf:"varint,2,opt,name=max_validators,json=maxValidators,proto3" json:"max_validators,omitempty"`
	// max_entries is the max entries for either unbonding delegation or redelegation (per pair/trio).
	MaxEntries uint32 `protobuf:"varint,3,opt,name=max_entries,json=maxEntries,proto3" json:"max_entries,omitempty"`
	// historical_entries is the number of historical entries to persist.
	HistoricalEntries uint32 `protobuf:"varint,4,opt,name=historical_entries,json=historicalEntries,proto3" json:"historical_entries,omitempty"`
	// bond_denom defines the bondable coin denomination.
	BondDenom string `protobuf:"bytes,5,opt,name=bond_denom,json=bondDenom,proto3" json:"bond_denom,omitempty"`
}

// ParamKeyTable returns the parameter key table for wasm module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&LegacyParams{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
func (p *LegacyParams) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte("UnbondingTime"), &p.UnbondingTime, func(i interface{}) error { return nil }),
		paramtypes.NewParamSetPair([]byte("MaxValidators"), &p.MaxValidators, func(i interface{}) error { return nil }),
		paramtypes.NewParamSetPair([]byte("MaxEntries"), &p.MaxEntries, func(i interface{}) error { return nil }),
		paramtypes.NewParamSetPair([]byte("HistoricalEntries"), &p.HistoricalEntries, func(i interface{}) error { return nil }),
		paramtypes.NewParamSetPair([]byte("BondDenom"), &p.BondDenom, func(i interface{}) error { return nil }),
	}
}
