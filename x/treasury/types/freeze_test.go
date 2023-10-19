package types

import (
	"testing"
)

func TestFreezeListContains(t *testing.T) {
	fl := NewFreezeList()
	addr1 := "cosmos10zn3xx8nhvtdynux5tzjer23q2qpg0tzcz8m7t"
	addr2 := "cosmos1njlydj87f05jmzdt9wmam0z28dlrc97q973wzn"

	if fl.Contains(addr1) {
		t.Errorf("Expected Contains() to return false for an empty FreezeList")
	}

	fl.Add(addr1)
	if !fl.Contains(addr1) {
		t.Errorf("Expected Contains() to return true after adding the address")
	}

	if fl.Contains(addr2) {
		t.Errorf("Expected Contains() to return false for an address that hasn't been added")
	}
}

func TestFreezeListAdd(t *testing.T) {
	fl := NewFreezeList()
	addr1 := "cosmos10zn3xx8nhvtdynux5tzjer23q2qpg0tzcz8m7t"

	// Add an address
	fl.Add(addr1)

	if !fl.Contains(addr1) {
		t.Errorf("Expected Add() to add the address to FreezeList")
	}

	// Add the same address again, it should not be duplicated
	fl.Add(addr1)
	if len(fl.Frozen) != 1 {
		t.Errorf("Expected Add() not to duplicate the address in FreezeList")
	}
}

func TestFreezeListRemove(t *testing.T) {
	fl := NewFreezeList()
	addr1 := "cosmos10zn3xx8nhvtdynux5tzjer23q2qpg0tzcz8m7t"
	addr2 := "cosmos1njlydj87f05jmzdt9wmam0z28dlrc97q973wzn"

	// Try to remove an address from an empty FreezeList
	fl.Remove(addr1)
	if len(fl.Frozen) != 0 {
		t.Errorf("Expected Remove() to have no effect on an empty FreezeList")
	}

	// Add an address and then remove it
	fl.Add(addr1)
	fl.Remove(addr1)
	if len(fl.Frozen) != 0 || fl.Contains(addr1) {
		t.Errorf("Expected Remove() to remove the added address from FreezeList")
	}

	// Try to remove an address that was not added
	fl.Remove(addr2)
	if len(fl.Frozen) != 0 {
		t.Errorf("Expected Remove() not to affect FreezeList when removing an address that wasn't added")
	}
}

func TestFreezeListNewFreezeList(t *testing.T) {
	fl := NewFreezeList()

	if len(fl.Frozen) != 0 {
		t.Errorf("Expected NewFreezeList() to create an empty FreezeList")
	}
}
