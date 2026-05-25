// nolint
package formular

import (
	"encoding/json"
	"testing"
)

func TestMenuSnapshotStateAppliesSnapshotChangingMessages(t *testing.T) {
	state := NewMenuSnapshotState()
	readonly := true
	state.Apply(MenuSnapshotMessage{
		MessageBase: MessageBase{Type: MessageMenuSnapshot, MenuID: "settings", MenuGeneration: 1},
		Blocks: []Block{
			{
				ID:         "profile",
				Order:      10,
				Generation: 1,
				Form:       true,
				Items: []Item{
					{Type: ItemField, ID: "name", Label: "Name", Field: &Field{Kind: FieldText, Value: "Ada", Validation: true}},
					{Type: ItemField, ID: "email", Label: "Email", Field: &Field{Kind: FieldText, Value: "ada@example.com"}},
				},
			},
			{
				ID:         "live",
				Order:      20,
				Generation: 1,
				Items: []Item{
					{Type: ItemField, ID: "manual", Label: "Manual", Field: &Field{Kind: FieldText, Value: "old"}},
				},
			},
		},
	})

	if !state.Apply(FieldValidateMessage{
		MessageBase: MessageBase{Type: MessageFieldValidate, MenuID: "settings"},
		Field:       FieldRef{BlockID: "profile", FieldID: "name"},
		Value:       "Grace",
	}) {
		t.Fatal("expected field.validate to update cached candidate value")
	}
	if !state.Apply(FieldStatusMessage{
		MessageBase: MessageBase{Type: MessageFieldStatus, MenuID: "settings"},
		Field:       FieldRef{BlockID: "profile", FieldID: "name"},
		Status:      StatusOK,
		StatusText:  "Looks good",
		Readonly:    &readonly,
	}) {
		t.Fatal("expected field.status to update cached field status")
	}
	if !state.Apply(FormApplyMessage{
		MessageBase: MessageBase{Type: MessageFormApply, MenuID: "settings"},
		BlockID:     "profile",
		Values:      map[string]any{"name": "Grace Hopper", "email": "grace@example.com"},
	}) {
		t.Fatal("expected form.apply to update cached form values")
	}
	if !state.Apply(FieldUpdateMessage{
		MessageBase: MessageBase{Type: MessageFieldUpdate, MenuID: "settings"},
		Field:       FieldRef{BlockID: "live", FieldID: "manual"},
		Value:       "new",
	}) {
		t.Fatal("expected field.update to update cached realtime value")
	}
	if !state.Apply(BlockSnapshotMessage{
		MessageBase: MessageBase{Type: MessageBlockSnapshot, MenuID: "settings", MenuGeneration: 2, BlockGeneration: 1},
		Block: Block{
			ID:         "status",
			Order:      99,
			Generation: 1,
			Items:      []Item{{Type: ItemLabel, ID: "message", Text: "Accepted"}},
		},
	}) {
		t.Fatal("expected block.snapshot to add cached block")
	}
	if !state.Apply(BlockDeleteMessage{
		MessageBase: MessageBase{Type: MessageBlockDelete, MenuID: "settings"},
		BlockID:     "live",
	}) {
		t.Fatal("expected block.delete to remove cached block")
	}

	snapshot, ok := state.ForceSnapshot("settings")
	if !ok {
		t.Fatal("missing cached snapshot")
	}
	if !snapshot.Force {
		t.Fatal("force snapshot should force frontend reinitialization")
	}
	if snapshot.MenuGeneration != 2 {
		t.Fatalf("menu generation = %d, want 2", snapshot.MenuGeneration)
	}
	if len(snapshot.Blocks) != 2 {
		t.Fatalf("block count = %d, want 2", len(snapshot.Blocks))
	}
	name := snapshot.Blocks[0].Items[0].Field
	if name.Value != "Grace Hopper" || name.Status != StatusOK || name.StatusText != "Looks good" || !name.Readonly {
		t.Fatalf("unexpected cached name field: %+v", name)
	}
	email := snapshot.Blocks[0].Items[1].Field
	if email.Value != "grace@example.com" {
		t.Fatalf("email value = %v, want grace@example.com", email.Value)
	}
	if snapshot.Blocks[1].ID != "status" {
		t.Fatalf("second block = %s, want status", snapshot.Blocks[1].ID)
	}
}

func TestMenuSnapshotStateFieldValidateClearsCachedValidationStatus(t *testing.T) {
	state := NewMenuSnapshotState()
	state.Apply(MenuSnapshotMessage{
		MessageBase: MessageBase{Type: MessageMenuSnapshot, MenuID: "settings"},
		Blocks: []Block{{
			ID: "profile",
			Items: []Item{{
				Type:  ItemField,
				ID:    "email",
				Label: "Email",
				Field: &Field{Kind: FieldText, Value: "ok@example.com", Validation: true, Status: StatusOK, StatusText: "Looks good"},
			}},
		}},
	})

	if !state.Apply(FieldValidateMessage{
		MessageBase: MessageBase{Type: MessageFieldValidate, MenuID: "settings"},
		Field:       FieldRef{BlockID: "profile", FieldID: "email"},
		Value:       "invalid",
	}) {
		t.Fatal("expected field.validate to apply")
	}

	snapshot, ok := state.Snapshot("settings")
	if !ok {
		t.Fatal("missing cached snapshot")
	}
	field := snapshot.Blocks[0].Items[0].Field
	if field.Value != "invalid" || field.Status != StatusUnset || field.StatusText != "" {
		t.Fatalf("unexpected cached validation field: %+v", field)
	}
}

func TestMenuSnapshotStateAppliesNestedArrayValuesFromJSON(t *testing.T) {
	state := NewMenuSnapshotState()
	state.Apply(MenuSnapshotMessage{
		MessageBase: MessageBase{Type: MessageMenuSnapshot, MenuID: "settings"},
		Blocks: []Block{{
			ID:    "servers",
			Order: 1,
			Items: []Item{{
				Type:  ItemField,
				ID:    "items",
				Label: "Servers",
				Field: &Field{
					Kind: FieldArray,
					Templates: []ArrayTemplate{{
						Name: "database",
						Items: []Item{
							{Type: ItemField, ID: "dsn", Label: "DSN", Field: &Field{Kind: FieldText, Value: "postgres://localhost/app"}},
							{Type: ItemField, ID: "pool", Label: "Pool", Field: &Field{Kind: FieldInt, Value: 10}},
						},
					}},
				},
			}},
		}},
	})

	raw := []byte(`{
		"type":"field.update",
		"menuId":"settings",
		"field":{"blockId":"servers","fieldId":"items"},
		"value":[{"id":"local-1","template":"database","values":{"dsn":"mysql://localhost/demo","pool":24}}]
	}`)
	applied, err := state.ApplyJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Fatal("expected JSON field.update to apply")
	}

	snapshot, ok := state.Snapshot("settings")
	if !ok {
		t.Fatal("missing cached snapshot")
	}
	array := snapshot.Blocks[0].Items[0].Field
	if len(array.Elements) != 1 {
		t.Fatalf("element count = %d, want 1", len(array.Elements))
	}
	element := array.Elements[0]
	if element.ID != "local-1" || element.Template != "database" {
		t.Fatalf("unexpected element: %+v", element)
	}
	if got := element.Items[0].Field.Value; got != "mysql://localhost/demo" {
		t.Fatalf("dsn = %v, want mysql://localhost/demo", got)
	}
	if got := element.Items[1].Field.Value; got != float64(24) {
		t.Fatalf("pool = %T(%v), want float64(24)", got, got)
	}
}

func TestMenuSnapshotStateIgnoresEphemeralMessages(t *testing.T) {
	state := NewMenuSnapshotState()
	data, err := json.Marshal(AutocompleteHintsMessage{
		MessageBase: MessageBase{Type: MessageAutocompleteHints, MenuID: "settings"},
		Field:       FieldRef{BlockID: "profile", FieldID: "timezone"},
		Prefix:      "Europe/T",
		Hints:       []string{"Europe/Tbilisi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	applied, err := state.ApplyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("autocomplete hints should be ignored")
	}
	if snapshots := state.Snapshots(); len(snapshots) != 0 {
		t.Fatalf("snapshot count = %d, want 0", len(snapshots))
	}
}
