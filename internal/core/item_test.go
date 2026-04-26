package core

import (
	"testing"

	"github.com/b1tAction/paradiced/pkg/constants"
	"github.com/b1tAction/paradiced/pkg/id"
)

func TestNewItem(t *testing.T) {
	item := NewItem(constants.ItemTypeAnyDoor)
	if item.Type != constants.ItemTypeAnyDoor {
		t.Errorf("Item.Type = %s, expected any_door", item.Type)
	}
	if item.ID.UUID() == "" {
		t.Error("Item.ID should have a valid UUID")
	}
	if !item.Usable {
		t.Error("NewItem should set Usable=true by default")
	}
}

func TestNewItemWithID(t *testing.T) {
	specificID := id.NewItemID()
	item := NewItemWithID(constants.ItemTypeReverseClock, specificID)
	if item.Type != constants.ItemTypeReverseClock {
		t.Errorf("Item.Type = %s, expected reverse_clock", item.Type)
	}
	if item.ID != specificID {
		t.Errorf("Item.ID = %v, expected %v (specific ID should be preserved)", item.ID, specificID)
	}
	if !item.Usable {
		t.Error("NewItemWithID should set Usable=true by default")
	}
}