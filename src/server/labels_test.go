package server

import (
	"context"
	"testing"

	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

func TestBuildListEdit(t *testing.T) {
	action := func(existing *storage.Label) *waSyncAction.LabelEditAction {
		return buildListEdit("13", "Leads", 2, false, existing).Mutations[0].Value.LabelEditAction
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

	// Delete keeps the list type, only flips deleted
	d := buildListEdit("13", "Leads", 2, true, &storage.Label{Type: int32(waSyncAction.LabelEditAction_CUSTOM)}).Mutations[0].Value.LabelEditAction
	if !d.GetDeleted() || d.GetType() != waSyncAction.LabelEditAction_CUSTOM {
		t.Fatalf("delete: got deleted=%v type=%v", d.GetDeleted(), d.GetType())
	}
}

type fakeLIDs struct{ store.LIDStore }

var testPN = types.NewJID("972500000000", types.DefaultUserServer)
var testLID = types.NewJID("100000000000001", types.HiddenUserServer)

func (fakeLIDs) GetLIDForPN(_ context.Context, pn types.JID) (types.JID, error) {
	if pn == testPN {
		return testLID, nil
	}
	return types.EmptyJID, nil
}

func (fakeLIDs) GetPNForLID(_ context.Context, lid types.JID) (types.JID, error) {
	if lid == testLID {
		return testPN, nil
	}
	return types.EmptyJID, nil
}

func TestChatJIDs(t *testing.T) {
	ctx := context.Background()
	if got := chatJIDs(ctx, fakeLIDs{}, testPN); len(got) != 2 || got[1] != testLID {
		t.Fatalf("pn: got %v", got)
	}
	if got := chatJIDs(ctx, fakeLIDs{}, testLID); len(got) != 2 || got[1] != testPN {
		t.Fatalf("lid: got %v", got)
	}
	group := types.NewJID("123", types.GroupServer)
	if got := chatJIDs(ctx, fakeLIDs{}, group); len(got) != 1 {
		t.Fatalf("group: got %v", got)
	}
	if got := chatJIDs(ctx, nil, testPN); len(got) != 1 {
		t.Fatalf("no store: got %v", got)
	}
}
