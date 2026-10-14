package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	ModuleName   = "ustcstaking"
	StoreKey     = "x_" + ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName

	BondDenom = "uusd"

	PrincipalPoolName = ModuleName
	RewardPoolName    = ModuleName + "_reward_pool"
)

var (
	PositionKeyPrefix      = []byte{0x10}
	OwnerPositionKeyPrefix = []byte{0x11}
	UnbondingKeyPrefix     = []byte{0x12}
	NextPositionIDKey      = []byte{0x13}
	RewardStateKey         = []byte{0x20}
	ReservedRewardsKey     = []byte{0x21}
	ParamsKey              = []byte{0x30}
)

func OwnerPositionPrefix(owner sdk.AccAddress) []byte {
	return address.MustLengthPrefix(owner.Bytes())
}

func OwnerPositionKey(owner sdk.AccAddress, id uint64) []byte {
	key := OwnerPositionPrefix(owner)
	key = append(key, sdk.Uint64ToBigEndian(id)...)
	return key
}

func OwnerFromPositionKey(key []byte) (sdk.AccAddress, uint64, error) {
	if len(key) < 10 || int(key[0])+9 != len(key) || key[0] == 0 {
		return nil, 0, fmt.Errorf("invalid owner position key length %d", len(key))
	}
	return sdk.AccAddress(key[1 : len(key)-8]), sdk.BigEndianToUint64(key[len(key)-8:]), nil
}
