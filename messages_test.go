// nolint
package formular

import "testing"

func TestMenuSnapshotMessageCopyDoesNotShareNestedState(t *testing.T) {
	readonly := true
	min := 1.0
	progress := uint(10)
	original := MenuSnapshotMessage{
		MessageBase: MessageBase{
			Type:           MessageMenuSnapshot,
			MenuID:         "settings",
			MenuGeneration: 1,
		},
		Blocks: []Block{
			{
				ID:         "account",
				Order:      10,
				Generation: 2,
				Form:       true,
				Copyable:   &Copyable{Text: "copy"},
				Items: []Item{
					{Type: ItemProgressbar, ID: "sync", Label: "Sync", Progress: &progress},
					{Type: ItemLogs, ID: "logs", Label: "Logs", Logs: []LogLine{{Level: LogInfo, Text: "ready"}}},
					{
						Type:  ItemField,
						ID:    "email",
						Label: "Email",
						Field: &Field{
							Kind:          FieldText,
							Value:         map[string]any{"nested": []any{"value"}},
							Autocomplete:  &Autocomplete{Enabled: true, Tag: "email"},
							AllowedValues: []any{map[string]any{"name": "user@example.com"}},
							Min:           &min,
							Copyable:      &Copyable{Text: "field copy"},
							Templates: []ArrayTemplate{
								{
									Name: "credential",
									Items: []Item{
										{Type: ItemButton, ID: "rotate", Label: "Rotate"},
									},
								},
							},
							Elements: []ArrayElement{
								{
									ID:       "cred-1",
									Template: "credential",
									Copyable: &Copyable{Text: "secret"},
									Items: []Item{
										{
											Type:  ItemField,
											ID:    "token",
											Label: "Token",
											Field: &Field{
												Kind:     FieldText,
												Readonly: readonly,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	copied := original.Copy()

	copied.Blocks[0].Copyable.Text = "changed"
	*copied.Blocks[0].Items[0].Progress = 20
	copied.Blocks[0].Items[1].Logs[0].Text = "changed"
	copied.Blocks[0].Items[2].Field.Autocomplete.Tag = "changed"
	copied.Blocks[0].Items[2].Field.Min = nil
	copied.Blocks[0].Items[2].Field.Value.(map[string]any)["nested"].([]any)[0] = "changed"
	copied.Blocks[0].Items[2].Field.AllowedValues[0].(map[string]any)["name"] = "changed"
	copied.Blocks[0].Items[2].Field.Copyable.Text = "changed"
	copied.Blocks[0].Items[2].Field.Templates[0].Items[0].Label = "changed"
	copied.Blocks[0].Items[2].Field.Elements[0].Copyable.Text = "changed"
	copied.Blocks[0].Items[2].Field.Elements[0].Items[0].Field.Readonly = false

	if original.Blocks[0].Copyable.Text != "copy" {
		t.Fatal("block copyable was shared")
	}
	if original.Blocks[0].Items[0].Progress == nil || *original.Blocks[0].Items[0].Progress != 10 {
		t.Fatal("progress pointer was shared")
	}
	if original.Blocks[0].Items[1].Logs[0].Text != "ready" {
		t.Fatal("logs were shared")
	}
	if original.Blocks[0].Items[2].Field.Autocomplete.Tag != "email" {
		t.Fatal("autocomplete pointer was shared")
	}
	if original.Blocks[0].Items[2].Field.Min == nil || *original.Blocks[0].Items[2].Field.Min != 1 {
		t.Fatal("numeric pointer was shared")
	}
	if got := original.Blocks[0].Items[2].Field.Value.(map[string]any)["nested"].([]any)[0]; got != "value" {
		t.Fatal("field value was shared")
	}
	if got := original.Blocks[0].Items[2].Field.AllowedValues[0].(map[string]any)["name"]; got != "user@example.com" {
		t.Fatal("allowed values were shared")
	}
	if original.Blocks[0].Items[2].Field.Copyable.Text != "field copy" {
		t.Fatal("field copyable was shared")
	}
	if original.Blocks[0].Items[2].Field.Templates[0].Items[0].Label != "Rotate" {
		t.Fatal("array template items were shared")
	}
	if original.Blocks[0].Items[2].Field.Elements[0].Copyable.Text != "secret" {
		t.Fatal("array element copyable was shared")
	}
	if !original.Blocks[0].Items[2].Field.Elements[0].Items[0].Field.Readonly {
		t.Fatal("array element field was shared")
	}
}

func TestFrontendMessageCopies(t *testing.T) {
	readonly := true
	status := FieldStatusMessage{
		MessageBase: MessageBase{Type: MessageFieldStatus, MenuID: "settings"},
		Field: FieldRef{
			BlockID: "account",
			FieldID: "token",
			ElementPath: []ElementPathSegment{
				{ArrayFieldID: "credentials", ElementID: "cred-1"},
			},
		},
		Status:   StatusOK,
		Readonly: &readonly,
	}

	copiedStatus := status.Copy()
	copiedStatus.Readonly = nil
	copiedStatus.Field.ElementPath[0].ElementID = "changed"

	if status.Readonly == nil || !*status.Readonly {
		t.Fatal("readonly pointer was shared")
	}
	if status.Field.ElementPath[0].ElementID != "cred-1" {
		t.Fatal("field ref element path was shared")
	}

	apply := FormApplyMessage{
		MessageBase: MessageBase{Type: MessageFormApply, MenuID: "settings"},
		BlockID:     "account",
		Values: map[string]any{
			"credentials": []ArrayElementValue{
				{
					ID:       "cred-1",
					Template: "credential",
					Values:   map[string]any{"token": "secret"},
				},
			},
		},
	}

	copiedApply := apply.Copy()
	copiedApply.Values["credentials"].([]ArrayElementValue)[0].Values["token"] = "changed"

	if got := apply.Values["credentials"].([]ArrayElementValue)[0].Values["token"]; got != "secret" {
		t.Fatal("form apply values were shared")
	}
}
