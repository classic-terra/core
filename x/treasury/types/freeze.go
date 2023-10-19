package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type FreezeList struct {
	Frozen []string `json:"frozen"`
}

func NewFreezeList() FreezeList {
	return FreezeList{
		Frozen: []string{},
	}
}

func (fl *FreezeList) Contains(target string) bool {
	return sdk.SliceContains(fl.Frozen, target)
}

func (fl *FreezeList) Add(target string) {
	if fl.Contains(target) {
		return
	}
	fl.Frozen = append(fl.Frozen, target)

}

func (fl *FreezeList) Remove(target string) {
	if !fl.Contains(target) {
		return
	}

	var updated []string
	for _, item := range fl.Frozen {
		if item != target {
			updated = append(updated, item)
		}
	}
	fl.Frozen = updated

}
