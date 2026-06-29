// nolint
package formular

import (
	"errors"
	"testing"
)

func TestMessageConstructorsPopulateEnvelopeAndCopyInputs(t *testing.T) {
	block := Block{
		ID:    "profile",
		Items: []Item{TextField("name", "Name", "Ada")},
	}
	snapshot := MenuSnapshot("settings", 7, block)
	block.Items[0].Field.Value = "changed"

	if snapshot.Type != MessageMenuSnapshot || snapshot.MenuID != "settings" || snapshot.MenuGeneration != 7 {
		t.Fatalf("unexpected menu snapshot envelope: %+v", snapshot.MessageBase)
	}
	if snapshot.Blocks[0].Items[0].Field.Value != "Ada" {
		t.Fatal("menu snapshot shared caller block state")
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatal(err)
	}

	forced := ForcedMenuSnapshot("settings", 8, snapshot.Blocks...)
	if !forced.Force {
		t.Fatal("forced snapshot should set force")
	}

	blockMsg := BlockSnapshot("settings", 9, 10, block)
	if blockMsg.Type != MessageBlockSnapshot || blockMsg.MenuGeneration != 9 || blockMsg.BlockGeneration != 10 {
		t.Fatalf("unexpected block snapshot envelope: %+v", blockMsg.MessageBase)
	}
	if blockMsg.Block.Items[0].Field.Value != "changed" {
		t.Fatal("block snapshot should copy the current block value")
	}
}

func TestFieldStatusFromError(t *testing.T) {
	ref := FieldRef{
		BlockID: "profile",
		FieldID: "email",
		ElementPath: []ElementPathSegment{
			{ArrayFieldID: "contacts", ElementID: "primary"},
		},
	}

	ok := FieldStatusFromError("settings", 1, 2, ref, nil)
	ref.ElementPath[0].ElementID = "changed"
	if ok.Type != MessageFieldStatus || ok.Status != StatusOK || ok.StatusText != "" {
		t.Fatalf("unexpected ok status: %+v", ok)
	}
	if ok.Field.ElementPath[0].ElementID != "primary" {
		t.Fatal("field status shared field ref state")
	}

	errMsg := FieldStatusFromError("settings", 1, 2, ref, errors.New("invalid email"))
	if errMsg.Status != StatusError || errMsg.StatusText != "invalid email" {
		t.Fatalf("unexpected error status: %+v", errMsg)
	}
}

func TestAutocompleteHintsConstructorCopiesHints(t *testing.T) {
	hints := []string{"Europe/Tbilisi"}
	msg := AutocompleteHints("settings", 4, FieldRef{BlockID: "profile", FieldID: "timezone"}, "Europe/T", hints)
	hints[0] = "changed"

	if msg.Type != MessageAutocompleteHints || msg.MenuGeneration != 4 || msg.Prefix != "Europe/T" {
		t.Fatalf("unexpected autocomplete hints envelope: %+v", msg)
	}
	if msg.Hints[0] != "Europe/Tbilisi" {
		t.Fatal("autocomplete hints shared caller slice")
	}
	if err := msg.Validate(); err != nil {
		t.Fatal(err)
	}
}
