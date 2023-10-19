package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type FreezeList struct {
	Frozen []sdk.Address `json:"frozen"`
}

func NewFreezeList() FreezeList {
	return FreezeList{
		Frozen: []sdk.Address{},
	}
}

func (fl FreezeList) Contains(target sdk.Address) bool {
	return sdk.SliceContains(fl.Frozen, target)
}

func (fl FreezeList) Add(target sdk.Address) {
	if fl.Contains(target) {
		return
	}
	fl.Frozen = append(fl.Frozen, target)
}

func (fl FreezeList) Remove(target sdk.Address) {
	if !fl.Contains(target) {
		return
	}

	var updated []sdk.Address
	for _, item := range fl.Frozen {
		if item != target {
			updated = append(updated, item)
		}
	}
	fl.Frozen = updated

}
