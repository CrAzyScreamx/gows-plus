package server

import (
	"testing"

	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/proto/waSyncAction"
)

func TestBuildListEdit(t *testing.T) {
	action := func(existing *storage.Label) *waSyncAction.LabelEditAction {
		return buildListEdit("13", "Leads", 2, existing).Mutations[0].Value.LabelEditAction
	}

	// New list: CUSTOM + active, like WA Web's labelAddAction
	a := action(nil)
	if a.GetType() != waSyncAction.LabelEditAction_CUSTOM || !a.GetIsActive() || a.GetDeleted() {
		t.Fatalf("new list: got type=%v isActive=%v deleted=%v", a.GetType(), a.GetIsActive(), a.GetDeleted())
	}
	if a.PredefinedID != nil {
		t.Fatalf("new list: predefinedId should be unset, got %v", a.GetPredefinedID())
	}

	// Editing a disabled built-in list keeps its type/isActive/predefinedId
	inactive := false
	a = action(&storage.Label{Type: int32(waSyncAction.LabelEditAction_FAVORITES), IsActive: &inactive, PredefinedID: 7})
	if a.GetType() != waSyncAction.LabelEditAction_FAVORITES || a.GetIsActive() || a.GetPredefinedID() != 7 {
		t.Fatalf("existing list: got type=%v isActive=%v predefinedId=%v", a.GetType(), a.GetIsActive(), a.GetPredefinedID())
	}

	// Label stored before this fix (no type) is upgraded to a CUSTOM list
	a = action(&storage.Label{})
	if a.GetType() != waSyncAction.LabelEditAction_CUSTOM || !a.GetIsActive() {
		t.Fatalf("legacy label: got type=%v isActive=%v", a.GetType(), a.GetIsActive())
	}
}
